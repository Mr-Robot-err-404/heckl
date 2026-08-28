package github

import "fmt"

type User struct {
	Login string `json:"login"`
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
}

type PRFile struct {
	SHA      string `json:"sha"`
	Filename string `json:"filename"`
	Status   string `json:"status"`
	Additions int   `json:"additions"`
	Deletions int   `json:"deletions"`
	Changes   int   `json:"changes"`
	Patch    string `json:"patch"`
}

func (c *Client) ListRepoPRs(owner, repo string) ([]PR, error) {
	var prs []PR
	err := c.decode(fmt.Sprintf("/repos/%s/%s/pulls?state=open&per_page=100", owner, repo), &prs)
	return prs, err
}

func (c *Client) ListAssignedPRs() ([]PR, error) {
	var prs []PR
	err := c.decode("/search/issues?q=is:pr+is:open+assignee:@me&per_page=100", &prs)
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
