package ghauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrNoToken = errors.New("no github token")

type Source string

const (
	SourceEnv    Source = "GITHUB_TOKEN"
	SourceFile   Source = "token file"
	SourceGHCLI  Source = "gh cli"
	SourceDevice Source = "device flow"
)

type Token struct {
	Value  string
	Source Source
}

type Options struct {
	TokenFile string
	UseGHCLI  bool
}

func Resolve(opts Options) (*Token, error) {
	if v := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); v != "" {
		return &Token{Value: v, Source: SourceEnv}, nil
	}
	if v := readTokenFile(opts.TokenFile); v != "" {
		return &Token{Value: v, Source: SourceFile}, nil
	}
	if opts.UseGHCLI {
		if v := ghCLIToken(); v != "" {
			return &Token{Value: v, Source: SourceGHCLI}, nil
		}
	}
	return nil, ErrNoToken
}

func readTokenFile(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func ghCLIToken() string {
	if _, err := exec.LookPath("gh"); err != nil {
		return ""
	}
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func SaveToken(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("ghauth: create %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return fmt.Errorf("ghauth: write %s: %w", path, err)
	}
	return nil
}

func Login(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ghauth: verify token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("ghauth: token rejected by github")
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("ghauth: verify token: github returned %d", resp.StatusCode)
	}

	var user struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("ghauth: decode user: %w", err)
	}
	return user.Login, nil
}
