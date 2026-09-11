package github

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
)

const markReadConcurrency = 8

type NotificationSubject struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Type  string `json:"type"`
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

func (c *Client) ListNotifications() ([]Notification, error) {
	var notifications []Notification
	err := c.decode("/notifications?participating=true&per_page=100", &notifications)
	return notifications, err
}

func (c *Client) MarkThreadRead(id string) error {
	resp, err := c.do("PATCH", "/notifications/threads/"+id)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("github: mark thread %s read returned %d", id, resp.StatusCode)
	}
	return nil
}

func (c *Client) MarkThreadsRead(ids []string) {
	sem := make(chan struct{}, markReadConcurrency)
	var wg sync.WaitGroup

	for _, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := c.MarkThreadRead(id); err != nil {
				slog.Warn("notifications: mark read failed", "thread", id, "err", err)
			}
		}()
	}
	wg.Wait()
}
