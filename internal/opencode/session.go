package opencode

type Session struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (c *Client) CreateSession(title string) (*Session, error) {
	body := map[string]string{}
	if title != "" {
		body["title"] = title
	}
	var s Session
	err := c.decode("POST", "/session", body, &s)
	return &s, err
}

func (c *Client) GetSession(id string) (*Session, error) {
	var s Session
	err := c.decode("GET", "/session/"+id, nil, &s)
	return &s, err
}

func (c *Client) DeleteSession(id string) error {
	return c.decode("DELETE", "/session/"+id, nil, nil)
}
