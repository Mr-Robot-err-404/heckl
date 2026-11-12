package opencode

import (
	"encoding/json"
	"net/url"
	"strings"
)

type ModelRef struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"id"`
	Variant    string `json:"variant,omitempty"`
}

type ToolState struct {
	Status string          `json:"status"`
	Input  json.RawMessage `json:"input,omitempty"`
}

type Part struct {
	Type  string     `json:"type"`
	Text  string     `json:"text,omitempty"`
	Tool  string     `json:"name,omitempty"`
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

type Message struct {
	ID        string          `json:"id"`
	Role      string          `json:"role"`
	SessionID string          `json:"sessionID"`
	Error     json.RawMessage `json:"error,omitempty"`
}

type MessageResponse struct {
	Info  Message `json:"info"`
	Parts []Part  `json:"parts"`
}

func (m Message) HasError() bool {
	return len(m.Error) > 0 && string(m.Error) != "null"
}

const abortedError = "MessageAbortedError"

func (m Message) Aborted() bool {
	if len(m.Error) == 0 {
		return false
	}
	var e struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(m.Error, &e); err != nil {
		return false
	}
	return e.Name == abortedError
}

type PromptRequest struct {
	Text string `json:"text"`
}

func ParseModel(ref string) *ModelRef {
	provider, model, ok := strings.Cut(strings.TrimSpace(ref), "/")
	if !ok || provider == "" || model == "" {
		return nil
	}
	model, variant, _ := strings.Cut(model, "#")
	return &ModelRef{ProviderID: provider, ModelID: model, Variant: variant}
}

func (c *Client) Prompt(sessionID string, req PromptRequest) (*MessageResponse, error) {
	if err := c.decode("POST", "/api/session/"+sessionID+"/prompt", req, nil); err != nil {
		return nil, err
	}
	if err := c.decode("POST", "/api/experimental/session/"+sessionID+"/wait", nil, nil); err != nil {
		return nil, err
	}
	msgs, err := c.Messages(sessionID)
	if err != nil {
		return nil, err
	}
	session, err := c.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	var result MessageResponse
	for _, msg := range msgs {
		if msg.Info.Role == "assistant" || msg.Info.HasError() {
			result = msg
		}
	}
	if session.Outcome == "interrupted" {
		result.Info.Error = json.RawMessage(`{"name":"MessageAbortedError"}`)
	}
	if session.Outcome == "failed" && !result.Info.HasError() {
		result.Info.Error = json.RawMessage(`{"type":"SessionFailed","message":"Session failed without an error message"}`)
	}
	return &result, nil
}

func (c *Client) Messages(sessionID string) ([]MessageResponse, error) {
	var msgs []MessageResponse
	path := "/api/session/" + sessionID + "/message?order=asc"
	for {
		var page struct {
			Data []struct {
				ID      string          `json:"id"`
				Type    string          `json:"type"`
				Content []Part          `json:"content"`
				Error   json.RawMessage `json:"error"`
			} `json:"data"`
			Cursor struct {
				Next string `json:"next"`
			} `json:"cursor"`
		}
		if err := c.decode("GET", path, nil, &page); err != nil {
			return nil, err
		}
		for _, msg := range page.Data {
			msgs = append(msgs, MessageResponse{
				Info:  Message{ID: msg.ID, Role: msg.Type, SessionID: sessionID, Error: msg.Error},
				Parts: msg.Content,
			})
		}
		if page.Cursor.Next == "" {
			return msgs, nil
		}
		path = "/api/session/" + sessionID + "/message?cursor=" + url.QueryEscape(page.Cursor.Next)
	}
}
