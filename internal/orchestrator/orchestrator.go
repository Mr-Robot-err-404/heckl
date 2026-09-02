package orchestrator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/harrylawton/pr-review/internal/github"
	"github.com/harrylawton/pr-review/internal/reviewer"
	"github.com/harrylawton/pr-review/internal/store"
)

type storeConcern = store.ReviewConcern

type PRSource interface {
	GetPR(owner, repo string, number int) (*github.PR, error)
	GetPRDiff(owner, repo string, number int) ([]byte, error)
}

type ReviewStore interface {
	ListReviewSessions(ctx context.Context, owner, repo string, prNumber int) ([]*store.ReviewSession, error)
	ListConcerns(ctx context.Context, sessionID int64) ([]*store.ReviewConcern, error)
	CreateReviewSession(ctx context.Context, in store.NewReviewSession) (*store.ReviewSession, error)
}

type SessionPath func(opencodeSessionID string) string

type Orchestrator struct {
	runner *Runner
	hub    *Hub
}

func New(ctx context.Context, logger *slog.Logger, source PRSource, rev *reviewer.Reviewer, st ReviewStore, path SessionPath) *Orchestrator {
	if path == nil {
		path = func(string) string { return "" }
	}
	hub := newHub(ctx, logger, st, path)
	return &Orchestrator{
		runner: newRunner(ctx, logger, source, rev, st, hub.Publish, path),
		hub:    hub,
	}
}

func (o *Orchestrator) Start(owner, repo string, prNumber int) (*Review, error) {
	return o.runner.Start(owner, repo, prNumber)
}

func (o *Orchestrator) Subscribe(owner, repo string, prNumber int) (*Subscription, *Review, error) {
	key := PRKey(owner, repo, prNumber)

	seed := o.runner.InFlight(key)
	if seed == nil && !o.hub.watching(key) {
		seed = o.hub.hydrate(owner, repo, prNumber)
	}

	return o.hub.Subscribe(key, seed)
}

func (o *Orchestrator) Unsubscribe(sub *Subscription) {
	o.hub.Unsubscribe(sub)
}

func (o *Orchestrator) SessionPath(opencodeSessionID string) string {
	if opencodeSessionID == "" {
		return ""
	}
	return o.hub.path(opencodeSessionID)
}

func (o *Orchestrator) Active() []*Review {
	return o.runner.Active()
}

func (o *Orchestrator) Latest(owner, repo string, prNumber int) *Review {
	key := PRKey(owner, repo, prNumber)
	if review := o.runner.InFlight(key); review != nil {
		return review
	}
	return o.hub.Cached(key)
}

func toConcerns(in []*storeConcern) []Concern {
	out := make([]Concern, 0, len(in))
	for _, c := range in {
		out = append(out, Concern{
			File:     c.File,
			Line:     c.Line,
			Side:     c.Side,
			Severity: c.Severity,
			Title:    c.Title,
			Body:     c.Body,
		})
	}
	return out
}

func PRKey(owner, repo string, prNumber int) string {
	return fmt.Sprintf("%s/%s#%d", owner, repo, prNumber)
}

func diffSize(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	return fmt.Sprintf("%dKB", n/1024)
}

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
