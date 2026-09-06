package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/harrylawton/pr-review/internal/checkout"
	"github.com/harrylawton/pr-review/internal/config"
	"github.com/harrylawton/pr-review/internal/ghauth"
	"github.com/harrylawton/pr-review/internal/github"
	"github.com/harrylawton/pr-review/internal/logs"
	"github.com/harrylawton/pr-review/internal/opencode"
	"github.com/harrylawton/pr-review/internal/orchestrator"
	"github.com/harrylawton/pr-review/internal/preflight"
	"github.com/harrylawton/pr-review/internal/reviewer"
	"github.com/harrylawton/pr-review/internal/server"
	"github.com/harrylawton/pr-review/internal/store"
)

func runServe() {
	cfg, err := config.Load()
	if err != nil {
		fatal(err.Error())
	}
	slog.SetDefault(slog.New(logs.New(os.Stdout, logLevel(cfg.Server.LogLevel))))

	checks := preflight.Run(context.Background(), cfg)
	if preflight.Failed(checks) {
		printChecks(checks)
		fatal("preflight failed - fix the above, or run `pr-review setup`")
	}
	for _, c := range checks {
		if c.Status != preflight.OK {
			slog.Warn("preflight", "check", c.Name, "detail", c.Detail, "hint", c.Hint)
		}
	}

	tok, err := ghauth.Resolve(ghauth.Options{
		TokenFile: cfg.GitHub.TokenFile,
		UseGHCLI:  cfg.GitHub.UseGHCLI,
	})
	if err != nil {
		fatal("github: no token - run `pr-review setup`")
	}
	slog.Info("github token loaded", "source", string(tok.Source))

	db, err := store.Open(cfg.Server.Database)
	if err != nil {
		fatal(err.Error())
	}
	defer db.Close()

	oc, err := opencode.Setup(opencode.SetupOptions{
		BaseURL:    cfg.OpenCode.URL,
		ProjectDir: cfg.OpenCode.ProjectDir,
		Spawn:      cfg.OpenCode.Spawn,
	})
	if err != nil {
		fatal("opencode setup failed: " + err.Error())
	}

	gh := github.New(tok.Value)
	gh.Warm()
	co := checkout.New(cfg.Server.DataDir)
	rev := reviewer.New(oc, co, db)
	sessionPath := func(id string) string { return opencode.SessionPath(cfg.OpenCode.ProjectDir, id) }
	orc := orchestrator.New(context.Background(), slog.Default(), gh, rev, db, sessionPath)

	if cfg.Server.RemoteHost != "" {
		slog.Info("remote host override set - non-local clients will ssh here", "host", cfg.Server.RemoteHost)
	}

	srv := server.New(gh, db, oc, orc, co, cfg)

	dist, err := fs.Sub(static, "dist")
	if err != nil {
		fatal(err.Error())
	}
	srv.Static(http.FS(dist))

	slog.Info("listening", "addr", cfg.Server.Addr)
	if err := http.ListenAndServe(cfg.Server.Addr, srv); err != nil {
		fatal(err.Error())
	}
}

func logLevel(level string) slog.Level {
	if strings.EqualFold(level, "debug") {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}
