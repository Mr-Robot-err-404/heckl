package opencode

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const startupPollInterval = 200 * time.Millisecond
const startupTimeout = 15 * time.Second

var requiredAgents = []string{"pr-reviewer.md"}

func Setup(baseURL, projectDir string) (*Client, error) {
	if err := checkAgentFiles(projectDir); err != nil {
		return nil, fmt.Errorf("opencode: setup: %w", err)
	}

	c := New(baseURL)
	if healthy, _, err := c.Health(); err == nil && healthy {
		return c, nil
	}

	if err := spawnServer(baseURL, projectDir); err != nil {
		return nil, fmt.Errorf("opencode: setup: spawn server: %w", err)
	}

	if err := waitHealthy(c, startupTimeout); err != nil {
		return nil, fmt.Errorf("opencode: setup: %w", err)
	}

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
