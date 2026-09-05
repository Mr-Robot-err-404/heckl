package opencode

import (
	"encoding/json"
	"strings"
)

type ModelRef struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

type ToolState struct {
	Status string          `json:"status"`
	Input  json.RawMessage `json:"input,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type Part struct {
	Type  string     `json:"type"`
	Text  string     `json:"text,omitempty"`
	Tool  string     `json:"tool,omitempty"`
	State *ToolState `json:"state,omitempty"`
}

func ToolInput(msgs []MessageResponse, name string) (json.RawMessage, bool) {
	for i := len(msgs) - 1; i >= 0; i-- {
		parts := msgs[i].Parts
		for j := len(parts) - 1; j >= 0; j-- {
			p := parts[j]
			if p.Type != "tool" || p.Tool != name || p.State == nil {
				continue
			}
			if p.State.Status != "completed" || len(p.State.Input) == 0 {
				continue
			}
			return p.State.Input, true
		}
	}
	return nil, false
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
	System string        `json:"system,omitempty"`
	Parts  []Part        `json:"parts"`
	Format *OutputFormat `json:"format,omitempty"`
}

func ParseModel(ref string) *ModelRef {
	provider, model, ok := strings.Cut(strings.TrimSpace(ref), "/")
	if !ok || provider == "" || model == "" {
		return nil
	}
	return &ModelRef{ProviderID: provider, ModelID: model}
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
