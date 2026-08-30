package worktree

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

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
	return filepath.Join(m.stateDir, "repos", owner, repo+".git")
}

func (m *Manager) worktreePath(owner, repo string, prNumber int) string {
	return filepath.Join(m.stateDir, "worktrees", owner, repo, strconv.Itoa(prNumber))
}

func (m *Manager) EnsureWorktree(owner, repo string, prNumber int, headSHA string) (string, error) {
	unlock := m.lockFor(owner + "/" + repo)
	defer unlock()

	bare := m.repoPath(owner, repo)
	if err := m.ensureClone(owner, repo, bare); err != nil {
		return "", fmt.Errorf("worktree: clone %s/%s: %w", owner, repo, err)
	}

	prRef := fmt.Sprintf("refs/pull/%d/head", prNumber)
	if err := runGit(bare, "fetch", "origin", prRef); err != nil {
		return "", fmt.Errorf("worktree: fetch %s #%d: %w", owner+"/"+repo, prNumber, err)
	}

	wt := m.worktreePath(owner, repo, prNumber)
	info, err := os.Stat(wt)
	switch {
	case os.IsNotExist(err):
		if err := os.MkdirAll(filepath.Dir(wt), 0o755); err != nil {
			return "", err
		}
		if err := runGit(bare, "worktree", "add", "--detach", wt, headSHA); err != nil {
			return "", fmt.Errorf("worktree: add %s: %w", wt, err)
		}
	case err != nil:
		return "", err
	case !info.IsDir():
		return "", fmt.Errorf("worktree: %s exists and is not a directory", wt)
	default:
		if err := runGit(wt, "checkout", "--detach", headSHA); err != nil {
			return "", fmt.Errorf("worktree: checkout %s at %s: %w", wt, headSHA, err)
		}
	}

	return wt, nil
}

func (m *Manager) ensureClone(owner, repo, bare string) error {
	if _, err := os.Stat(bare); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(bare), 0o755); err != nil {
		return err
	}

	url := fmt.Sprintf("git@github.com:%s/%s.git", owner, repo)
	return runGit("", "clone", "--bare", url, bare)
}
