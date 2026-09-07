package checkout

import (
	"context"
	"errors"
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

var ErrPathMissing = errors.New("checkout: path not found at revision")

// FileAt returns the contents of path as of sha. ref is the branch the sha is
// expected to be reachable from, fetched only when the clone does not already
// have the commit.
func (m *Manager) FileAt(ctx context.Context, owner, repo, sha, ref, path string) ([]byte, error) {
	if sha == "" || path == "" {
		return nil, fmt.Errorf("checkout: file at: empty sha or path")
	}

	unlock := m.lockFor(owner + "/" + repo)
	defer unlock()

	repoPath := m.repoPath(owner, repo)
	if err := m.ensureClone(ctx, owner, repo, repoPath); err != nil {
		return nil, fmt.Errorf("checkout: clone %s/%s: %w", owner, repo, err)
	}

	if !hasCommit(ctx, repoPath, sha) && ref != "" {
		if err := runGit(ctx, repoPath, "fetch", "--no-tags", "origin", ref); err != nil {
			return nil, fmt.Errorf("checkout: fetch %s %s: %w", repo, ref, err)
		}
	}

	out, err := gitBytes(ctx, repoPath, "show", sha+":"+path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s@%s", ErrPathMissing, path, sha)
	}
	return out, nil
}

func hasCommit(ctx context.Context, repoPath, sha string) bool {
	_, err := gitOutput(ctx, repoPath, "cat-file", "-e", sha+"^{commit}")
	return err == nil
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
