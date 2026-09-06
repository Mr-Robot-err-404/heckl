package ghauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	deviceCodeURL  = "https://github.com/login/device/code"
	accessTokenURL = "https://github.com/login/oauth/access_token"
	scopes         = "repo read:org"
)

type DeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Interval         int    `json:"interval"`
}

func RequestDeviceCode(ctx context.Context, clientID string) (*DeviceCode, error) {
	form := url.Values{"client_id": {clientID}, "scope": {scopes}}
	var out DeviceCode
	if err := postForm(ctx, deviceCodeURL, form, &out); err != nil {
		return nil, err
	}
	if out.DeviceCode == "" {
		return nil, fmt.Errorf("github: device code request returned nothing - is %q a valid oauth client id with device flow enabled?", clientID)
	}
	if out.Interval <= 0 {
		out.Interval = 5
	}
	return &out, nil
}

func PollForToken(ctx context.Context, clientID string, code *DeviceCode) (string, error) {
	form := url.Values{
		"client_id":   {clientID},
		"device_code": {code.DeviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}

	interval := time.Duration(code.Interval) * time.Second
	deadline := time.Now().Add(time.Duration(code.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(interval):
		}

		var res tokenResponse
		if err := postForm(ctx, accessTokenURL, form, &res); err != nil {
			return "", err
		}

		switch res.Error {
		case "":
			if res.AccessToken == "" {
				return "", fmt.Errorf("github: token response had neither a token nor an error")
			}
			return res.AccessToken, nil
		case "authorization_pending":
		case "slow_down":
			interval += 5 * time.Second
		case "expired_token":
			return "", fmt.Errorf("github: the device code expired before you approved it")
		case "access_denied":
			return "", fmt.Errorf("github: authorisation was denied")
		default:
			return "", fmt.Errorf("github: %s: %s", res.Error, res.ErrorDescription)
		}
	}
	return "", fmt.Errorf("github: timed out waiting for you to approve the device code")
}

func postForm(ctx context.Context, endpoint string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("github: %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("github: %s returned %d: %s", endpoint, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.Unmarshal(body, out)
}
