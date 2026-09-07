package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/Mr-Robot-err-404/heckl/internal/github"
	"github.com/Mr-Robot-err-404/heckl/internal/reviewer"
	"github.com/Mr-Robot-err-404/heckl/internal/store"
)

var ErrNotRunning = errors.New("no review in flight")

type activeRun struct {
	review    *Review
	cancel    context.CancelFunc
	whole     bool
	cancelled map[string]bool
}

func (a *activeRun) isCancelled(agent string) bool {
	return a.whole || a.cancelled[agent]
}

type Runner struct {
	ctx      context.Context
	logger   *slog.Logger
	source   PRSource
	reviewer *reviewer.Reviewer
	store    ReviewStore
	publish  func(key string, review *Review)
	path     SessionPath

	mu       sync.RWMutex
	inFlight map[string]*activeRun
}

func newRunner(ctx context.Context, logger *slog.Logger, source PRSource, rev *reviewer.Reviewer, st ReviewStore, publish func(string, *Review), path SessionPath) *Runner {
	return &Runner{
		ctx:      ctx,
		logger:   logger,
		source:   source,
		reviewer: rev,
		store:    st,
		publish:  publish,
		path:     path,
		inFlight: make(map[string]*activeRun),
	}
}

func (r *Runner) Active() []*Review {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Review, 0, len(r.inFlight))
	for _, active := range r.inFlight {
		out = append(out, active.review.clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	return out
}

func (r *Runner) InFlight(key string) *Review {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if active, ok := r.inFlight[key]; ok {
		return active.review.clone()
	}
	return nil
}

type StartRequest struct {
	Owner    string
	Repo     string
	PRNumber int
	Agents   []string
	Base     *Review
}

func (r *Runner) Start(req StartRequest) (*Review, error) {
	key := PRKey(req.Owner, req.Repo, req.PRNumber)
	agents := normaliseAgents(req.Agents)

	r.mu.Lock()
	if existing, running := r.inFlight[key]; running {
		snapshot := existing.review.clone()
		r.mu.Unlock()
		return snapshot, nil
	}

	id, err := newID()
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}

	review := newReview(id, req.Owner, req.Repo, req.PRNumber, agents)
	if req.Base != nil {
		review.seedFrom(req.Base, agents)
	}
	review.Status = StatusRunning
	review.startStage("", StageFetch)

	runCtx, cancel := context.WithCancel(r.ctx)
	r.inFlight[key] = &activeRun{review: review, cancel: cancel, cancelled: map[string]bool{}}
	snapshot := review.clone()
	sessionID := review.SessionID
	r.mu.Unlock()

	r.publish(key, snapshot)
	go r.run(runCtx, key, req.Owner, req.Repo, req.PRNumber, agents, sessionID)

	return snapshot, nil
}

func (r *Runner) Cancel(key, agent string) (*Review, error) {
	r.mu.Lock()
	active, ok := r.inFlight[key]
	if !ok {
		r.mu.Unlock()
		return nil, ErrNotRunning
	}
	if agent != "" && active.review.agent(agent) == nil {
		r.mu.Unlock()
		return nil, fmt.Errorf("agent %q is not part of this review", agent)
	}

	active.whole = active.whole || agent == ""

	var sessions []string
	for i := range active.review.Agents {
		a := &active.review.Agents[i]
		if agent != "" && a.Name != agent {
			continue
		}
		active.cancelled[a.Name] = true
		if a.OpencodeSessionID != "" && (a.Status == StatusRunning || a.Status == StatusPending) {
			sessions = append(sessions, a.OpencodeSessionID)
		}
	}
	snapshot := active.review.clone()
	stopCheckout := active.cancel
	r.mu.Unlock()

	for _, id := range sessions {
		r.abort(id)
	}
	if agent == "" {
		stopCheckout()
	}
	return snapshot, nil
}

func (r *Runner) abort(opencodeSessionID string) {
	if err := r.reviewer.Abort(opencodeSessionID); err != nil {
		r.logger.Warn("orchestrator: abort opencode session", "session_id", opencodeSessionID, "err", err)
	}
}

func (r *Runner) cancelledFor(key string) func(string) bool {
	return func(agent string) bool {
		r.mu.RLock()
		defer r.mu.RUnlock()
		active, ok := r.inFlight[key]
		return ok && active.isCancelled(agent)
	}
}

func normaliseAgents(selected []string) []string {
	allowed := reviewer.AgentOrder()
	out := make([]string, 0, len(allowed))
	for _, name := range allowed {
		for _, want := range selected {
			if want == name {
				out = append(out, name)
				break
			}
		}
	}
	if len(out) == 0 {
		return []string{reviewer.AgentReviewer}
	}
	return out
}

