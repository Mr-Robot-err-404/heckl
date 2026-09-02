package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

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

	review, err := s.orchestrator.Start(owner, repo, number)
	if err != nil {
		slog.Error("review start failed", "owner", owner, "repo", repo, "pr", number, "err", err)
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("review started", "owner", owner, "repo", repo, "pr", number, "review_id", review.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(review)
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

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	if err := writeEvent(w, rc, "snapshot", snapshot); err != nil {
		return
	}

	ticker := time.NewTicker(streamPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
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
			review, has := sub.Take()
			if !has {
				continue
			}
			if err := writeEvent(w, rc, "review", review); err != nil {
				return
			}
		}
	}
}

func (s *Server) handleGetReview(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		jsonError(w, "invalid pr number", http.StatusBadRequest)
		return
	}
	jsonOK(w, s.orchestrator.Latest(owner, repo, number))
}

const defaultHistoryLimit = 50

func (s *Server) handleActiveReviews(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, s.orchestrator.Active())
}

func (s *Server) handleReviewHistory(w http.ResponseWriter, r *http.Request) {
	limit := defaultHistoryLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			jsonError(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = n
	}

	sessions, err := s.store.ListRecentReviewSessions(r.Context(), limit)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, sess := range sessions {
		sess.OpencodeSessionPath = s.orchestrator.SessionPath(sess.OpencodeSessionID)
	}
	jsonOK(w, sessions)
}

func (s *Server) handleListReviews(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		jsonError(w, "invalid pr number", http.StatusBadRequest)
		return
	}

	sessions, err := s.store.ListReviewSessions(r.Context(), owner, repo, number)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type sessionWithConcerns struct {
		Session  *store.ReviewSession   `json:"session"`
		Concerns []*store.ReviewConcern `json:"concerns"`
	}

	out := make([]sessionWithConcerns, 0, len(sessions))
	for _, sess := range sessions {
		concerns, err := s.store.ListConcerns(r.Context(), sess.ID)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, sessionWithConcerns{Session: sess, Concerns: concerns})
	}

	jsonOK(w, out)
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
