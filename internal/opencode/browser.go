package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) BrowserCookies(ctx context.Context, host string) ([]*http.Cookie, error) {
	var pairing struct {
		Code string `json:"code"`
	}
	pairRequest, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/pair", nil)
	if err != nil {
		return nil, err
	}
	c.authorize(pairRequest)
	pairResponse, err := c.http.Do(pairRequest)
	if err != nil {
		return nil, err
	}
	defer pairResponse.Body.Close()
	if pairResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opencode: pairing returned %d", pairResponse.StatusCode)
	}
	if err := json.NewDecoder(pairResponse.Body).Decode(&pairing); err != nil {
		return nil, err
	}
	if pairing.Code == "" {
		return nil, fmt.Errorf("opencode: empty pairing code")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/auth/connect/"+url.PathEscape(pairing.Code), nil)
	if err != nil {
		return nil, err
	}
	req.Host = host
	req.Header.Set("Accept", "text/html")
	client := *c.http
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 300 || resp.StatusCode >= 400 || len(resp.Cookies()) == 0 {
		return nil, fmt.Errorf("opencode: browser authentication returned %d without a session cookie", resp.StatusCode)
	}
	return resp.Cookies(), nil
}