func (r *Runner) run(ctx context.Context, key, owner, repo string, prNumber int, agents []string, sessionID int64) {
	log := r.logger.With("owner", owner, "repo", repo, "pr", prNumber)
	defer r.release(key)

	t := time.Now()
	pr, diff, err := r.fetch(owner, repo, prNumber)
	if err != nil {
		r.settle(key, sessionID, StageFetch, cancelCause(ctx, err), log)
		return
	}

	headSHA := pr.HeadSHA()
	r.update(key, func(rv *Review) {
		rv.HeadSHA = headSHA
		rv.endStage("", StageFetch, diffSize(len(diff)), nil)
	})
	log.Info("orchestrator: pr fetched", "head_sha", headSHA, "diff_bytes", len(diff), "duration_ms", time.Since(t).Milliseconds())

	r.update(key, func(rv *Review) { rv.startStage("", StageCheckout) })
	worktree, err := r.reviewer.Checkout().Worktree(ctx, owner, repo, prNumber, headSHA)
	if err != nil {
		r.settle(key, sessionID, StageCheckout, cancelCause(ctx, err), log)
		return
	}
	r.update(key, func(rv *Review) { rv.endStage("", StageCheckout, shortSHA(headSHA), nil) })

	out, err := r.reviewer.Review(r.ctx, reviewer.ReviewRequest{
		Owner:        owner,
		Repo:         repo,
		PRNumber:     prNumber,
		HeadSHA:      headSHA,
		Title:        pr.Title,
		Body:         pr.Body,
		Diff:         string(diff),
		CheckoutPath: worktree,
		Agents:       agents,
		SessionID:    sessionID,
		StartedAt:    t,
		Progress:     r.progressFor(key),
		Cancelled:    r.cancelledFor(key),
	})
	if err != nil {
		r.settle(key, sessionID, "", err, log)
		return
	}

	r.update(key, func(rv *Review) {
		rv.SessionID = out.Session.ID
		rv.Summary = out.Session.Summary
		rv.Concerns = toConcerns(out.Concerns)
		rv.mergeStoredAgents(out.Agents, r.path)
		rv.finish(nil)
	})
	log.Info("orchestrator: review complete", "concerns", len(out.Concerns), "duration_ms", time.Since(t).Milliseconds())
}

func cancelCause(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return reviewer.ErrCancelled
	}
	return err
}

func (r *Runner) settle(key string, sessionID int64, stage string, cause error, log *slog.Logger) {
	r.update(key, func(rv *Review) {
		if stage != "" {
			rv.endStage("", stage, "", cause)
		}
		rv.finish(cause)
	})

	if errors.Is(cause, reviewer.ErrCancelled) {
		if sessionID != 0 {
			r.restore(key, sessionID)
		}
		log.Info("orchestrator: review cancelled", "stage", stage)
		return
	}

	r.persistFailure(key, cause)
	log.Error("orchestrator: review failed", "stage", stage, "err", cause)
}

func (r *Runner) restore(key string, sessionID int64) {
	if r.store == nil {
		return
	}
	concerns, err := r.store.ListConcerns(r.ctx, sessionID)
	if err != nil {
		r.logger.Error("orchestrator: restore concerns", "pr", key, "session_id", sessionID, "err", err)
		return
	}
	agents, err := r.store.ListSessionAgents(r.ctx, sessionID)
	if err != nil {
		r.logger.Error("orchestrator: restore agents", "pr", key, "session_id", sessionID, "err", err)
		return
	}
	r.update(key, func(rv *Review) {
		rv.Concerns = toConcerns(concerns)
		for i := range rv.Agents {
			rv.Agents[i].Stages = []Stage{}
		}
		rv.mergeStoredAgents(agents, r.path)
		rv.Status = StatusDone
	})
}

func (r *Runner) fetch(owner, repo string, prNumber int) (*github.PR, []byte, error) {
	pr, err := r.source.GetPR(owner, repo, prNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch pr: %w", err)
	}
	diff, err := r.source.GetPRDiff(owner, repo, prNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch diff: %w", err)
	}
	return pr, diff, nil
}

func (r *Runner) progressFor(key string) reviewer.Progress {
	return func(ev reviewer.ProgressEvent) {
		abort := ""
		r.mutate(key, func(active *activeRun) {
			rv := active.review
			if ev.SessionID != "" {
				rv.setAgentSession(ev.Agent, ev.SessionID, r.path(ev.SessionID))
				if rv.OpencodeSessionID == "" {
					rv.OpencodeSessionID = ev.SessionID
					rv.OpencodeSessionPath = r.path(ev.SessionID)
				}
				if active.isCancelled(ev.Agent) {
					abort = ev.SessionID
				}
			}
			if ev.Done {
				rv.endStage(ev.Agent, ev.Stage, ev.Detail, ev.Err)
				return
			}
			rv.startStage(ev.Agent, ev.Stage)
		})
		if abort != "" {
			r.abort(abort)
		}
	}
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func (r *Runner) update(key string, mutate func(*Review)) {
	r.mutate(key, func(active *activeRun) { mutate(active.review) })
}

func (r *Runner) mutate(key string, apply func(*activeRun)) {
	r.mu.Lock()
	active, ok := r.inFlight[key]
	if !ok {
		r.mu.Unlock()
		return
	}
	apply(active)
	snapshot := active.review.clone()
	r.mu.Unlock()

	r.publish(key, snapshot)
}

func (r *Runner) persistFailure(key string, cause error) {
	if r.store == nil {
		return
	}
	review := r.InFlight(key)
	if review == nil {
		return
	}
	if review.SessionID != 0 {
		r.logger.Warn("orchestrator: re-run failed, keeping stored review", "pr", key, "session_id", review.SessionID, "err", cause)
		return
	}
	_, err := r.store.CreateReviewSession(r.ctx, store.NewReviewSession{
		Owner:             review.Owner,
		Repo:              review.Repo,
		PRNumber:          review.PRNumber,
		HeadSHA:           review.HeadSHA,
		OpencodeSessionID: review.OpencodeSessionID,
		Status:            store.ReviewStatusError,
		Error:             cause.Error(),
		DurationMS:        time.Since(review.StartedAt).Milliseconds(),
	})
	if err != nil {
		r.logger.Error("orchestrator: persist failed review", "pr", key, "err", err)
	}
}

func (r *Runner) release(key string) {
	r.mu.Lock()
	if active, ok := r.inFlight[key]; ok {
		active.cancel()
		delete(r.inFlight, key)
	}
	r.mu.Unlock()
}
