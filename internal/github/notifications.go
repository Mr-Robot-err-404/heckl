package github

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

const markReadConcurrency = 8

type NotificationSubject struct {
	Title            string `json:"title"`
	URL              string `json:"url"`
	LatestCommentURL string `json:"latest_comment_url"`
	Type             string `json:"type"`
}

type NotificationRepo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    User   `json:"owner"`
}

type Notification struct {
	ID         string              `json:"id"`
	Reason     string              `json:"reason"`
	Unread     bool                `json:"unread"`
	UpdatedAt  string              `json:"updated_at"`
	Subject    NotificationSubject `json:"subject"`
	Repository NotificationRepo    `json:"repository"`
}

func (n Notification) IsPullRequest() bool { return n.Subject.Type == "PullRequest" }

func (n Notification) PRNumber() int {
	i := strings.LastIndex(n.Subject.URL, "/")
	if i == -1 {
		return 0
	}
	number, err := strconv.Atoi(n.Subject.URL[i+1:])
	if err != nil {
		return 0
	}
	return number
}

func (c *Client) ListNotifications(ctx context.Context) ([]Notification, error) {
	var notifications []Notification
	err := c.decode(ctx, "/notifications?participating=true&per_page=100", &notifications)
	return notifications, err
}

type NotificationActivity struct {
	User      User   `json:"user"`
	Body      string `json:"body"`
	HTMLURL   string `json:"html_url"`
	CreatedAt string `json:"created_at"`
}

func (c *Client) GetNotificationActivity(ctx context.Context, rawURL string) (*NotificationActivity, error) {
	if rawURL == "" {
		return nil, nil
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host != "api.github.com" || !strings.HasPrefix(u.Path, "/repos/") {
		return nil, fmt.Errorf("github: invalid notification activity URL")
	}

	var activity NotificationActivity
	if err := c.decode(ctx, u.RequestURI(), &activity); err != nil {
		return nil, err
	}
	return &activity, nil
}

func (c *Client) MarkThreadRead(ctx context.Context, id string) error {
	resp, err := c.do(ctx, "PATCH", "/notifications/threads/"+id)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("github: mark thread %s read returned %d", id, resp.StatusCode)
	}
	return nil
}

func (c *Client) MarkThreadsRead(ctx context.Context, ids []string) {
	sem := make(chan struct{}, markReadConcurrency)
	var wg sync.WaitGroup

	for _, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := c.MarkThreadRead(ctx, id); err != nil {
				slog.Warn("notifications: mark read failed", "thread", id, "err", err)
			}
		}()
	}
	wg.Wait()
}
