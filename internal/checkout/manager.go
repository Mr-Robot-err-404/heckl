package checkout

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

type Handle struct {
	Path    string
	release func()
}

func (h *Handle) Release() {
	if h == nil || h.release == nil {
		return
	}
	h.release()
	h.release = nil
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

func (m *Manager) Acquire(ctx context.Context, owner, repo string, prNumber int, headSHA string) (*Handle, error) {
	unlock := m.lockFor(owner + "/" + repo)

	path, err := m.checkout(ctx, owner, repo, prNumber, headSHA)
	if err != nil {
		unlock()
		return nil, err
	}

	return &Handle{Path: path, release: unlock}, nil
}

func (m *Manager) checkout(ctx context.Context, owner, repo string, prNumber int, headSHA string) (string, error) {
	path := m.repoPath(owner, repo)
	if err := m.ensureClone(ctx, owner, repo, path); err != nil {
		return "", fmt.Errorf("checkout: clone %s/%s: %w", owner, repo, err)
	}

	prRef := fmt.Sprintf("refs/pull/%d/head", prNumber)
	if err := runGit(ctx, path, "fetch", "--no-tags", "origin", prRef); err != nil {
		return "", fmt.Errorf("checkout: fetch %s/%s #%d: %w", owner, repo, prNumber, err)
	}

	if err := runGit(ctx, path, "checkout", "--detach", "--force", headSHA); err != nil {
		return "", fmt.Errorf("checkout: %s at %s: %w", path, headSHA, err)
	}

	return path, nil
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
