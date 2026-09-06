package opencode

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const startupPollInterval = 200 * time.Millisecond
const startupTimeout = 15 * time.Second

var requiredAgents = []string{"pr-reviewer.md"}

type SetupOptions struct {
	BaseURL    string
	ProjectDir string
	Spawn      bool
}

func Setup(opts SetupOptions) (*Client, error) {
	if err := checkAgentFiles(opts.ProjectDir); err != nil {
		return nil, fmt.Errorf("opencode: setup: %w", err)
	}

	c := New(opts.BaseURL)
	if healthy, version, err := c.Health(); err == nil && healthy {
		slog.Info("opencode already running", "url", c.baseURL, "version", version)
		return c, nil
	}
	if !opts.Spawn {
		return nil, fmt.Errorf("opencode: nothing responding at %s and opencode.spawn is false - start `opencode serve` yourself", c.baseURL)
	}

	slog.Info("opencode not responding - spawning", "url", c.baseURL, "dir", opts.ProjectDir)
	if err := spawnServer(c.baseURL, opts.ProjectDir); err != nil {
		return nil, fmt.Errorf("opencode: setup: spawn server: %w", err)
	}

	if err := waitHealthy(c, startupTimeout); err != nil {
		return nil, fmt.Errorf("opencode: setup: %w", err)
	}

	_, version, _ := c.Health()
	slog.Info("opencode ready", "url", c.baseURL, "version", version)
	return c, nil
}

func checkAgentFiles(projectDir string) error {
	dir := filepath.Join(projectDir, ".opencode", "agents")
	for _, name := range requiredAgents {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("required agent file missing: %s: %w", path, err)
		}
	}
	return nil
}

func spawnServer(baseURL, projectDir string) error {
	port, err := portFromURL(baseURL)
	if err != nil {
		return err
	}

	cmd := exec.Command("opencode", "serve", "--port", port)
	cmd.Dir = projectDir
	cmd.Stderr = os.Stderr
	return cmd.Start()
}

func waitHealthy(c *Client, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		healthy, _, err := c.Health()
		if err == nil && healthy {
			return nil
		}
		lastErr = err
		time.Sleep(startupPollInterval)
	}
	return fmt.Errorf("server did not become healthy within %s: %w", timeout, lastErr)
}

func portFromURL(baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base url %q: %w", baseURL, err)
	}
	port := u.Port()
	if port == "" {
		return "", fmt.Errorf("base url %q has no explicit port", baseURL)
	}
	return port, nil
}
