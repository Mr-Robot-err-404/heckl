package checkout

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const shaLabelLen = 12

type Manager struct {
	stateDir string

	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func New(stateDir string) *Manager {
	return &Manager{
		stateDir: stateDir,
		locks:    make(map[string]*sync.Mutex),
	}
}

func (m *Manager) lockFor(key string) func() {
	m.mu.Lock()
	l, ok := m.locks[key]
	if !ok {
		l = &sync.Mutex{}
		m.locks[key] = l
	}
	m.mu.Unlock()

	l.Lock()
	return l.Unlock
}

func (m *Manager) repoPath(owner, repo string) string {
	return filepath.Join(m.stateDir, "repos", owner, repo)
}

func (m *Manager) worktreePath(owner, repo string, prNumber int, headSHA string) string {
	label := headSHA
	if len(label) > shaLabelLen {
		label = label[:shaLabelLen]
	}
	return filepath.Join(m.stateDir, "worktrees", owner, repo, fmt.Sprintf("%d-%s", prNumber, label))
}

// Worktree returns a working tree for the PR at headSHA, isolated from every
// other checkout. The repo lock is held only while shared state is mutated —
// the clone, the fetch, and the worktree registry — never for the caller's
// use of the returned path.
func (m *Manager) Worktree(ctx context.Context, owner, repo string, prNumber int, headSHA string) (string, error) {
	if headSHA == "" {
		return "", fmt.Errorf("checkout: %s/%s #%d: empty head sha", owner, repo, prNumber)
	}

	unlock := m.lockFor(owner + "/" + repo)
	defer unlock()

	repoPath := m.repoPath(owner, repo)
	if err := m.ensureClone(ctx, owner, repo, repoPath); err != nil {
		return "", fmt.Errorf("checkout: clone %s/%s: %w", owner, repo, err)
	}

	prRef := fmt.Sprintf("refs/pull/%d/head", prNumber)
	if err := runGit(ctx, repoPath, "fetch", "--no-tags", "origin", prRef); err != nil {
		return "", fmt.Errorf("checkout: fetch %s/%s #%d: %w", owner, repo, prNumber, err)
	}

	dir := m.worktreePath(owner, repo, prNumber, headSHA)
	if err := ensureWorktree(ctx, repoPath, dir, headSHA); err != nil {
		return "", fmt.Errorf("checkout: worktree %s/%s #%d: %w", owner, repo, prNumber, err)
	}
	return dir, nil
}

func (m *Manager) ensureClone(ctx context.Context, owner, repo, path string) error {
	if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	url := fmt.Sprintf("git@github.com:%s/%s.git", owner, repo)
	return runGit(ctx, "", "clone", "--filter=blob:none", "--no-checkout", url, path)
}

// ensureWorktree makes the registry and the filesystem agree before trusting
// either. A directory that exists but is not a worktree checked out at headSHA
// is torn down and rebuilt rather than reused.
func ensureWorktree(ctx context.Context, repoPath, dir, headSHA string) error {
	if err := runGit(ctx, repoPath, "worktree", "prune"); err != nil {
		return err
	}

	if worktreeIsAt(ctx, dir, headSHA) {
		return nil
	}

	if err := removeWorktree(ctx, repoPath, dir); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return err
	}
	return runGit(ctx, repoPath, "worktree", "add", "--detach", dir, headSHA)
}

func worktreeIsAt(ctx context.Context, dir, headSHA string) bool {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return false
	}
	head, err := gitOutput(ctx, dir, "rev-parse", "HEAD")
	return err == nil && head == headSHA
}

func removeWorktree(ctx context.Context, repoPath, dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return runGit(ctx, repoPath, "worktree", "prune")
	}

	if err := runGit(ctx, repoPath, "worktree", "remove", "--force", dir); err == nil {
		return nil
	}

	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return runGit(ctx, repoPath, "worktree", "prune")
}
