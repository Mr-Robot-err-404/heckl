package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/harrylawton/pr-review/internal/github"
	"github.com/harrylawton/pr-review/internal/reviewer"
	"github.com/harrylawton/pr-review/internal/store"
)

type Runner struct {
	ctx      context.Context
	logger   *slog.Logger
	source   PRSource
	reviewer *reviewer.Reviewer
	store    ReviewStore
	publish  func(key string, review *Review)
	path     SessionPath

	mu       sync.RWMutex
	inFlight map[string]*Review
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
		inFlight: make(map[string]*Review),
	}
}

func (r *Runner) Active() []*Review {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Review, 0, len(r.inFlight))
	for _, review := range r.inFlight {
		out = append(out, review.clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	return out
}

func (r *Runner) InFlight(key string) *Review {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if review, ok := r.inFlight[key]; ok {
		return review.clone()
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
		snapshot := existing.clone()
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
	r.inFlight[key] = review
	snapshot := review.clone()
	sessionID := review.SessionID
	r.mu.Unlock()

	r.publish(key, snapshot)
	go r.run(key, req.Owner, req.Repo, req.PRNumber, agents, sessionID)

	return snapshot, nil
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

func (r *Runner) run(key, owner, repo string, prNumber int, agents []string, sessionID int64) {
	log := r.logger.With("owner", owner, "repo", repo, "pr", prNumber)
	defer r.release(key)

	t := time.Now()
	pr, diff, err := r.fetch(owner, repo, prNumber)
	if err != nil {
		r.update(key, func(rv *Review) {
			rv.endStage("", StageFetch, "", err)
			rv.finish(err)
		})
		r.persistFailure(key, err)
		log.Error("orchestrator: fetch failed", "err", err)
		return
	}

	headSHA := pr.HeadSHA()
	r.update(key, func(rv *Review) {
		rv.HeadSHA = headSHA
		rv.endStage("", StageFetch, diffSize(len(diff)), nil)
	})
	log.Info("orchestrator: pr fetched", "head_sha", headSHA, "diff_bytes", len(diff), "duration_ms", time.Since(t).Milliseconds())

	r.update(key, func(rv *Review) { rv.startStage("", StageCheckout) })
	handle, err := r.reviewer.Checkout().Acquire(r.ctx, owner, repo, prNumber, headSHA)
	if err != nil {
		r.update(key, func(rv *Review) {
			rv.endStage("", StageCheckout, "", err)
			rv.finish(err)
		})
		r.persistFailure(key, err)
		log.Error("orchestrator: checkout failed", "err", err)
		return
	}
	defer handle.Release()
	r.update(key, func(rv *Review) { rv.endStage("", StageCheckout, shortSHA(headSHA), nil) })

	out, err := r.reviewer.Review(r.ctx, reviewer.ReviewRequest{
		Owner:        owner,
		Repo:         repo,
		PRNumber:     prNumber,
		HeadSHA:      headSHA,
		Title:        pr.Title,
		Body:         pr.Body,
		Diff:         string(diff),
		CheckoutPath: handle.Path,
		Agents:       agents,
		SessionID:    sessionID,
		StartedAt:    t,
		Progress:     r.progressFor(key),
	})

	r.update(key, func(rv *Review) {
		if err != nil {
			rv.finish(err)
			return
		}
		rv.SessionID = out.Session.ID
		rv.Summary = out.Session.Summary
		rv.Concerns = toConcerns(out.Concerns)
		rv.mergeStoredAgents(out.Agents, r.path)
		rv.finish(nil)
	})

	if err != nil {
		r.persistFailure(key, err)
		log.Error("orchestrator: review failed", "err", err)
		return
	}
	log.Info("orchestrator: review complete", "concerns", len(out.Concerns), "duration_ms", time.Since(t).Milliseconds())
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
		r.update(key, func(rv *Review) {
			if ev.SessionID != "" {
				rv.setAgentSession(ev.Agent, ev.SessionID, r.path(ev.SessionID))
				if rv.OpencodeSessionID == "" {
					rv.OpencodeSessionID = ev.SessionID
					rv.OpencodeSessionPath = r.path(ev.SessionID)
				}
			}
			if ev.Done {
				rv.endStage(ev.Agent, ev.Stage, ev.Detail, ev.Err)
				return
			}
			rv.startStage(ev.Agent, ev.Stage)
		})
	}
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func (r *Runner) update(key string, mutate func(*Review)) {
	r.mu.Lock()
	review, ok := r.inFlight[key]
	if !ok {
		r.mu.Unlock()
		return
	}
	mutate(review)
	snapshot := review.clone()
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
	delete(r.inFlight, key)
	r.mu.Unlock()
}
