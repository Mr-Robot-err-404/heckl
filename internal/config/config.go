package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	path string

	Server   Server   `toml:"server"`
	GitHub   GitHub   `toml:"github"`
	OpenCode OpenCode `toml:"opencode"`
	Tmux     Tmux     `toml:"tmux"`
}

type Server struct {
	Addr       string `toml:"addr"`
	DataDir    string `toml:"data_dir"`
	Database   string `toml:"database"`
	LogLevel   string `toml:"log_level"`
	RemoteHost string `toml:"remote_host"`
}

type GitHub struct {
	OAuthClientID string `toml:"oauth_client_id"`
	TokenFile     string `toml:"token_file"`
	UseGHCLI      bool   `toml:"use_gh_cli"`
}

type OpenCode struct {
	URL        string `toml:"url"`
	ProjectDir string `toml:"project_dir"`
	Spawn      bool   `toml:"spawn"`
}

type Tmux struct {
	Enabled    bool   `toml:"enabled"`
	Editor     string `toml:"editor"`
	MaxWindows int    `toml:"max_windows"`
}

func Defaults() Config {
	return Config{
		Server: Server{
			Addr:     ":7331",
			DataDir:  filepath.Join(DataHome(), "data"),
			Database: filepath.Join(DataHome(), "pr-review.db"),
			LogLevel: "info",
		},
		GitHub: GitHub{
			TokenFile: filepath.Join(Home(), "token"),
			UseGHCLI:  true,
		},
		OpenCode: OpenCode{
			URL:   "http://127.0.0.1:4420",
			Spawn: true,
		},
		Tmux: Tmux{
			Enabled:    true,
			Editor:     "nvim",
			MaxWindows: 20,
		},
	}
}

func Path() string {
	if p := strings.TrimSpace(os.Getenv("PR_REVIEW_CONFIG")); p != "" {
		return p
	}
	return filepath.Join(Home(), "config.toml")
}

func Exists() bool {
	_, err := os.Stat(Path())
	return err == nil
}

func Load() (*Config, error) {
	path := Path()
	cfg := Defaults()
	cfg.path = path

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("no config at %s — run `pr-review setup`", path)
	}
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	cfg.applyEnv()
	if err := cfg.normalise(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Path() string { return c.path }

func (c *Config) applyEnv() {
	if v := strings.TrimSpace(os.Getenv("REMOTE_HOST")); v != "" {
		c.Server.RemoteHost = v
	}
	if v := strings.TrimSpace(os.Getenv("LOG_LEVEL")); v != "" {
		c.Server.LogLevel = v
	}
}

func (c *Config) normalise() error {
	if c.OpenCode.ProjectDir == "" {
		return fmt.Errorf("config: opencode.project_dir is empty — it must point at the directory holding .opencode/")
	}
	for _, p := range []*string{&c.Server.DataDir, &c.Server.Database, &c.OpenCode.ProjectDir, &c.GitHub.TokenFile} {
		expanded, err := expand(*p)
		if err != nil {
			return err
		}
		*p = expanded
	}
	if c.Tmux.MaxWindows <= 0 {
		c.Tmux.MaxWindows = Defaults().Tmux.MaxWindows
	}
	if c.Tmux.Editor == "" {
		c.Tmux.Editor = Defaults().Tmux.Editor
	}
	return nil
}

func expand(path string) (string, error) {
	if path == "" {
		return path, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("config: expand %q: %w", path, err)
		}
		path = filepath.Join(home, path[2:])
	}
	return filepath.Abs(path)
}

func Home() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "pr-review")
	}
	return ".pr-review"
}

func DataHome() string {
	if dir := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); dir != "" {
		return filepath.Join(dir, "pr-review")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "pr-review")
	}
	return ".pr-review"
}
