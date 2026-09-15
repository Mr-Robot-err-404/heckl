package server

import (
	"context"
	"log/slog"
	"sync"

	"github.com/Mr-Robot-err-404/heckl/internal/github"
	"github.com/Mr-Robot-err-404/heckl/internal/store"
)

func (s *Server) enrichPRs(ctx context.Context, prs []prResponse) {
	if len(prs) == 0 {
		return
	}

	type repoKey struct{ owner, repo string }
	type enrichment struct {
		numbers   []int
		reviews   map[int][]github.Review
		summaries map[int]*store.RepoReviewSummary
	}

	groups := make(map[repoKey]*enrichment)
	for _, pr := range prs {
		key := repoKey{owner: pr.Owner, repo: pr.Repo}
		if groups[key] == nil {
			groups[key] = &enrichment{}
		}
		groups[key].numbers = append(groups[key].numbers, pr.Number)
	}

	var viewer string
	var wg sync.WaitGroup
	wg.Go(func() {
		login, err := s.gh.Viewer(ctx)
		if err != nil {
			slog.Warn("github: viewer lookup failed", "err", err)
			return
		}
		viewer = login
	})
	for key, group := range groups {
		wg.Go(func() {
			group.reviews = s.gh.ReviewsForPRs(ctx, key.owner, key.repo, group.numbers)
		})
		wg.Go(func() {
			stored, err := s.store.RepoReviewSummary(ctx, key.owner, key.repo)
			if err != nil {
				slog.Error("store: repo review summary failed", "owner", key.owner, "repo", key.repo, "err", err)
				stored = map[int]*store.RepoReviewSummary{}
			}
			group.summaries = stored
		})
	}
	wg.Wait()

	for i := range prs {
		pr := &prs[i]
		group := groups[repoKey{owner: pr.Owner, repo: pr.Repo}]
		verdict := github.Verdict(group.reviews[pr.Number], viewer)
		pr.Approvals = toUsers(verdict.Approvals)
		pr.ChangesRequested = toUsers(verdict.ChangesRequested)
		pr.ViewerApproved = verdict.ViewerApproved
		pr.ViewerHasReviewed = verdict.ViewerHasReviewed
		pr.Review = group.summaries[pr.Number]
	}
}
