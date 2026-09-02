package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/harrylawton/pr-review/internal/store"
)

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

type hubEntry struct {
	review *Review
	subs   map[string]*Subscription
}

type Hub struct {
	ctx    context.Context
	logger *slog.Logger
	store  ReviewStore
	path   SessionPath

	mu      sync.Mutex
	entries map[string]*hubEntry
}

func newHub(ctx context.Context, logger *slog.Logger, st ReviewStore, path SessionPath) *Hub {
	return &Hub{
		ctx:     ctx,
		logger:  logger,
		store:   st,
		path:    path,
		entries: make(map[string]*hubEntry),
	}
}

func (h *Hub) Publish(key string, review *Review) {
	h.mu.Lock()
	entry, watched := h.entries[key]
	if !watched {
		h.mu.Unlock()
		return
	}
	entry.review = review
	subs := make([]*Subscription, 0, len(entry.subs))
	for _, sub := range entry.subs {
		subs = append(subs, sub)
	}
	h.mu.Unlock()

	for _, sub := range subs {
		sub.push(review)
	}
}

func (h *Hub) Subscribe(key string, seed *Review) (*Subscription, *Review, error) {
	id, err := newID()
	if err != nil {
		return nil, nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	entry, ok := h.entries[key]
	if !ok {
		entry = &hubEntry{review: seed, subs: make(map[string]*Subscription)}
		h.entries[key] = entry
	}

	sub := newSubscription(id, key)
	entry.subs[id] = sub

	h.logger.Info("orchestrator: subscriber added", "pr", key, "sub_id", id, "subscribers", len(entry.subs))
	return sub, entry.review, nil
}

func (h *Hub) Unsubscribe(sub *Subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry, ok := h.entries[sub.Key]
	if !ok {
		return
	}
	if _, ok := entry.subs[sub.ID]; !ok {
		return
	}
	delete(entry.subs, sub.ID)
	sub.close()

	if len(entry.subs) == 0 {
		delete(h.entries, sub.Key)
	}
	h.logger.Info("orchestrator: subscriber removed", "pr", sub.Key, "sub_id", sub.ID, "subscribers", len(entry.subs))
}

func (h *Hub) Cached(key string) *Review {
	h.mu.Lock()
	defer h.mu.Unlock()
	if entry, ok := h.entries[key]; ok {
		return entry.review
	}
	return nil
}

func (h *Hub) watching(key string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.entries[key]
	return ok
}

func (h *Hub) hydrate(owner, repo string, prNumber int) *Review {
	if h.store == nil {
		return nil
	}
	key := PRKey(owner, repo, prNumber)

	sessions, err := h.store.ListReviewSessions(h.ctx, owner, repo, prNumber)
	if err != nil {
		h.logger.Error("orchestrator: hydrate sessions", "pr", key, "err", err)
		return nil
	}
	if len(sessions) == 0 {
		return nil
	}
	sess := sessions[0]

	var concerns []*storeConcern
	if sess.Status != store.ReviewStatusError {
		concerns, err = h.store.ListConcerns(h.ctx, sess.ID)
		if err != nil {
			h.logger.Error("orchestrator: hydrate concerns", "pr", key, "session_id", sess.ID, "err", err)
			return nil
		}
	}

	ended, err := time.Parse(time.RFC3339, sess.CreatedAt)
	if err != nil {
		ended = time.Now().UTC()
	}
	at := ended.Add(-time.Duration(sess.DurationMS) * time.Millisecond)

	status := StatusDone
	if sess.Status == store.ReviewStatusError {
		status = StatusError
	}

	h.logger.Info("orchestrator: hydrated from store", "pr", key, "session_id", sess.ID, "concerns", len(concerns))

	return &Review{
		ID:        fmt.Sprintf("stored-%d", sess.ID),
		Owner:     sess.Owner,
		Repo:      sess.Repo,
		PRNumber:  sess.PRNumber,
		HeadSHA:   sess.HeadSHA,
		Status:    status,
		Stages:    []Stage{},
		Agents:    []Agent{},
		SessionID: sess.ID,
		Summary:   sess.Summary,
		Error:     sess.Error,

		OpencodeSessionPath: h.path(sess.OpencodeSessionID),
		Concerns:            toConcerns(concerns),
		StartedAt:           at,
		EndedAt:             &ended,
	}
}
