package reviewer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/harrylawton/pr-review/internal/checkout"
	"github.com/harrylawton/pr-review/internal/opencode"
	"github.com/harrylawton/pr-review/internal/store"
)

const maxDiffBytes = 60000

const reportTool = "report"

const (
	AgentReviewer = "pr-reviewer"
	AgentSkeptic  = "pr-skeptic"
)

var agentOrder = []string{AgentReviewer, AgentSkeptic}

func AgentOrder() []string { return append([]string{}, agentOrder...) }

func agentRank(name string) int {
	for i, a := range agentOrder {
		if a == name {
			return i
		}
	}
	return len(agentOrder)
}

type Reviewer struct {
	oc       *opencode.Client
	checkout *checkout.Manager
	store    *store.Store
}

func New(oc *opencode.Client, co *checkout.Manager, s *store.Store) *Reviewer {
	return &Reviewer{oc: oc, checkout: co, store: s}
}

func (r *Reviewer) Checkout() *checkout.Manager { return r.checkout }

type ProgressEvent struct {
	Agent     string
	Stage     string
	Done      bool
	Detail    string
	SessionID string
	Err       error
}

type Progress func(ProgressEvent)

const (
	StageSession = "session"
	StagePrompt  = "prompt"
	StageParse   = "parse"
	StageStore   = "store"
)

type ReviewRequest struct {
	Owner        string
	Repo         string
	PRNumber     int
	HeadSHA      string
	Title        string
	Body         string
	Diff         string
	CheckoutPath string
	Agents       []string
	StartedAt    time.Time
	Progress     Progress
}

func (r ReviewRequest) elapsedMS() int64 {
	if r.StartedAt.IsZero() {
		return 0
	}
	return time.Since(r.StartedAt).Milliseconds()
}

