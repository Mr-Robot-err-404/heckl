package opencode

const (
	PermissionAllow = "allow"
	PermissionDeny  = "deny"
	PermissionAsk   = "ask"
)

const (
	PermissionExternalDirectory = "external_directory"
	PermissionBash              = "bash"
	PermissionEdit              = "edit"
	PermissionRead              = "read"
	PermissionGlob              = "glob"
	PermissionGrep              = "grep"
)

type PermissionRule struct {
	Permission string `json:"permission"`
	Pattern    string `json:"pattern"`
	Action     string `json:"action"`
}

type Session struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func WorktreePermission(path string) []PermissionRule {
	return []PermissionRule{
		{Permission: PermissionExternalDirectory, Pattern: "*", Action: PermissionDeny},
		{Permission: PermissionExternalDirectory, Pattern: path + "/*", Action: PermissionAllow},
	}
}

type CreateSessionRequest struct {
	Title      string           `json:"title,omitempty"`
	Agent      string           `json:"agent,omitempty"`
	Permission []PermissionRule `json:"permission,omitempty"`
}

func (c *Client) CreateSession(req CreateSessionRequest) (*Session, error) {
	var s Session
	err := c.decode("POST", "/session", req, &s)
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
