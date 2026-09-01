package orchestrator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/harrylawton/pr-review/internal/github"
	"github.com/harrylawton/pr-review/internal/reviewer"
	"github.com/harrylawton/pr-review/internal/store"
)

type storeConcern = store.ReviewConcern

type PRSource interface {
	GetPR(owner, repo string, number int) (*github.PR, error)
	GetPRDiff(owner, repo string, number int) ([]byte, error)
}

type Subscription struct {
	ID  string
	Key string

	mu      sync.Mutex
	pending *Review
	signal  chan struct{}
	closed  bool
}

func newSubscription(id, key string) *Subscription {
	return &Subscription{ID: id, Key: key, signal: make(chan struct{}, 1)}
}

func (s *Subscription) push(review *Review) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.pending = review
	s.mu.Unlock()

	select {
	case s.signal <- struct{}{}:
	default:
	}
}

func (s *Subscription) Ready() <-chan struct{} { return s.signal }

func (s *Subscription) Take() (*Review, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil {
		return nil, false
	}
	review := s.pending
	s.pending = nil
	return review, true
}

func (s *Subscription) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	close(s.signal)
}

type Orchestrator struct {
	ctx      context.Context
	logger   *slog.Logger
	source   PRSource
	reviewer *reviewer.Reviewer

	mu          sync.RWMutex
	reviews     map[string]*Review
	latest      map[string]string
	active      map[string]struct{}
	subscribers map[string]map[string]*Subscription
}

func New(ctx context.Context, logger *slog.Logger, source PRSource, rev *reviewer.Reviewer) *Orchestrator {
	return &Orchestrator{
		ctx:         ctx,
		logger:      logger,
		source:      source,
		reviewer:    rev,
		reviews:     make(map[string]*Review),
		latest:      make(map[string]string),
		active:      make(map[string]struct{}),
		subscribers: make(map[string]map[string]*Subscription),
	}
}

func (o *Orchestrator) Start(owner, repo string, prNumber int) (*Review, error) {
	key := PRKey(owner, repo, prNumber)

	o.mu.Lock()
	if _, running := o.active[key]; running {
		existing := o.reviews[o.latest[key]].clone()
		o.mu.Unlock()
		return existing, nil
	}

	id, err := newID()
	if err != nil {
		o.mu.Unlock()
		return nil, err
	}

	review := newReview(id, owner, repo, prNumber)
	review.Status = StatusRunning
	review.startStage(StageFetch)
	review.setAgent(agentName, StatusRunning)
	o.reviews[id] = review
	o.latest[key] = id
	o.active[key] = struct{}{}
	snapshot := review.clone()
	o.mu.Unlock()

	o.broadcast(key, snapshot)
	go o.run(id, key, owner, repo, prNumber)

	return snapshot, nil
}

func (o *Orchestrator) run(id, key, owner, repo string, prNumber int) {
	log := o.logger.With("review_id", id, "owner", owner, "repo", repo, "pr", prNumber)

	defer func() {
		o.mu.Lock()
		delete(o.active, key)
		o.mu.Unlock()
	}()

	t := time.Now()
	pr, diff, err := o.fetch(owner, repo, prNumber)
	if err != nil {
		o.update(key, id, func(r *Review) {
			r.endStage(StageFetch, "", err)
			r.finish(err)
		})
		log.Error("orchestrator: fetch failed", "err", err)
		return
	}
	headSHA := pr.HeadSHA()
	o.update(key, id, func(r *Review) {
		r.HeadSHA = headSHA
		r.endStage(StageFetch, diffSize(len(diff)), nil)
	})
	log.Info("orchestrator: pr fetched", "head_sha", headSHA, "diff_bytes", len(diff), "duration_ms", time.Since(t).Milliseconds())

	sess, concerns, err := o.reviewer.Review(o.ctx, reviewer.ReviewRequest{
		Owner:    owner,
		Repo:     repo,
		PRNumber: prNumber,
		HeadSHA:  headSHA,
		Title:    pr.Title,
		Body:     pr.Body,
		Diff:     string(diff),
		Progress: o.progressFor(key, id),
	})

	o.update(key, id, func(r *Review) {
		if err != nil {
			r.finish(err)
			return
		}
		r.SessionID = sess.ID
		r.Summary = sess.Summary
		r.Concerns = toConcerns(concerns)
		r.finish(nil)
	})

	if err != nil {
		log.Error("orchestrator: review failed", "err", err)
		return
	}
	log.Info("orchestrator: review complete", "concerns", len(concerns), "duration_ms", time.Since(t).Milliseconds())
}

func (o *Orchestrator) fetch(owner, repo string, prNumber int) (*github.PR, []byte, error) {
	pr, err := o.source.GetPR(owner, repo, prNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch pr: %w", err)
	}
	diff, err := o.source.GetPRDiff(owner, repo, prNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch diff: %w", err)
	}
	return pr, diff, nil
}

func (o *Orchestrator) progressFor(key, id string) reviewer.Progress {
	return func(ev reviewer.ProgressEvent) {
		o.update(key, id, func(r *Review) {
			if ev.Done {
				r.endStage(ev.Stage, ev.Detail, ev.Err)
				return
			}
			r.startStage(ev.Stage)
		})
	}
}

func (o *Orchestrator) update(key, id string, mutate func(*Review)) {
	o.mu.Lock()
	review, ok := o.reviews[id]
	if !ok {
		o.mu.Unlock()
		return
	}
	mutate(review)
	snapshot := review.clone()
	o.mu.Unlock()

	o.broadcast(key, snapshot)
}

func (o *Orchestrator) broadcast(key string, snapshot *Review) {
	o.mu.RLock()
	subs := make([]*Subscription, 0, len(o.subscribers[key]))
	for _, sub := range o.subscribers[key] {
		subs = append(subs, sub)
	}
	o.mu.RUnlock()

	for _, sub := range subs {
		sub.push(snapshot)
	}
}

func (o *Orchestrator) Latest(owner, repo string, prNumber int) *Review {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.latestLocked(PRKey(owner, repo, prNumber))
}

func (o *Orchestrator) latestLocked(key string) *Review {
	id, ok := o.latest[key]
	if !ok {
		return nil
	}
	review, ok := o.reviews[id]
	if !ok {
		return nil
	}
	return review.clone()
}

func (o *Orchestrator) Subscribe(owner, repo string, prNumber int) (*Subscription, *Review, error) {
	key := PRKey(owner, repo, prNumber)

	o.mu.Lock()
	defer o.mu.Unlock()

	id, err := newID()
	if err != nil {
		return nil, nil, err
	}

	sub := newSubscription(id, key)
	list, ok := o.subscribers[key]
	if !ok {
		list = make(map[string]*Subscription)
		o.subscribers[key] = list
	}
	list[id] = sub

	o.logger.Info("orchestrator: subscriber added", "pr", key, "sub_id", id, "subscribers", len(list))
	return sub, o.latestLocked(key), nil
}

func (o *Orchestrator) Unsubscribe(sub *Subscription) {
	o.mu.Lock()
	defer o.mu.Unlock()

	list, ok := o.subscribers[sub.Key]
	if !ok {
		return
	}
	if _, ok := list[sub.ID]; !ok {
		return
	}
	delete(list, sub.ID)
	sub.close()

	if len(list) == 0 {
		delete(o.subscribers, sub.Key)
	}
	o.logger.Info("orchestrator: subscriber removed", "pr", sub.Key, "sub_id", sub.ID, "subscribers", len(list))
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
