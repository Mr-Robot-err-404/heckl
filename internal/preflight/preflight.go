package preflight

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Mr-Robot-err-404/heckl/internal/config"
	"github.com/Mr-Robot-err-404/heckl/internal/ghauth"
	"github.com/Mr-Robot-err-404/heckl/internal/store"
)

type Status int

const (
	OK Status = iota
	Warn
	Fail
)

type Check struct {
	Name   string
	Status Status
	Detail string
	Hint   string
}

func Run(ctx context.Context, cfg *config.Config) []Check {
	checks := []Check{
		binary("git", true, "install git - clones and worktrees are done with it, not delegated to an agent"),
		binary("opencode", true, "install opencode: https://opencode.ai - reviews cannot run without it"),
		agentFiles(cfg),
		database(cfg),
		token(cfg),
	}
	if cfg.Tmux.Enabled {
		checks = append(checks,
			binary("tmux", false, "install tmux, or set tmux.enabled = false - line picking will not open sessions without it"),
			binary(cfg.Tmux.Editor, false, fmt.Sprintf("install %s, or point tmux.editor at an editor you have", cfg.Tmux.Editor)),
		)
	}
	return checks
}

func Failed(checks []Check) bool {
	for _, c := range checks {
		if c.Status == Fail {
			return true
		}
	}
	return false
}

func binary(name string, required bool, hint string) Check {
	path, err := exec.LookPath(name)
	if err == nil {
		return Check{Name: name, Status: OK, Detail: path}
	}
	status := Warn
	if required {
		status = Fail
	}
	return Check{Name: name, Status: status, Detail: "not on PATH", Hint: hint}
}

func agentFiles(cfg *config.Config) Check {
	dir := filepath.Join(cfg.OpenCode.ProjectDir, ".opencode")
	agent := filepath.Join(dir, "agents", "heckl.md")
	if _, err := os.Stat(agent); err != nil {
		return Check{
			Name:   "opencode agents",
			Status: Fail,
			Detail: "heckl.md not found under " + dir,
			Hint:   "point opencode.project_dir at the heckl checkout - .opencode/agents and .opencode/tools live there",
		}
	}
	return Check{Name: "opencode agents", Status: OK, Detail: dir}
}

func database(cfg *config.Config) Check {
	path := cfg.Server.Database
	if _, err := os.Stat(path); err != nil {
		return Check{
			Name:   "database",
			Status: Fail,
			Detail: "missing: " + path,
			Hint:   "run `heckl setup`, or `heckl migrate up`",
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return Check{Name: "database", Status: Fail, Detail: err.Error()}
	}
	defer db.Close()

	pending, err := store.PendingMigrations(db)
	if err != nil {
		return Check{Name: "database", Status: Fail, Detail: err.Error(), Hint: "run `heckl migrate up`"}
	}
	if pending > 0 {
		return Check{
			Name:   "database",
			Status: Fail,
			Detail: fmt.Sprintf("%d migration(s) pending", pending),
			Hint:   "run `heckl migrate up`",
		}
	}
	return Check{Name: "database", Status: OK, Detail: path}
}

func token(cfg *config.Config) Check {
	tok, err := ghauth.Resolve(ghauth.Options{
		TokenFile: cfg.GitHub.TokenFile,
		UseGHCLI:  cfg.GitHub.UseGHCLI,
	})
	if err != nil {
		return Check{
			Name:   "github auth",
			Status: Fail,
			Detail: "no token",
			Hint:   "run `heckl setup` to sign in, or export GITHUB_TOKEN",
		}
	}
	return Check{Name: "github auth", Status: OK, Detail: string(tok.Source)}
}
