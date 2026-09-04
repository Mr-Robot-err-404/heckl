package github

import "sort"

type ReviewerNote struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	State     string `json:"state,omitempty"`
	File      string `json:"file,omitempty"`
	Line      *int   `json:"line,omitempty"`
	Side      string `json:"side,omitempty"`
	Outdated  bool   `json:"outdated,omitempty"`
	Reply     bool   `json:"reply,omitempty"`
	URL       string `json:"url,omitempty"`
	CreatedAt string `json:"createdAt"`
}

type ReviewerThread struct {
	User  User           `json:"user"`
	Notes []ReviewerNote `json:"notes"`
}

func GroupReviewerNotes(reviews []Review, comments []ReviewComment, viewer string) []ReviewerThread {
	byAuthor := map[string]*ReviewerThread{}

	add := func(user User, note ReviewerNote) {
		if user.Login == "" || user.Login == viewer {
			return
		}
		thread, ok := byAuthor[user.Login]
		if !ok {
			thread = &ReviewerThread{User: user, Notes: []ReviewerNote{}}
			byAuthor[user.Login] = thread
		}
		thread.Notes = append(thread.Notes, note)
	}

	for _, review := range reviews {
		if review.Body == "" {
			continue
		}
		add(review.User, ReviewerNote{
			ID:        review.ID,
			Body:      review.Body,
			State:     review.State,
			URL:       review.HTMLURL,
			CreatedAt: review.SubmittedAt,
		})
	}

	for _, c := range comments {
		add(c.User, ReviewerNote{
			ID:        c.ID,
			Body:      c.Body,
			File:      c.Path,
			Line:      c.Line,
			Side:      diffSide(c.Side),
			Outdated:  c.Line == nil,
			Reply:     c.InReplyToID != 0,
			URL:       c.HTMLURL,
			CreatedAt: c.CreatedAt,
		})
	}

	out := make([]ReviewerThread, 0, len(byAuthor))
	for _, thread := range byAuthor {
		sort.SliceStable(thread.Notes, func(i, j int) bool {
			return thread.Notes[i].CreatedAt < thread.Notes[j].CreatedAt
		})
		out = append(out, *thread)
	}
	sort.SliceStable(out, func(i, j int) bool { return latestAt(out[i]) > latestAt(out[j]) })
	return out
}

func latestAt(thread ReviewerThread) string {
	if len(thread.Notes) == 0 {
		return ""
	}
	return thread.Notes[len(thread.Notes)-1].CreatedAt
}

func diffSide(side string) string {
	switch side {
	case "RIGHT":
		return "additions"
	case "LEFT":
		return "deletions"
	}
	return ""
}
