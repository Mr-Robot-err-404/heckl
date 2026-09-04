package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/harrylawton/pr-review/internal/orchestrator"
	"github.com/harrylawton/pr-review/internal/reviewer"
	"github.com/harrylawton/pr-review/internal/store"
)

const streamPingInterval = 5 * time.Second

func (s *Server) handleReview(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		jsonError(w, "invalid pr number", http.StatusBadRequest)
		return
	}

	var body struct {
		Agents []string `json:"agents"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	review, err := s.orchestrator.Start(owner, repo, number, body.Agents)
	if err != nil {
		slog.Error("review start failed", "owner", owner, "repo", repo, "pr", number, "err", err)
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("review started", "owner", owner, "repo", repo, "pr", number, "review_id", review.ID, "agents", len(review.Agents))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(review)
}

func (s *Server) handleRerunAgent(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	agent := r.PathValue("agent")
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		jsonError(w, "invalid pr number", http.StatusBadRequest)
		return
	}

	review, err := s.orchestrator.Rerun(owner, repo, number, agent)
	if errors.Is(err, orchestrator.ErrNoReview) {
		jsonError(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		slog.Error("agent rerun failed", "owner", owner, "repo", repo, "pr", number, "agent", agent, "err", err)
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info("agent rerun started", "owner", owner, "repo", repo, "pr", number, "agent", agent, "review_id", review.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(review)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, reviewer.AgentOrder())
}

func (s *Server) handleAgentConfig(w http.ResponseWriter, r *http.Request) {
	configs, err := s.agentConfigs(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	models, err := s.oc.Models()
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}
	jsonOK(w, map[string]any{"agents": configs, "models": models})
}

func (s *Server) handleSaveAgentConfig(w http.ResponseWriter, r *http.Request) {
	var body []store.AgentConfig
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}

	known := make(map[string]bool, len(reviewer.AgentOrder()))
	for _, name := range reviewer.AgentOrder() {
		known[name] = true
	}
	for _, c := range body {
		if !known[c.Name] {
			jsonError(w, "unknown agent: "+c.Name, http.StatusBadRequest)
			return
		}
	}

	if err := s.store.SaveAgentConfigs(r.Context(), body); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.handleAgentConfig(w, r)
}

type agentConfigResponse struct {
	store.AgentConfig
	DefaultModel string `json:"defaultModel"`
}

func (s *Server) agentConfigs(ctx context.Context) ([]agentConfigResponse, error) {
	stored, err := s.store.ListAgentConfigs(ctx)
	if err != nil {
		return nil, err
	}

	defaults := map[string]string{}
	agents, err := s.oc.Agents()
	if err != nil {
		return nil, err
	}
	for _, a := range agents {
		defaults[a.Name] = a.Model.Ref()
	}

	names := reviewer.AgentOrder()
	out := make([]agentConfigResponse, 0, len(names))
	for _, name := range names {
		cfg := stored[name]
		cfg.Name = name
		out = append(out, agentConfigResponse{AgentConfig: cfg, DefaultModel: defaults[name]})
	}
	return out, nil
}

func (s *Server) handleReviewStream(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		jsonError(w, "invalid pr number", http.StatusBadRequest)
		return
	}

	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		jsonError(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	sub, snapshot, err := s.orchestrator.Subscribe(owner, repo, number)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer s.orchestrator.Unsubscribe(sub)

	streamHeaders(w)
	pumpReviews(r.Context(), w, rc, sub, snapshot)
}

func (s *Server) handleReviewsStream(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		jsonError(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	sub, active, err := s.orchestrator.SubscribeAll()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer s.orchestrator.Unsubscribe(sub)

	streamHeaders(w)
	pumpReviews(r.Context(), w, rc, sub, active)
}

func streamHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
}

func pumpReviews(
	ctx context.Context,
	w http.ResponseWriter,
	rc *http.ResponseController,
	sub *orchestrator.Subscription,
	snapshot any,
) {
	if err := writeEvent(w, rc, "snapshot", snapshot); err != nil {
		return
	}

	ticker := time.NewTicker(streamPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		case _, ok := <-sub.Ready():
			if !ok {
				return
			}
			for _, review := range sub.Take() {
				if err := writeEvent(w, rc, "review", review); err != nil {
					return
				}
			}
		}
	}
}

const (
	defaultHistoryLimit = 20
	maxHistoryLimit     = 100
)

func (s *Server) handleReviewHistory(w http.ResponseWriter, r *http.Request) {
	limit, err := intParam(r, "limit", defaultHistoryLimit)
	if err != nil || limit <= 0 || limit > maxHistoryLimit {
		jsonError(w, "invalid limit", http.StatusBadRequest)
		return
	}
	offset, err := intParam(r, "offset", 0)
	if err != nil || offset < 0 {
		jsonError(w, "invalid offset", http.StatusBadRequest)
		return
	}

	sessions, err := s.store.ListRecentReviewSessions(r.Context(), store.HistoryQuery{
		Owner:  r.URL.Query().Get("owner"),
		Repo:   r.URL.Query().Get("repo"),
		Limit:  limit + 1,
		Offset: offset,
	})
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hasMore := len(sessions) > limit
	if hasMore {
		sessions = sessions[:limit]
	}
	for _, sess := range sessions {
		sess.OpencodeSessionPath = s.orchestrator.SessionPath(sess.OpencodeSessionID)
	}

	jsonOK(w, map[string]any{"sessions": sessions, "hasMore": hasMore})
}

func intParam(r *http.Request, name string, fallback int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}

func writeEvent(w http.ResponseWriter, rc *http.ResponseController, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return err
	}
	return rc.Flush()
}
