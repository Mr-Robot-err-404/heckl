package orchestrator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"

	"github.com/harrylawton/pr-review/internal/reviewer"
	"github.com/harrylawton/pr-review/internal/store"
)

type storeConcern = store.ReviewConcern

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
	reviewer *reviewer.Reviewer

	mu          sync.RWMutex
	reviews     map[string]*Review
	latest      map[string]string
	active      map[string]struct{}
	subscribers map[string]map[string]*Subscription
}

func New(ctx context.Context, logger *slog.Logger, rev *reviewer.Reviewer) *Orchestrator {
	return &Orchestrator{
		ctx:         ctx,
		logger:      logger,
		reviewer:    rev,
		reviews:     make(map[string]*Review),
		latest:      make(map[string]string),
		active:      make(map[string]struct{}),
		subscribers: make(map[string]map[string]*Subscription),
	}
}

type StartInput struct {
	Owner    string
	Repo     string
	PRNumber int
	HeadSHA  string
	Title    string
	Body     string
	Diff     string
}

func (o *Orchestrator) Start(in StartInput) (*Review, error) {
	key := PRKey(in.Owner, in.Repo, in.PRNumber)

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

	review := newReview(id, in.Owner, in.Repo, in.PRNumber)
	review.HeadSHA = in.HeadSHA
	review.Status = StatusRunning
	o.reviews[id] = review
	o.latest[key] = id
	o.active[key] = struct{}{}
	snapshot := review.clone()
	o.mu.Unlock()

	o.broadcast(key, snapshot)
	go o.run(id, key, in)

	return snapshot, nil
}

func (o *Orchestrator) run(id, key string, in StartInput) {
	log := o.logger.With("review_id", id, "owner", in.Owner, "repo", in.Repo, "pr", in.PRNumber)

	o.update(key, id, func(r *Review) {
		r.setAgent("pr-reviewer", StatusRunning)
	})

	sess, concerns, err := o.reviewer.Review(o.ctx, reviewer.ReviewRequest{
		Owner:    in.Owner,
		Repo:     in.Repo,
		PRNumber: in.PRNumber,
		HeadSHA:  in.HeadSHA,
		Title:    in.Title,
		Body:     in.Body,
		Diff:     in.Diff,
		Progress: o.progressFor(key, id),
	})

	o.mu.Lock()
	delete(o.active, key)
	o.mu.Unlock()

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
	log.Info("orchestrator: review complete", "concerns", len(concerns))
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

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