type Concern struct {
	Agent    string `json:"-"`
	File     string `json:"file"`
	Line     *int   `json:"line,omitempty"`
	Side     string `json:"side,omitempty"`
	Anchor   string `json:"anchor,omitempty"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

type reviewOutput struct {
	Summary  string    `json:"summary"`
	Concerns []Concern `json:"concerns"`
}

type agentResult struct {
	agent     string
	sessionID string
	out       reviewOutput
	err       error
}

func (r *Reviewer) Review(ctx context.Context, req ReviewRequest) (*store.ReviewSession, []*store.ReviewConcern, error) {
	log := slog.With("owner", req.Owner, "repo", req.Repo, "pr", req.PRNumber)
	emit := req.Progress
	if emit == nil {
		emit = func(ProgressEvent) {}
	}

	agents := req.Agents
	if len(agents) == 0 {
		agents = []string{AgentReviewer}
	}

	results := make([]agentResult, len(agents))
	var wg sync.WaitGroup
	for i, name := range agents {
		wg.Go(func() {
			results[i] = r.runAgent(req, name, emit, log)
		})
	}
	wg.Wait()

	sortByAgentRank(results)

	succeeded, failed := partition(results)
	if len(succeeded) == 0 {
		return nil, nil, firstError(failed)
	}
	for _, res := range failed {
		log.Warn("reviewer: agent failed, continuing", "agent", res.agent, "err", res.err)
	}

	index := parseDiffIndex(req.Diff)
	var concerns []Concern
	for i := range succeeded {
		for _, c := range succeeded[i].out.Concerns {
			c.Agent = succeeded[i].agent
			concerns = append(concerns, index.resolve(c))
		}
	}

	emit(ProgressEvent{Stage: StageStore})
	session, stored, err := r.persist(ctx, req, succeeded, concerns)
	if err != nil {
		emit(ProgressEvent{Stage: StageStore, Done: true, Err: err})
		return nil, nil, err
	}
	emit(ProgressEvent{Stage: StageStore, Done: true})

	log.Info("reviewer: review complete", "agents", len(succeeded), "failed", len(failed), "concerns", len(stored))
	return session, stored, nil
}

func (r *Reviewer) runAgent(req ReviewRequest, name string, emit Progress, parent *slog.Logger) agentResult {
	log := parent.With("agent", name)
	res := agentResult{agent: name}

	fail := func(stage string, err error) agentResult {
		emit(ProgressEvent{Agent: name, Stage: stage, Done: true, Err: err})
		log.Error("reviewer: "+stage+" failed", "err", err)
		res.err = err
		return res
	}

	emit(ProgressEvent{Agent: name, Stage: StageSession})
	t := time.Now()
	sess, err := r.oc.CreateSession(opencode.CreateSessionRequest{
		Title:      fmt.Sprintf("%s/%s #%d — %s", req.Owner, req.Repo, req.PRNumber, name),
		Agent:      name,
		Permission: opencode.ReadOnlyPermission(req.CheckoutPath),
	})
	if err != nil {
		return fail(StageSession, fmt.Errorf("reviewer: create session (%s): %w", name, err))
	}
	res.sessionID = sess.ID
	log = log.With("session_id", sess.ID)
	emit(ProgressEvent{Agent: name, Stage: StageSession, Done: true, SessionID: sess.ID})
	log.Info("reviewer: session created", "duration_ms", time.Since(t).Milliseconds())

	prompt := buildPrompt(req)
	emit(ProgressEvent{Agent: name, Stage: StagePrompt})
	t = time.Now()
	msg, err := r.oc.Prompt(sess.ID, opencode.PromptRequest{
		Agent: name,
		Parts: []opencode.Part{{Type: "text", Text: prompt}},
	})
	if err != nil {
		return fail(StagePrompt, fmt.Errorf("reviewer: prompt (%s): %w", name, err))
	}
	msgs, err := r.oc.Messages(sess.ID)
	if err != nil {
		return fail(StagePrompt, fmt.Errorf("reviewer: read session messages (%s): %w", name, err))
	}
	reported, ok := opencode.ToolInput(msgs, reportTool)
	if !ok {
		if msg.Info.HasError() {
			return fail(StagePrompt, fmt.Errorf("reviewer: model error (%s): %s", name, string(msg.Info.Error)))
		}
		return fail(StagePrompt, fmt.Errorf("reviewer: %s never called the %s tool", name, reportTool))
	}
	if msg.Info.HasError() {
		log.Warn("reviewer: model error after report delivered", "err", string(msg.Info.Error))
	}
	emit(ProgressEvent{Agent: name, Stage: StagePrompt, Done: true})
	log.Info("reviewer: model responded", "duration_ms", time.Since(t).Milliseconds())

	emit(ProgressEvent{Agent: name, Stage: StageParse})
	if err := json.Unmarshal(reported, &res.out); err != nil {
		log.Error("reviewer: raw tool input", "raw", string(reported))
		return fail(StageParse, fmt.Errorf("reviewer: parse %s input (%s): %w", reportTool, name, err))
	}
	emit(ProgressEvent{Agent: name, Stage: StageParse, Done: true, Detail: concernCount(len(res.out.Concerns))})
	log.Info("reviewer: parsed concerns", "count", len(res.out.Concerns))

	return res
}

func (r *Reviewer) persist(
	ctx context.Context,
	req ReviewRequest,
	results []agentResult,
	concerns []Concern,
) (*store.ReviewSession, []*store.ReviewConcern, error) {
	session, err := r.store.CreateReviewSession(ctx, store.NewReviewSession{
		Owner:             req.Owner,
		Repo:              req.Repo,
		PRNumber:          req.PRNumber,
		HeadSHA:           req.HeadSHA,
		OpencodeSessionID: primarySessionID(results),
		Summary:           primarySummary(results),
		Agents:            strings.Join(agentNames(results), ","),
		Status:            store.ReviewStatusDone,
		DurationMS:        req.elapsedMS(),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("reviewer: store session: %w", err)
	}

	out := make([]*store.ReviewConcern, 0, len(concerns))
	for _, c := range concerns {
		stored, err := r.store.CreateConcern(ctx, store.NewConcern{
			SessionID: session.ID,
			Agent:     c.Agent,
			File:      c.File,
			Line:      c.Line,
			Side:      c.Side,
			Severity:  c.Severity,
			Title:     c.Title,
			Body:      c.Body,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("reviewer: store concern: %w", err)
		}
		out = append(out, stored)
	}
	return session, out, nil
}

func sortByAgentRank(results []agentResult) {
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && agentRank(results[j].agent) < agentRank(results[j-1].agent); j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}

func partition(results []agentResult) (succeeded, failed []agentResult) {
	for _, res := range results {
		if res.err != nil {
			failed = append(failed, res)
			continue
		}
		succeeded = append(succeeded, res)
	}
	return succeeded, failed
}

func firstError(results []agentResult) error {
	for _, res := range results {
		if res.err != nil {
			return res.err
		}
	}
	return nil
}

func agentNames(results []agentResult) []string {
	names := make([]string, 0, len(results))
	for _, res := range results {
		names = append(names, res.agent)
	}
	return names
}

func primarySummary(results []agentResult) string {
	for _, res := range results {
		if s := strings.TrimSpace(res.out.Summary); s != "" {
			return s
		}
	}
	return ""
}

func primarySessionID(results []agentResult) string {
	for _, res := range results {
		if res.sessionID != "" {
			return res.sessionID
		}
	}
	return ""
}

func buildPrompt(req ReviewRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "PR: %s/%s #%d — %s\n", req.Owner, req.Repo, req.PRNumber, req.Title)
	if body := strings.TrimSpace(req.Body); body != "" {
		fmt.Fprintf(&b, "\n%s\n", body)
	}
	fmt.Fprintf(&b, "\n---\nDiff:\n%s\n", truncateDiff(req.Diff, maxDiffBytes))
	fmt.Fprintf(&b, "\n---\nReview this diff, then call the %s tool exactly once with the summary and every concern you are confident about. An empty concerns list is a valid, complete review.", reportTool)
	return b.String()
}

func truncateDiff(diff string, maxBytes int) string {
	if len(diff) <= maxBytes {
		return diff
	}
	cut := strings.LastIndex(diff[:maxBytes], "\ndiff --git ")
	if cut <= 0 {
		cut = maxBytes
	}
	return diff[:cut] + fmt.Sprintf("\n\n[diff truncated — %d of %d bytes shown. Review only what is above; do not go looking for the rest.]\n", cut, len(diff))
}

func concernCount(n int) string {
	if n == 1 {
		return "1 concern"
	}
	return fmt.Sprintf("%d concerns", n)
}
