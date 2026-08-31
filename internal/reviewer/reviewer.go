package reviewer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/harrylawton/pr-review/internal/opencode"
	"github.com/harrylawton/pr-review/internal/store"
	"github.com/harrylawton/pr-review/internal/worktree"
)

type Reviewer struct {
	oc    *opencode.Client
	wt    *worktree.Manager
	store *store.Store
}

func New(oc *opencode.Client, wt *worktree.Manager, s *store.Store) *Reviewer {
	return &Reviewer{oc: oc, wt: wt, store: s}
}

type ProgressEvent struct {
	Stage  string
	Done   bool
	Detail string
	Err    error
}

type Progress func(ProgressEvent)

const (
	StageWorktree = "worktree"
	StageSession  = "session"
	StagePrompt   = "prompt"
	StageParse    = "parse"
	StageStore    = "store"
)

type ReviewRequest struct {
	Owner    string
	Repo     string
	PRNumber int
	HeadSHA  string
	Title    string
	Body     string
	Diff     string
	Progress Progress
}

type Concern struct {
	File     string `json:"file"`
	Line     *int   `json:"line,omitempty"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

type reviewOutput struct {
	Summary  string    `json:"summary"`
	Concerns []Concern `json:"concerns"`
}

var concernsSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"summary": map[string]any{
			"type":        "string",
			"description": "What this PR is trying to do, in one or two sentences. Plain and specific.",
		},
		"concerns": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file":     map[string]any{"type": "string"},
					"line":     map[string]any{"type": "integer"},
					"severity": map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}},
					"title":    map[string]any{"type": "string"},
					"body":     map[string]any{"type": "string"},
				},
				"required":             []string{"file", "severity", "title", "body"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"summary", "concerns"},
	"additionalProperties": false,
}

func (r *Reviewer) Review(ctx context.Context, req ReviewRequest) (*store.ReviewSession, []*store.ReviewConcern, error) {
	log := slog.With("owner", req.Owner, "repo", req.Repo, "pr", req.PRNumber)
	emit := req.Progress
	if emit == nil {
		emit = func(ProgressEvent) {}
	}
	fail := func(stage string, err error) error {
		emit(ProgressEvent{Stage: stage, Done: true, Err: err})
		log.Error("reviewer: "+stage+" failed", "err", err)
		return err
	}

	emit(ProgressEvent{Stage: StageWorktree})
	log.Info("reviewer: ensuring worktree", "head_sha", req.HeadSHA)
	t := time.Now()
	wtPath, err := r.wt.EnsureWorktree(req.Owner, req.Repo, req.PRNumber, req.HeadSHA)
	if err != nil {
		return nil, nil, fail(StageWorktree, fmt.Errorf("reviewer: worktree: %w", err))
	}
	emit(ProgressEvent{Stage: StageWorktree, Done: true, Detail: wtPath})
	log.Info("reviewer: worktree ready", "path", wtPath, "duration_ms", time.Since(t).Milliseconds())

	emit(ProgressEvent{Stage: StageSession})
	log.Info("reviewer: creating opencode session", "agent", "pr-reviewer")
	t = time.Now()
	sess, err := r.oc.CreateSession(opencode.CreateSessionRequest{
		Title:      fmt.Sprintf("%s/%s #%d", req.Owner, req.Repo, req.PRNumber),
		Agent:      "pr-reviewer",
		Permission: opencode.WorktreePermission(wtPath),
	})
	if err != nil {
		return nil, nil, fail(StageSession, fmt.Errorf("reviewer: create session: %w", err))
	}
	log = log.With("session_id", sess.ID)
	emit(ProgressEvent{Stage: StageSession, Done: true, Detail: sess.ID})
	log.Info("reviewer: session created", "duration_ms", time.Since(t).Milliseconds())

	prompt := buildPrompt(req)
	emit(ProgressEvent{Stage: StagePrompt})
	log.Info("reviewer: prompting model", "prompt_bytes", len(prompt))
	t = time.Now()
	msg, err := r.oc.Prompt(sess.ID, opencode.PromptRequest{
		Parts: []opencode.Part{{Type: "text", Text: prompt}},
		Format: &opencode.OutputFormat{
			Type:       "json_schema",
			Schema:     concernsSchema,
			RetryCount: 2,
		},
	})
	if err != nil {
		return nil, nil, fail(StagePrompt, fmt.Errorf("reviewer: prompt: %w", err))
	}
	if msg.Info.HasError() {
		return nil, nil, fail(StagePrompt, fmt.Errorf("reviewer: model error: %s", string(msg.Info.Error)))
	}
	if len(msg.Info.Structured) == 0 {
		return nil, nil, fail(StagePrompt, fmt.Errorf("reviewer: no structured output returned"))
	}
	emit(ProgressEvent{Stage: StagePrompt, Done: true})
	log.Info("reviewer: model responded", "duration_ms", time.Since(t).Milliseconds())

	emit(ProgressEvent{Stage: StageParse})
	var out reviewOutput
	if err := json.Unmarshal(msg.Info.Structured, &out); err != nil {
		log.Error("reviewer: raw structured output", "raw", string(msg.Info.Structured))
		return nil, nil, fail(StageParse, fmt.Errorf("reviewer: parse output: %w", err))
	}
	emit(ProgressEvent{Stage: StageParse, Done: true, Detail: fmt.Sprintf("%d concerns", len(out.Concerns))})
	log.Info("reviewer: parsed concerns", "count", len(out.Concerns), "summary", out.Summary)

	emit(ProgressEvent{Stage: StageStore})
	reviewSess, err := r.store.CreateReviewSession(ctx, req.Owner, req.Repo, req.PRNumber, req.HeadSHA, sess.ID, out.Summary)
	if err != nil {
		return nil, nil, fail(StageStore, fmt.Errorf("reviewer: store session: %w", err))
	}

	concerns := make([]*store.ReviewConcern, 0, len(out.Concerns))
	for _, c := range out.Concerns {
		stored, err := r.store.CreateConcern(ctx, reviewSess.ID, c.File, c.Line, c.Severity, c.Title, c.Body)
		if err != nil {
			return nil, nil, fail(StageStore, fmt.Errorf("reviewer: store concern: %w", err))
		}
		concerns = append(concerns, stored)
	}
	emit(ProgressEvent{Stage: StageStore, Done: true})

	log.Info("reviewer: review complete", "concerns", len(concerns))
	return reviewSess, concerns, nil
}

func buildPrompt(req ReviewRequest) string {
	return fmt.Sprintf(`PR: %s/%s #%d — %s

%s

---
Diff:
%s

---
Review this diff like a senior engineer doing a quick pass, not an audit.
The worktree is available if you genuinely need to check something the diff
doesn't answer — don't explore it just because you can. Return your
concerns, which may be an empty list.`, req.Owner, req.Repo, req.PRNumber, req.Title, req.Body, truncateDiff(req.Diff, 20000))
}

func truncateDiff(diff string, maxBytes int) string {
	if len(diff) <= maxBytes {
		return diff
	}
	cutoff := fmt.Sprintf("\n[diff truncated at %s — use the worktree to read full files]\n", byteSize(len(diff)))
	return diff[:maxBytes] + cutoff
}

func byteSize(n int) string {
	return fmt.Sprintf("%dKB", n/1024)
}
