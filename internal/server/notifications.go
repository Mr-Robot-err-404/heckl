package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/Mr-Robot-err-404/heckl/internal/github"
)

const notificationFetchConcurrency = 16

var notificationReasons = map[string]bool{
	"assign":           true,
	"author":           true,
	"comment":          true,
	"mention":          true,
	"review_requested": true,
	"team_mention":     true,
}

type notificationRow struct {
	ID            string `json:"id"`
	Reason        string `json:"reason"`
	Title         string `json:"title"`
	Owner         string `json:"owner"`
	Repo          string `json:"repo"`
	PRNumber      int    `json:"prNumber"`
	UpdatedAt     string `json:"updatedAt"`
	State         string `json:"state,omitempty"`
	ActivityActor string `json:"activityActor,omitempty"`
	ActivityBody  string `json:"activityBody,omitempty"`
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

	notifications, err := s.gh.ListNotifications(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	out := make([]notificationRow, 0, len(notifications))
	byID := make(map[string]github.Notification, len(notifications))
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
		byID[n.ID] = n
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, notificationFetchConcurrency)
	for i := range out {
		wg.Add(1)
		go func(row *notificationRow) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			pr, err := s.gh.GetPR(r.Context(), row.Owner, row.Repo, row.PRNumber)
			if err != nil {
				slog.Warn("notifications: get pr state failed", "owner", row.Owner, "repo", row.Repo, "number", row.PRNumber, "err", err)
			} else {
				row.State = notificationPRState(pr.State, pr.Draft, pr.MergedAt != nil)
			}

			activity, err := s.gh.GetNotificationActivity(r.Context(), byID[row.ID].Subject.LatestCommentURL)
			if err != nil {
				slog.Warn("notifications: get activity failed", "thread", row.ID, "err", err)
			} else if activity != nil {
				row.ActivityActor = activity.User.Login
				row.ActivityBody = strings.TrimSpace(activity.Body)
			}
		}(&out[i])
	}
	wg.Wait()
	jsonOK(w, out)
}

func notificationPRState(state string, draft, merged bool) string {
	if merged {
		return "merged"
	}
	if draft {
		return "draft"
	}
	return strings.ToLower(state)
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

	s.gh.MarkThreadsRead(r.Context(), body.IDs)
	slog.Info("notifications marked read", "count", len(body.IDs))
	jsonOK(w, struct {
		Read int `json:"read"`
	}{Read: len(body.IDs)})
}
