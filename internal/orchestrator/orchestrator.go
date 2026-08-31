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

const subscriberBuffer = 32

type Subscription struct {
	ID       string
	ReviewID string
	Ch       chan *Review
}

type Orchestrator struct {
	ctx      context.Context
	logger   *slog.Logger
	reviewer *reviewer.Reviewer

	mu          sync.RWMutex
	reviews     map[string]*Review
	active      map[string]string
	subscribers map[string]map[string]*Subscription
}

func New(ctx context.Context, logger *slog.Logger, rev *reviewer.Reviewer) *Orchestrator {
	return &Orchestrator{
		ctx:         ctx,
		logger:      logger,
		reviewer:    rev,
		reviews:     make(map[string]*Review),
		active:      make(map[string]string),
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
	key := prKey(in.Owner, in.Repo, in.PRNumber)

	o.mu.Lock()
	if id, ok := o.active[key]; ok {
		existing := o.reviews[id]
		o.mu.Unlock()
		return existing.clone(), nil
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
	o.active[key] = id
	snapshot := review.clone()
	o.mu.Unlock()

	go o.run(id, key, in)

	return snapshot, nil
}

func (o *Orchestrator) run(id, key string, in StartInput) {
	log := o.logger.With("review_id", id, "owner", in.Owner, "repo", in.Repo, "pr", in.PRNumber)

	o.update(id, func(r *Review) {
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
		Progress: o.progressFor(id),
	})

	o.update(id, func(r *Review) {
		if err != nil {
			r.finish(err)
			return
		}
		r.SessionID = sess.ID
		r.Summary = sess.Summary
		r.Concerns = toConcerns(concerns)
		r.finish(nil)
	})

	o.mu.Lock()
	delete(o.active, key)
	o.mu.Unlock()

	if err != nil {
		log.Error("orchestrator: review failed", "err", err)
		return
	}
	log.Info("orchestrator: review complete", "concerns", len(concerns))
}

func (o *Orchestrator) progressFor(id string) reviewer.Progress {
	return func(ev reviewer.ProgressEvent) {
		o.update(id, func(r *Review) {
			if ev.Done {
				r.endStage(ev.Stage, ev.Detail, ev.Err)
				return
			}
			r.startStage(ev.Stage)
		})
	}
}

func (o *Orchestrator) update(id string, mutate func(*Review)) {
	o.mu.Lock()
	review, ok := o.reviews[id]
	if !ok {
		o.mu.Unlock()
		return
	}
	mutate(review)
	snapshot := review.clone()
	subs := make([]*Subscription, 0, len(o.subscribers[id]))
	for _, sub := range o.subscribers[id] {
		subs = append(subs, sub)
	}
	o.mu.Unlock()

	for _, sub := range subs {
		select {
		case sub.Ch <- snapshot:
		default:
		}
	}
}

func (o *Orchestrator) Get(id string) (*Review, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	review, ok := o.reviews[id]
	if !ok {
		return nil, false
	}
	return review.clone(), true
}

func (o *Orchestrator) Subscribe(reviewID string) (*Subscription, *Review, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	review, ok := o.reviews[reviewID]
	if !ok {
		return nil, nil, fmt.Errorf("orchestrator: unknown review %s", reviewID)
	}

	id, err := newID()
	if err != nil {
		return nil, nil, err
	}

	sub := &Subscription{ID: id, ReviewID: reviewID, Ch: make(chan *Review, subscriberBuffer)}
	list, ok := o.subscribers[reviewID]
	if !ok {
		list = make(map[string]*Subscription)
		o.subscribers[reviewID] = list
	}
	list[id] = sub

	o.logger.Info("orchestrator: subscriber added", "review_id", reviewID, "sub_id", id, "subscribers", len(list))
	return sub, review.clone(), nil
}

func (o *Orchestrator) Unsubscribe(sub *Subscription) {
	o.mu.Lock()
	defer o.mu.Unlock()

	list, ok := o.subscribers[sub.ReviewID]
	if !ok {
		return
	}
	if _, ok := list[sub.ID]; !ok {
		return
	}
	delete(list, sub.ID)
	close(sub.Ch)

	if len(list) == 0 {
		delete(o.subscribers, sub.ReviewID)
	}
	o.logger.Info("orchestrator: subscriber removed", "review_id", sub.ReviewID, "sub_id", sub.ID, "subscribers", len(list))
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

func prKey(owner, repo string, prNumber int) string {
	return fmt.Sprintf("%s/%s#%d", owner, repo, prNumber)
}

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
