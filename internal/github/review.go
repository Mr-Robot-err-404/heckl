package github

import (
	"fmt"
	"log/slog"
	"sync"
)

const reviewFetchConcurrency = 24

func (c *Client) Viewer() (string, error) {
	c.viewerMu.Lock()
	defer c.viewerMu.Unlock()

	if c.viewer != "" {
		return c.viewer, nil
	}

	var u User
	if err := c.decode("/user", &u); err != nil {
		return "", err
	}
	c.viewer = u.Login
	return c.viewer, nil
}

func (c *Client) ListPRReviews(owner, repo string, number int) ([]Review, error) {
	var reviews []Review
	err := c.decode(fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews?per_page=100", owner, repo, number), &reviews)
	return reviews, err
}

func (c *Client) ListPRReviewComments(owner, repo string, number int) ([]ReviewComment, error) {
	var comments []ReviewComment
	err := c.decode(fmt.Sprintf("/repos/%s/%s/pulls/%d/comments?per_page=100", owner, repo, number), &comments)
	return comments, err
}

func (c *Client) ReviewsForPRs(owner, repo string, numbers []int) map[int][]Review {
	out := make(map[int][]Review, len(numbers))

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, reviewFetchConcurrency)

	for _, number := range numbers {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			reviews, err := c.ListPRReviews(owner, repo, n)
			if err != nil {
				slog.Warn("github: list pr reviews failed", "owner", owner, "repo", repo, "number", n, "err", err)
				return
			}
			mu.Lock()
			out[n] = reviews
			mu.Unlock()
		}(number)
	}

	wg.Wait()
	return out
}

type ReviewVerdict struct {
	Approvals         []User
	ChangesRequested  []User
	ViewerApproved    bool
	ViewerHasReviewed bool
}

func Verdict(reviews []Review, viewer string) ReviewVerdict {
	latest := make(map[string]Review, len(reviews))
	for _, review := range reviews {
		switch review.State {
		case "APPROVED", "CHANGES_REQUESTED", "DISMISSED":
		default:
			continue
		}
		if prev, ok := latest[review.User.Login]; ok && prev.SubmittedAt > review.SubmittedAt {
			continue
		}
		latest[review.User.Login] = review
	}

	verdict := ReviewVerdict{Approvals: []User{}, ChangesRequested: []User{}}
	for login, review := range latest {
		if login == viewer {
			verdict.ViewerHasReviewed = true
		}
		switch review.State {
		case "APPROVED":
			verdict.Approvals = append(verdict.Approvals, review.User)
			if login == viewer {
				verdict.ViewerApproved = true
			}
		case "CHANGES_REQUESTED":
			verdict.ChangesRequested = append(verdict.ChangesRequested, review.User)
		}
	}
	return verdict
}
