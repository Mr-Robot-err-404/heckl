package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Mr-Robot-err-404/heckl/internal/checkout"
	"github.com/Mr-Robot-err-404/heckl/internal/orchestrator"
	"github.com/Mr-Robot-err-404/heckl/internal/store"
	"github.com/Mr-Robot-err-404/heckl/internal/tmux"
)

const reapTimeout = 2 * time.Minute

func (s *Server) pinnedPRs(ctx context.Context) map[string]bool {
	pinned := map[string]bool{}

	for _, review := range s.orchestrator.Active() {
		pinned[orchestrator.PRKey(review.Owner, review.Repo, review.PRNumber)] = true
	}

	rows, err := s.store.ListTmuxSessions(ctx)
	if err != nil {
		slog.Error("reap: list tmux sessions failed", "err", err)
		return pinned
	}
	if len(rows) == 0 || !tmux.Installed() {
		return pinned
	}

	live := s.tmux.LiveSessions(ctx)
	for _, row := range rows {
		if live[row.Name] {
			pinned[orchestrator.PRKey(row.Owner, row.Repo, int(row.PrNumber))] = true
		}
	}
	return pinned
}

func (s *Server) sessionAlive(ctx context.Context, ref store.TmuxRef) bool {
	if !tmux.Installed() {
		return false
	}
	name := tmux.SessionName(fmt.Sprintf("%s/%s/%d", ref.Owner, ref.Repo, ref.PRNumber))
	return s.tmux.HasSession(ctx, name)
}

func (s *Server) reviewing(owner, repo string, prNumber int) bool {
	key := orchestrator.PRKey(owner, repo, prNumber)
	for _, review := range s.orchestrator.Active() {
		if orchestrator.PRKey(review.Owner, review.Repo, review.PRNumber) == key {
			return true
		}
	}
	return false
}

func (s *Server) removeWorktrees(ctx context.Context, owner, repo string, trees []checkout.Worktree) int {
	removed := 0
	for _, tree := range trees {
		if s.reviewing(owner, repo, tree.PRNumber) {
			slog.Debug("reap: review started mid-sweep", "worktree", tree.Path)
			continue
		}
		if err := s.checkout.Remove(ctx, owner, repo, tree.Path); err != nil {
			slog.Error("reap: remove worktree failed", "worktree", tree.Path, "err", err)
			continue
		}
		slog.Info("reap: removed worktree", "worktree", tree.Path)
		removed++
	}
	return removed
}

// reapClosed requires open to be the complete set of open PR numbers.
func (s *Server) reapClosed(owner, repo string, open map[int]bool) {
	ctx, cancel := context.WithTimeout(context.Background(), reapTimeout)
	defer cancel()

	trees, err := s.checkout.Worktrees(owner, repo)
	if err != nil {
		slog.Error("reap: list worktrees failed", "owner", owner, "repo", repo, "err", err)
		return
	}
	if len(trees) == 0 {
		return
	}

	pinned := s.pinnedPRs(ctx)
	var stale []checkout.Worktree
	for _, tree := range trees {
		if open[tree.PRNumber] {
			continue
		}
		if pinned[orchestrator.PRKey(owner, repo, tree.PRNumber)] {
			slog.Debug("reap: worktree pinned", "worktree", tree.Path)
			continue
		}
		stale = append(stale, tree)
	}
	if len(stale) == 0 {
		return
	}

	start := time.Now()
	removed := s.removeWorktrees(ctx, owner, repo, stale)
	slog.Info("reap: closed prs swept",
		"owner", owner, "repo", repo,
		"removed", removed, "duration_ms", time.Since(start).Milliseconds())
}

func (s *Server) reapSessions(refs []store.TmuxRef) {
	ctx, cancel := context.WithTimeout(context.Background(), reapTimeout)
	defer cancel()

	pinned := s.pinnedPRs(ctx)
	removed := 0
	for _, ref := range refs {
		if pinned[orchestrator.PRKey(ref.Owner, ref.Repo, ref.PRNumber)] || s.sessionAlive(ctx, ref) {
			slog.Debug("reap: pr pinned", "owner", ref.Owner, "repo", ref.Repo, "pr", ref.PRNumber)
			continue
		}

		trees, err := s.checkout.Worktrees(ref.Owner, ref.Repo)
		if err != nil {
			slog.Error("reap: list worktrees failed", "owner", ref.Owner, "repo", ref.Repo, "err", err)
			continue
		}

		var dirs []checkout.Worktree
		for _, tree := range trees {
			if tree.PRNumber == ref.PRNumber {
				dirs = append(dirs, tree)
			}
		}
		removed += s.removeWorktrees(ctx, ref.Owner, ref.Repo, dirs)
	}
	slog.Info("reap: killed sessions swept", "removed", removed)
}
