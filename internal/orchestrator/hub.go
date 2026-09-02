package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/harrylawton/pr-review/internal/store"
)

const globalKey = "*"

type Subscription struct {
	ID  string
	Key string

	mu      sync.Mutex
	pending map[string]*Review
	signal  chan struct{}
	closed  bool
}

func newSubscription(id, key string) *Subscription {
	return &Subscription{
		ID:      id,
		Key:     key,
		pending: make(map[string]*Review),
		signal:  make(chan struct{}, 1),
	}
}

func (s *Subscription) push(key string, review *Review) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.pending[key] = review
	s.mu.Unlock()

	select {
	case s.signal <- struct{}{}:
	default:
	}
}

func (s *Subscription) Ready() <-chan struct{} { return s.signal }

func (s *Subscription) Take() []*Review {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.pending) == 0 {
		return nil
	}
	out := make([]*Review, 0, len(s.pending))
	for _, review := range s.pending {
		out = append(out, review)
	}
	clear(s.pending)
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.Before(out[j].StartedAt) })
	return out
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
	global  map[string]*Subscription
}

func newHub(ctx context.Context, logger *slog.Logger, st ReviewStore, path SessionPath) *Hub {
	return &Hub{
		ctx:     ctx,
		logger:  logger,
		store:   st,
		path:    path,
		entries: make(map[string]*hubEntry),
		global:  make(map[string]*Subscription),
	}
}

func (h *Hub) Publish(key string, review *Review) {
	h.mu.Lock()
	subs := make([]*Subscription, 0, len(h.global))
	for _, sub := range h.global {
		subs = append(subs, sub)
	}
	if entry, watched := h.entries[key]; watched {
		entry.review = review
		for _, sub := range entry.subs {
			subs = append(subs, sub)
		}
	}
	h.mu.Unlock()

	for _, sub := range subs {
		sub.push(key, review)
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

	h.logger.Debug("orchestrator: subscriber added", "pr", key, "sub_id", id, "subscribers", len(entry.subs))
	return sub, entry.review, nil
}

func (h *Hub) SubscribeAll() (*Subscription, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	sub := newSubscription(id, globalKey)
	h.global[id] = sub

	h.logger.Debug("orchestrator: global subscriber added", "sub_id", id, "subscribers", len(h.global))
	return sub, nil
}

func (h *Hub) Unsubscribe(sub *Subscription) {
	if sub.Key == globalKey {
		h.unsubscribeGlobal(sub)
		return
	}

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
	h.logger.Debug("orchestrator: subscriber removed", "pr", sub.Key, "sub_id", sub.ID, "subscribers", len(entry.subs))
}

func (h *Hub) unsubscribeGlobal(sub *Subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.global[sub.ID]; !ok {
		return
	}
	delete(h.global, sub.ID)
	sub.close()
	h.logger.Debug("orchestrator: global subscriber removed", "sub_id", sub.ID, "subscribers", len(h.global))
}

func (h *Hub) watching(key string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.entries[key]
	return ok
}

func storedAgents(csv string, status Status) []Agent {
	out := []Agent{}
	for _, name := range strings.Split(csv, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out = append(out, Agent{Name: name, Status: status, Stages: []Stage{}})
	}
	return out
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

	h.logger.Debug("orchestrator: hydrated from store", "pr", key, "session_id", sess.ID, "concerns", len(concerns))

	return &Review{
		ID:        fmt.Sprintf("stored-%d", sess.ID),
		Owner:     sess.Owner,
		Repo:      sess.Repo,
		PRNumber:  sess.PRNumber,
		HeadSHA:   sess.HeadSHA,
		Status:    status,
		Stages:    []Stage{},
		Agents:    storedAgents(sess.Agents, status),
		SessionID: sess.ID,
		Summary:   sess.Summary,
		Error:     sess.Error,

		OpencodeSessionPath: h.path(sess.OpencodeSessionID),
		Concerns:            toConcerns(concerns),
		StartedAt:           at,
		EndedAt:             &ended,
	}
}
