package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/harrylawton/pr-review/internal/orchestrator"
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

	pr, err := s.gh.GetPR(owner, repo, number)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	diff, err := s.gh.GetPRDiff(owner, repo, number)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	review, err := s.orchestrator.Start(orchestrator.StartInput{
		Owner:    owner,
		Repo:     repo,
		PRNumber: number,
		HeadSHA:  pr.HeadSHA(),
		Title:    pr.Title,
		Body:     pr.Body,
		Diff:     string(diff),
	})
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
	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	sub, snapshot, err := s.orchestrator.Subscribe(r.PathValue("id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	defer s.orchestrator.Unsubscribe(sub)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	writeEvent(w, "snapshot", snapshot)
	flusher.Flush()

	if isTerminal(snapshot.Status) {
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
			flusher.Flush()
		case review, ok := <-sub.Ch:
			if !ok {
				return
			}
			writeEvent(w, "review", review)
			flusher.Flush()
			if isTerminal(review.Status) {
				return
			}
		}
	}
}

func (s *Server) handleGetReview(w http.ResponseWriter, r *http.Request) {
	review, ok := s.orchestrator.Get(r.PathValue("id"))
	if !ok {
		jsonError(w, "review not found", http.StatusNotFound)
		return
	}
	jsonOK(w, review)
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

func isTerminal(status orchestrator.Status) bool {
	return status == orchestrator.StatusDone || status == orchestrator.StatusError
}

func writeEvent(w http.ResponseWriter, event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
}
