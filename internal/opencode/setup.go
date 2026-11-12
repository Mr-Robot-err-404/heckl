package opencode

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Mr-Robot-err-404/heckl/internal/bundle"
)

const startupPollInterval = 200 * time.Millisecond
const startupTimeout = 15 * time.Second

type SetupOptions struct {
	BaseURL      string
	ProjectDir   string
	Spawn        bool
	PasswordFile string
}

func Setup(opts SetupOptions) (*Client, error) {
	if err := checkBundleFiles(opts.ProjectDir); err != nil {
		return nil, fmt.Errorf("opencode: setup: %w", err)
	}

	c := New(opts.BaseURL)
	directory, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		return nil, err
	}
	c.directory = directory
	if err := c.LoadPassword(opts.PasswordFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("opencode: load password: %w", err)
	}
	healthy, version, healthErr := c.Health()
	if healthErr == nil {
		if !healthy {
			return nil, fmt.Errorf("opencode: %s runs unsupported version %q - Heckl requires OpenCode V2", c.baseURL, version)
		}
		if err := c.decode("GET", c.locationPath("/api/agent"), nil, nil); err != nil {
			return nil, fmt.Errorf("opencode: authenticate: %w - set OPENCODE_PASSWORD or opencode.password_file to the server password", err)
		}
		slog.Info("opencode already running", "url", c.baseURL, "version", version)
		return c, nil
	}
	var connectionErr *url.Error
	if !errors.As(healthErr, &connectionErr) {
		return nil, fmt.Errorf("opencode: %s is not an accessible V2 server: %w - check the server version and OPENCODE_PASSWORD or opencode.password_file", c.baseURL, healthErr)
	}
	if !opts.Spawn {
		return nil, fmt.Errorf("opencode: nothing responding at %s and opencode.spawn is false - start `opencode serve` yourself", c.baseURL)
	}
	if c.password == "" {
		if opts.PasswordFile == "" {
			return nil, fmt.Errorf("opencode: set OPENCODE_PASSWORD or opencode.password_file before spawning a server")
		}
		password := make([]byte, 32)
		if _, err := rand.Read(password); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(opts.PasswordFile), 0o700); err != nil {
			return nil, err
		}
		file, err := os.OpenFile(opts.PasswordFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, os.ErrExist) {
			if err := c.LoadPassword(opts.PasswordFile); err != nil {
				return nil, err
			}
		} else {
			if err != nil {
				return nil, err
			}
			c.password = hex.EncodeToString(password)
			_, writeErr := file.WriteString(c.password + "\n")
			closeErr := file.Close()
			if err := errors.Join(writeErr, closeErr); err != nil {
				return nil, err
			}
		}
	}
	slog.Info("opencode not responding - spawning", "url", c.baseURL, "dir", opts.ProjectDir)
	if err := spawnServer(c.baseURL, opts.ProjectDir, c.password); err != nil {
		return nil, fmt.Errorf("opencode: setup: spawn server: %w", err)
	}

	if err := waitHealthy(c, startupTimeout); err != nil {
		return nil, fmt.Errorf("opencode: setup: %w", err)
	}

	_, version, _ = c.Health()
	slog.Info("opencode ready", "url", c.baseURL, "version", version)
	return c, nil
}

func checkBundleFiles(projectDir string) error {
	agents, err := bundle.Agents()
	if err != nil {
		return err
	}
	plugins, err := bundle.Plugins()
	if err != nil {
		return err
	}
	for _, group := range []struct {
		dir   string
		names []string
	}{{"agents", agents}, {"plugins", plugins}} {
		for _, name := range group.names {
			path := filepath.Join(projectDir, ".opencode", group.dir, name)
			if _, err := os.Stat(path); err != nil {
				return fmt.Errorf("required bundle file missing: %s - run `heckl setup`: %w", path, err)
			}
		}
	}
	return nil
}

func spawnServer(baseURL, projectDir, password string) error {
	port, err := portFromURL(baseURL)
	if err != nil {
		return err
	}

	cmd := exec.Command("opencode", "serve", "--port", port)
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "OPENCODE_PASSWORD="+password)
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
