package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const baseURL = "https://api.github.com"

type Client struct {
	token  string
	http   *http.Client
}

func New(token string) *Client {
	return &Client{
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Token() string { return c.token }

func (c *Client) doAccept(method, path, accept string) (*http.Response, error) {
	req, err := http.NewRequest(method, baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	return c.http.Do(req)
}

func (c *Client) do(method, path string) (*http.Response, error) {
	return c.doAccept(method, path, "application/vnd.github+json")
}

func (c *Client) decode(path string, out any) error {
	resp, err := c.do("GET", path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("github: %s returned %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
