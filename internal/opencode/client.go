package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultBaseURL = "http://127.0.0.1:4420"

type Client struct {
	baseURL   string
	directory string
	password  string
	http      *http.Client
}

func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		password: environmentPassword(),
		http:     &http.Client{Timeout: 5 * time.Minute},
	}
}

func (c *Client) do(method, path string, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.authorize(req)
	return c.http.Do(req)
}

func environmentPassword() string {
	if password := os.Getenv("OPENCODE_PASSWORD"); password != "" {
		return password
	}
	return os.Getenv("OPENCODE_SERVER_PASSWORD")
}

func (c *Client) authorize(req *http.Request) {
	if c.password != "" {
		req.SetBasicAuth("opencode", c.password)
	}
}

func (c *Client) LoadPassword(path string) error {
	if c.password != "" || path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	c.password = strings.TrimSpace(string(data))
	if c.password == "" {
		return fmt.Errorf("opencode: password file %s is empty", path)
	}
	return nil
}

func (c *Client) decode(method, path string, body, out any) error {
	resp, err := c.do(method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("opencode: %s %s returned %d: %s", method, path, resp.StatusCode, string(msg))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

const healthTimeout = 3 * time.Second

func (c *Client) Health() (healthy bool, version string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), healthTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/info", nil)
	if err != nil {
		return false, "", err
	}
	c.authorize(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return false, "", fmt.Errorf("opencode: health returned %d", resp.StatusCode)
	}

	var res struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return false, "", err
	}
	return strings.HasPrefix(strings.TrimPrefix(res.Version, "v"), "2."), res.Version, nil
}
