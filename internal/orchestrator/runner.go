package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/harrylawton/pr-review/internal/github"
	"github.com/harrylawton/pr-review/internal/reviewer"
)

type Runner struct {
	ctx      context.Context
	logger   *slog.Logger
	source   PRSource
	reviewer *reviewer.Reviewer
	publish  func(key string, review *Review)
	path     SessionPath

	mu       sync.RWMutex
	inFlight map[string]*Review
}

func newRunner(ctx context.Context, logger *slog.Logger, source PRSource, rev *reviewer.Reviewer, publish func(string, *Review), path SessionPath) *Runner {
	return &Runner{
		ctx:      ctx,
		logger:   logger,
		source:   source,
		reviewer: rev,
		publish:  publish,
		path:     path,
		inFlight: make(map[string]*Review),
	}
}

func (r *Runner) InFlight(key string) *Review {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if review, ok := r.inFlight[key]; ok {
		return review.clone()
	}
	return nil
}

func (r *Runner) Start(owner, repo string, prNumber int) (*Review, error) {
	key := PRKey(owner, repo, prNumber)

	r.mu.Lock()
	if existing, running := r.inFlight[key]; running {
		snapshot := existing.clone()
		r.mu.Unlock()
		return snapshot, nil
	}

	id, err := newID()
	if err != nil {
		r.mu.Unlock()
		return nil, err
	}

	review := newReview(id, owner, repo, prNumber)
	review.Status = StatusRunning
	review.startStage(StageFetch)
	review.setAgent(agentName, StatusRunning)
	r.inFlight[key] = review
	snapshot := review.clone()
	r.mu.Unlock()

	r.publish(key, snapshot)
	go r.run(key, owner, repo, prNumber)

	return snapshot, nil
}

func (r *Runner) run(key, owner, repo string, prNumber int) {
	log := r.logger.With("owner", owner, "repo", repo, "pr", prNumber)
	defer r.release(key)

	t := time.Now()
	pr, diff, err := r.fetch(owner, repo, prNumber)
	if err != nil {
		r.update(key, func(rv *Review) {
			rv.endStage(StageFetch, "", err)
			rv.finish(err)
		})
		log.Error("orchestrator: fetch failed", "err", err)
		return
	}

	headSHA := pr.HeadSHA()
	r.update(key, func(rv *Review) {
		rv.HeadSHA = headSHA
		rv.endStage(StageFetch, diffSize(len(diff)), nil)
	})
	log.Info("orchestrator: pr fetched", "head_sha", headSHA, "diff_bytes", len(diff), "duration_ms", time.Since(t).Milliseconds())

	sess, concerns, err := r.reviewer.Review(r.ctx, reviewer.ReviewRequest{
		Owner:    owner,
		Repo:     repo,
		PRNumber: prNumber,
		HeadSHA:  headSHA,
		Title:    pr.Title,
		Body:     pr.Body,
		Diff:     string(diff),
		Progress: r.progressFor(key),
	})

	r.update(key, func(rv *Review) {
		if err != nil {
			rv.finish(err)
			return
		}
		rv.SessionID = sess.ID
		rv.Summary = sess.Summary
		rv.Concerns = toConcerns(concerns)
		rv.finish(nil)
	})

	if err != nil {
		log.Error("orchestrator: review failed", "err", err)
		return
	}
	log.Info("orchestrator: review complete", "concerns", len(concerns), "duration_ms", time.Since(t).Milliseconds())
}

func (r *Runner) fetch(owner, repo string, prNumber int) (*github.PR, []byte, error) {
	pr, err := r.source.GetPR(owner, repo, prNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch pr: %w", err)
	}
	diff, err := r.source.GetPRDiff(owner, repo, prNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch diff: %w", err)
	}
	return pr, diff, nil
}

func (r *Runner) progressFor(key string) reviewer.Progress {
	return func(ev reviewer.ProgressEvent) {
		r.update(key, func(rv *Review) {
			if ev.SessionID != "" {
				rv.OpencodeSessionPath = r.path(ev.SessionID)
			}
			if ev.Done {
				rv.endStage(ev.Stage, ev.Detail, ev.Err)
				return
			}
			rv.startStage(ev.Stage)
		})
	}
}

func (r *Runner) update(key string, mutate func(*Review)) {
	r.mu.Lock()
	review, ok := r.inFlight[key]
	if !ok {
		r.mu.Unlock()
		return
	}
	mutate(review)
	snapshot := review.clone()
	r.mu.Unlock()

	r.publish(key, snapshot)
}

func (r *Runner) release(key string) {
	r.mu.Lock()
	delete(r.inFlight, key)
	r.mu.Unlock()
}
