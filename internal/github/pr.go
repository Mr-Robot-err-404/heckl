package github

import (
	"fmt"
	"io"
)

type User struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

type Review struct {
	ID          int64  `json:"id"`
	User        User   `json:"user"`
	State       string `json:"state"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	SubmittedAt string `json:"submitted_at"`
}

type ReviewComment struct {
	ID           int64  `json:"id"`
	User         User   `json:"user"`
	Body         string `json:"body"`
	Path         string `json:"path"`
	Line         *int   `json:"line"`
	OriginalLine *int   `json:"original_line"`
	Side         string `json:"side"`
	InReplyToID  int64  `json:"in_reply_to_id"`
	HTMLURL      string `json:"html_url"`
	CreatedAt    string `json:"created_at"`
}

type PRHead struct {
	SHA string `json:"sha"`
}

type PR struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	State     string `json:"state"`
	Body      string `json:"body"`
	HTMLURL   string `json:"html_url"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	User      User   `json:"user"`
	Draft     bool   `json:"draft"`
	Head      PRHead `json:"head"`

	RequestedReviewers []User `json:"requested_reviewers"`
}

func (pr *PR) HeadSHA() string { return pr.Head.SHA }

type PRFile struct {
	SHA       string `json:"sha"`
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Changes   int    `json:"changes"`
	Patch     string `json:"patch"`
}

func (c *Client) ListRepoPRs(owner, repo string) ([]PR, error) {
	var prs []PR
	err := c.decode(fmt.Sprintf("/repos/%s/%s/pulls?state=open&per_page=100", owner, repo), &prs)
	return prs, err
}

func (c *Client) GetPR(owner, repo string, number int) (*PR, error) {
	var pr PR
	err := c.decode(fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number), &pr)
	return &pr, err
}

func (c *Client) GetPRFiles(owner, repo string, number int) ([]PRFile, error) {
	var files []PRFile
	err := c.decode(fmt.Sprintf("/repos/%s/%s/pulls/%d/files?per_page=100", owner, repo, number), &files)
	return files, err
}

func (c *Client) GetPRDiff(owner, repo string, number int) ([]byte, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number)
	resp, err := c.doAccept("GET", path, "application/vnd.github.diff")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("github: get pr diff %s/%s#%d returned %d", owner, repo, number, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
