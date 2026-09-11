package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Mr-Robot-err-404/heckl/internal/store"
	"github.com/Mr-Robot-err-404/heckl/internal/tmux"
)

type tmuxPick struct {
	File string `json:"file"`
	Line int    `json:"line,omitempty"`
}

type tmuxSessionResponse struct {
	Session string   `json:"session"`
	Attach  string   `json:"attach"`
	Opened  []string `json:"opened"`
	Skipped []string `json:"skipped,omitempty"`
}

func requestHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		return r.Host
	}
	return host
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) attachCommand(r *http.Request, session string) string {
	attach := fmt.Sprintf("tmux attach -t '%s'", session)

	host := requestHost(r)
	if isLoopback(host) {
		return attach
	}
	if s.cfg.Server.RemoteHost != "" {
		host = s.cfg.Server.RemoteHost
	}
	return fmt.Sprintf("ssh -t %s %q", host, attach)
}

type tmuxLiveResponse struct {
	Session  string `json:"session"`
	Attach   string `json:"attach"`
	Windows  int    `json:"windows"`
	Worktree string `json:"worktree"`
	HeadSHA  string `json:"headSha"`
}

func (s *Server) handleGetTmuxSession(w http.ResponseWriter, r *http.Request) {
	owner, repo, number, ok := prPath(w, r)
	if !ok {
		return
	}

	row, err := s.store.GetTmuxSession(r.Context(), owner, repo, number)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if row == nil {
		jsonOK(w, nil)
		return
	}

	if !tmux.Installed() || !s.tmux.HasSession(r.Context(), row.Name) {
		if err := s.store.DeleteTmuxSession(r.Context(), owner, repo, number); err != nil {
			slog.Error("tmux: reap stale session row failed", "session", row.Name, "err", err)
		}
		slog.Info("tmux: reaped stale session row", "session", row.Name)
		jsonOK(w, nil)
		return
	}

	jsonOK(w, tmuxLiveResponse{
		Session:  row.Name,
		Attach:   s.attachCommand(r, row.Name),
		Windows:  int(row.Windows),
		Worktree: row.Worktree,
		HeadSHA:  row.HeadSha,
	})
}

type tmuxRow struct {
	Owner     string `json:"owner"`
	Repo      string `json:"repo"`
	PRNumber  int    `json:"prNumber"`
	Session   string `json:"session"`
	Attach    string `json:"attach"`
	Windows   int    `json:"windows"`
	Worktree  string `json:"worktree"`
	HeadSHA   string `json:"headSha"`
	CreatedAt string `json:"createdAt"`
	Live      bool   `json:"live"`
}

func (s *Server) handleListTmuxSessions(w http.ResponseWriter, r *http.Request) {
	rows, err := s.store.ListTmuxSessions(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	live := map[string]bool{}
	if tmux.Installed() {
		live = s.tmux.LiveSessions(r.Context())
	}

	out := make([]tmuxRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, tmuxRow{
			Owner:     row.Owner,
			Repo:      row.Repo,
			PRNumber:  int(row.PrNumber),
			Session:   row.Name,
			Attach:    s.attachCommand(r, row.Name),
			Windows:   int(row.Windows),
			Worktree:  row.Worktree,
			HeadSHA:   row.HeadSha,
			CreatedAt: row.CreatedAt,
			Live:      live[row.Name],
		})
	}
	jsonOK(w, out)
}

func (s *Server) handleCleanupTmuxSessions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Sessions []struct {
			Owner    string `json:"owner"`
			Repo     string `json:"repo"`
			PRNumber int    `json:"prNumber"`
		} `json:"sessions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if len(body.Sessions) == 0 {
		jsonError(w, "no sessions selected", http.StatusBadRequest)
		return
	}

	refs := make([]store.TmuxRef, 0, len(body.Sessions))
	names := make([]string, 0, len(body.Sessions))
	for _, sess := range body.Sessions {
		ref := store.TmuxRef{Owner: sess.Owner, Repo: sess.Repo, PRNumber: sess.PRNumber}
		refs = append(refs, ref)

		row, err := s.store.GetTmuxSession(r.Context(), ref.Owner, ref.Repo, ref.PRNumber)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if row != nil {
			names = append(names, row.Name)
		}
	}

	var killed []string
	if tmux.Installed() {
		killed = s.tmux.KillSessions(r.Context(), names)
	}
	if err := s.store.DeleteTmuxSessions(r.Context(), refs); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("tmux sessions cleaned up", "removed", len(refs), "killed", len(killed))
	jsonOK(w, struct {
		Removed int `json:"removed"`
		Killed  int `json:"killed"`
	}{Removed: len(refs), Killed: len(killed)})
}

func (s *Server) handleTmuxSession(w http.ResponseWriter, r *http.Request) {
	owner, repo, number, ok := prPath(w, r)
	if !ok {
		return
	}

	if !s.cfg.Tmux.Enabled {
		jsonError(w, "tmux sessions are disabled in the server config", http.StatusServiceUnavailable)
		return
	}
	if !tmux.Installed() {
		jsonError(w, "tmux is not installed on the server", http.StatusServiceUnavailable)
		return
	}

	var body struct {
		Picks []tmuxPick `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if len(body.Picks) == 0 {
		jsonError(w, "no files selected", http.StatusBadRequest)
		return
	}
	if len(body.Picks) > s.cfg.Tmux.MaxWindows {
		jsonError(w, fmt.Sprintf("too many files - %d max", s.cfg.Tmux.MaxWindows), http.StatusBadRequest)
		return
	}

	pr, err := s.gh.GetPR(owner, repo, number)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	worktree, err := s.checkout.Worktree(r.Context(), owner, repo, number, pr.HeadSHA())
	if err != nil {
		slog.Error("tmux: checkout failed", "owner", owner, "repo", repo, "pr", number, "err", err)
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var windows []tmux.Window
	var opened, skipped []string
	for _, p := range body.Picks {
		path := filepath.Clean(strings.TrimPrefix(p.File, "/"))
		if path == "." || strings.HasPrefix(path, "..") {
			skipped = append(skipped, p.File)
			continue
		}
		if _, err := os.Stat(filepath.Join(worktree, path)); err != nil {
			skipped = append(skipped, p.File)
			continue
		}
		windows = append(windows, tmux.EditorWindow(s.cfg.Tmux.Editor, path, p.Line))
		opened = append(opened, path)
	}
	if len(windows) == 0 {
		jsonError(w, "none of the selected files exist at this commit", http.StatusUnprocessableEntity)
		return
	}

	name := tmux.SessionName(fmt.Sprintf("%s/%s/%d", owner, repo, number))
	if s.tmux.HasSession(r.Context(), name) {
		if err := s.tmux.KillSession(r.Context(), name); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := s.tmux.CreateSession(r.Context(), name, worktree, windows); err != nil {
		slog.Error("tmux: create session failed", "session", name, "err", err)
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.store.SaveTmuxSession(r.Context(), store.NewTmuxSession{
		Owner:    owner,
		Repo:     repo,
		PRNumber: number,
		Name:     name,
		Worktree: worktree,
		HeadSHA:  pr.HeadSHA(),
		Windows:  len(windows),
	}); err != nil {
		slog.Error("tmux: persist session failed", "session", name, "err", err)
	}

	slog.Info("tmux session created", "session", name, "windows", len(windows), "skipped", len(skipped))
	jsonOK(w, tmuxSessionResponse{
		Session: name,
		Attach:  s.attachCommand(r, name),
		Opened:  opened,
		Skipped: skipped,
	})
}
