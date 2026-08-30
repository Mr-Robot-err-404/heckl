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
	oc      *opencode.Client
	wt      *worktree.Manager
	store   *store.Store
}

func New(oc *opencode.Client, wt *worktree.Manager, s *store.Store) *Reviewer {
	return &Reviewer{oc: oc, wt: wt, store: s}
}

type ReviewRequest struct {
	Owner     string
	Repo      string
	PRNumber  int
	HeadSHA   string
	Title     string
	Body      string
	Diff      string
}

type Concern struct {
	File     string `json:"file"`
	Line     *int   `json:"line,omitempty"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

type reviewOutput struct {
	Concerns []Concern `json:"concerns"`
}

var concernsSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
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
	"required":             []string{"concerns"},
	"additionalProperties": false,
}

func (r *Reviewer) Review(ctx context.Context, req ReviewRequest) (*store.ReviewSession, []*store.ReviewConcern, error) {
	log := slog.With("owner", req.Owner, "repo", req.Repo, "pr", req.PRNumber)

	log.Info("reviewer: ensuring worktree", "head_sha", req.HeadSHA)
	t := time.Now()
	wtPath, err := r.wt.EnsureWorktree(req.Owner, req.Repo, req.PRNumber, req.HeadSHA)
	if err != nil {
		log.Error("reviewer: worktree failed", "err", err, "duration_ms", time.Since(t).Milliseconds())
		return nil, nil, fmt.Errorf("reviewer: worktree: %w", err)
	}
	log.Info("reviewer: worktree ready", "path", wtPath, "duration_ms", time.Since(t).Milliseconds())

	log.Info("reviewer: creating opencode session", "agent", "pr-reviewer")
	t = time.Now()
	sess, err := r.oc.CreateSession(opencode.CreateSessionRequest{
		Title:      fmt.Sprintf("%s/%s #%d", req.Owner, req.Repo, req.PRNumber),
		Agent:      "pr-reviewer",
		Permission: opencode.WorktreePermission(wtPath),
	})
	if err != nil {
		log.Error("reviewer: create session failed", "err", err, "duration_ms", time.Since(t).Milliseconds())
		return nil, nil, fmt.Errorf("reviewer: create session: %w", err)
	}
	log = log.With("session_id", sess.ID)
	log.Info("reviewer: session created", "duration_ms", time.Since(t).Milliseconds())

	prompt := buildPrompt(req)
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
		log.Error("reviewer: prompt failed", "err", err, "duration_ms", time.Since(t).Milliseconds())
		return nil, nil, fmt.Errorf("reviewer: prompt: %w", err)
	}
	log.Info("reviewer: model responded", "duration_ms", time.Since(t).Milliseconds())
	if msg.Info.HasError() {
		log.Error("reviewer: model returned error", "model_error", string(msg.Info.Error))
		return nil, nil, fmt.Errorf("reviewer: model error: %s", string(msg.Info.Error))
	}
	if len(msg.Info.Structured) == 0 {
		log.Error("reviewer: no structured output returned")
		return nil, nil, fmt.Errorf("reviewer: no structured output returned")
	}

	var out reviewOutput
	if err := json.Unmarshal(msg.Info.Structured, &out); err != nil {
		log.Error("reviewer: parse output failed", "err", err, "raw", string(msg.Info.Structured))
		return nil, nil, fmt.Errorf("reviewer: parse output: %w", err)
	}
	log.Info("reviewer: parsed concerns", "count", len(out.Concerns))

	reviewSess, err := r.store.CreateReviewSession(ctx, req.Owner, req.Repo, req.PRNumber, req.HeadSHA, sess.ID)
	if err != nil {
		log.Error("reviewer: store session failed", "err", err)
		return nil, nil, fmt.Errorf("reviewer: store session: %w", err)
	}

	concerns := make([]*store.ReviewConcern, 0, len(out.Concerns))
	for _, c := range out.Concerns {
		stored, err := r.store.CreateConcern(ctx, reviewSess.ID, c.File, c.Line, c.Severity, c.Title, c.Body)
		if err != nil {
			log.Error("reviewer: store concern failed", "err", err, "file", c.File)
			return nil, nil, fmt.Errorf("reviewer: store concern: %w", err)
		}
		concerns = append(concerns, stored)
	}

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
