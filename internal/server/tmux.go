package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/harrylawton/pr-review/internal/tmux"
)

const maxTmuxWindows = 20

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
	if s.remoteHost != "" {
		host = s.remoteHost
	}
	return fmt.Sprintf("ssh -t %s %q", host, attach)
}

func (s *Server) handleTmuxSession(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		jsonError(w, "invalid pr number", http.StatusBadRequest)
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
	if len(body.Picks) > maxTmuxWindows {
		jsonError(w, fmt.Sprintf("too many files — %d max", maxTmuxWindows), http.StatusBadRequest)
		return
	}

	pr, err := s.gh.GetPR(owner, repo, number)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	handle, err := s.checkout.Acquire(r.Context(), owner, repo, number, pr.HeadSHA())
	if err != nil {
		slog.Error("tmux: checkout failed", "owner", owner, "repo", repo, "pr", number, "err", err)
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer handle.Release()

	var windows []tmux.Window
	var opened, skipped []string
	for _, p := range body.Picks {
		path := filepath.Clean(strings.TrimPrefix(p.File, "/"))
		if path == "." || strings.HasPrefix(path, "..") {
			skipped = append(skipped, p.File)
			continue
		}
		if _, err := os.Stat(filepath.Join(handle.Path, path)); err != nil {
			skipped = append(skipped, p.File)
			continue
		}
		windows = append(windows, tmux.NvimWindow(path, p.Line))
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
	if err := s.tmux.CreateSession(r.Context(), name, handle.Path, windows); err != nil {
		slog.Error("tmux: create session failed", "session", name, "err", err)
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("tmux session created", "session", name, "windows", len(windows), "skipped", len(skipped))
	jsonOK(w, tmuxSessionResponse{
		Session: name,
		Attach:  s.attachCommand(r, name),
		Opened:  opened,
		Skipped: skipped,
	})
}
