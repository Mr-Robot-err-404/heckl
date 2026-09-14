package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const (
	jobSweepWorktrees = "sweep_worktrees"
	sweepInterval     = 12 * time.Hour
	jobTick           = 15 * time.Minute
	jobStartupDelay   = 30 * time.Second
)

type job struct {
	name     string
	interval time.Duration
	run      func(context.Context) (string, error)
}

func (s *Server) jobs() []job {
	return []job{{
		name:     jobSweepWorktrees,
		interval: sweepInterval,
		run:      s.sweepWorktrees,
	}}
}

func (s *Server) StartJobs(ctx context.Context) {
	go func() {
		select {
		case <-time.After(jobStartupDelay):
		case <-ctx.Done():
			return
		}

		ticker := time.NewTicker(jobTick)
		defer ticker.Stop()

		for {
			s.runDueJobs(ctx)
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *Server) runDueJobs(ctx context.Context) {
	for _, j := range s.jobs() {
		last, err := s.store.LastJobRun(ctx, j.name)
		if err != nil {
			slog.Error("jobs: read last run failed", "job", j.name, "err", err)
			continue
		}
		if !last.IsZero() && time.Since(last) < j.interval {
			continue
		}

		start := time.Now()
		detail, err := j.run(ctx)
		took := time.Since(start)

		status := "ok"
		if err != nil {
			status = "error"
			detail = err.Error()
			slog.Error("jobs: run failed", "job", j.name, "err", err, "duration_ms", took.Milliseconds())
		} else {
			slog.Info("jobs: run complete", "job", j.name, "detail", detail, "duration_ms", took.Milliseconds())
		}

		if err := s.store.RecordJobRun(ctx, j.name, status, detail, took); err != nil {
			slog.Error("jobs: record run failed", "job", j.name, "err", err)
		}
	}
}

func (s *Server) sweepWorktrees(ctx context.Context) (string, error) {
	repos, err := s.store.ListRepos(ctx)
	if err != nil {
		return "", err
	}

	pinned := s.pinnedPRs(ctx)
	swept, skipped := 0, 0

	for _, repo := range repos {
		removed, err := s.sweepRepo(ctx, repo.Owner, repo.Name, pinned)
		if err != nil {
			skipped++
			slog.Error("jobs: sweep repo failed", "owner", repo.Owner, "repo", repo.Name, "err", err)
			continue
		}
		swept += removed
	}
	return fmt.Sprintf("removed %d worktrees across %d repos, %d skipped", swept, len(repos)-skipped, skipped), nil
}
