package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const baseURL = "https://api.github.com"

type Client struct {
	token string
	http  *http.Client

	viewerMu sync.Mutex
	viewer   string
}

func New(token string) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 64
	transport.MaxIdleConnsPerHost = 32
	transport.IdleConnTimeout = 5 * time.Minute

	return &Client{
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second, Transport: transport},
	}
}

func (c *Client) Warm(ctx context.Context) {
	go c.Viewer(ctx)
}

func (c *Client) Token() string { return c.token }

func (c *Client) doAccept(ctx context.Context, method, path, accept string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	return c.http.Do(req)
}

func (c *Client) do(ctx context.Context, method, path string) (*http.Response, error) {
	return c.doAccept(ctx, method, path, "application/vnd.github+json")
}

func (c *Client) decode(ctx context.Context, path string, out any) error {
	resp, err := c.do(ctx, "GET", path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("github: %s returned %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
