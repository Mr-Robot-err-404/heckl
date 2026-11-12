package opencode

type PermissionRule struct {
	Action   string `json:"action"`
	Resource string `json:"resource"`
	Effect   string `json:"effect"`
}

type Location struct {
	Directory string `json:"directory"`
}

type Session struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Outcome string `json:"outcome"`
}

func ReportOnlyPermissions() []PermissionRule {
	return []PermissionRule{
		{Action: "*", Resource: "*", Effect: "deny"},
		{Action: "report", Resource: "*", Effect: "allow"},
	}
}

func ReadOnlyPermission(path string) []PermissionRule {
	return append(ReportOnlyPermissions(),
		PermissionRule{Action: "execute", Resource: "*", Effect: "allow"},
		PermissionRule{Action: "read", Resource: "*", Effect: "allow"},
		PermissionRule{Action: "external_directory", Resource: path + "/*", Effect: "allow"},
	)
}

type CreateSessionRequest struct {
	Title      string           `json:"title,omitempty"`
	Agent      string           `json:"agent,omitempty"`
	Model      *ModelRef        `json:"model,omitempty"`
	Location   *Location        `json:"location,omitempty"`
	Metadata   map[string]any   `json:"metadata,omitempty"`
	Permission []PermissionRule `json:"permissions,omitempty"`
}

func (c *Client) CreateSession(req CreateSessionRequest) (*Session, error) {
	if req.Location == nil && c.directory != "" {
		req.Location = &Location{Directory: c.directory}
	}
	var res struct {
		Data Session `json:"data"`
	}
	err := c.decode("POST", "/api/session", req, &res)
	return &res.Data, err
}

func (c *Client) GetSession(id string) (*Session, error) {
	var res struct {
		Data Session `json:"data"`
	}
	err := c.decode("GET", "/api/session/"+id, nil, &res)
	return &res.Data, err
}

func (c *Client) DeleteSession(id string) error {
	return c.decode("DELETE", "/api/session/"+id, nil, nil)
}

func (c *Client) Abort(id string) error {
	return c.decode("POST", "/api/session/"+id+"/interrupt?resume=false", nil, nil)
}
