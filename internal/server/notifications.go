package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

var notificationReasons = map[string]bool{
	"assign":           true,
	"author":           true,
	"comment":          true,
	"mention":          true,
	"review_requested": true,
	"team_mention":     true,
}

type notificationRow struct {
	ID        string `json:"id"`
	Reason    string `json:"reason"`
	Title     string `json:"title"`
	Owner     string `json:"owner"`
	Repo      string `json:"repo"`
	PRNumber  int    `json:"prNumber"`
	UpdatedAt string `json:"updatedAt"`
}

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	repos, err := s.store.ListRepos(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	known := make(map[string]bool, len(repos))
	for _, repo := range repos {
		known[strings.ToLower(repo.Owner+"/"+repo.Name)] = true
	}

	notifications, err := s.gh.ListNotifications()
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	out := make([]notificationRow, 0, len(notifications))
	for _, n := range notifications {
		if !n.IsPullRequest() || !notificationReasons[n.Reason] {
			continue
		}
		if !known[strings.ToLower(n.Repository.FullName)] {
			continue
		}
		number := n.PRNumber()
		if number == 0 {
			continue
		}
		out = append(out, notificationRow{
			ID:        n.ID,
			Reason:    n.Reason,
			Title:     n.Subject.Title,
			Owner:     n.Repository.Owner.Login,
			Repo:      n.Repository.Name,
			PRNumber:  number,
			UpdatedAt: n.UpdatedAt,
		})
	}
	jsonOK(w, out)
}

func (s *Server) handleReadNotifications(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if len(body.IDs) == 0 {
		jsonError(w, "no notifications selected", http.StatusBadRequest)
		return
	}

	s.gh.MarkThreadsRead(body.IDs)
	slog.Info("notifications marked read", "count", len(body.IDs))
	jsonOK(w, struct {
		Read int `json:"read"`
	}{Read: len(body.IDs)})
}
