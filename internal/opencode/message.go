package opencode

import "encoding/json"

type ModelRef struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

type Part struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type OutputFormat struct {
	Type       string `json:"type"`
	Schema     any    `json:"schema"`
	RetryCount int    `json:"retryCount,omitempty"`
}

type Message struct {
	ID         string          `json:"id"`
	Role       string          `json:"role"`
	SessionID  string          `json:"sessionID"`
	Structured json.RawMessage `json:"structured,omitempty"`
	Error      json.RawMessage `json:"error,omitempty"`
}

type MessageResponse struct {
	Info  Message `json:"info"`
	Parts []Part  `json:"parts"`
}

func (m Message) HasError() bool {
	return len(m.Error) > 0
}

type PromptRequest struct {
	Agent  string        `json:"agent,omitempty"`
	Model  *ModelRef     `json:"model,omitempty"`
	Parts  []Part        `json:"parts"`
	Format *OutputFormat `json:"format,omitempty"`
}

func TextPrompt(text string) PromptRequest {
	return PromptRequest{Parts: []Part{{Type: "text", Text: text}}}
}

func (c *Client) Prompt(sessionID string, req PromptRequest) (*MessageResponse, error) {
	var res MessageResponse
	err := c.decode("POST", "/session/"+sessionID+"/message", req, &res)
	return &res, err
}

func (c *Client) Messages(sessionID string) ([]MessageResponse, error) {
	var msgs []MessageResponse
	err := c.decode("GET", "/session/"+sessionID+"/message", nil, &msgs)
	return msgs, err
}
