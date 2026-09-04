package opencode

import "sort"

type Model struct {
	ID         string `json:"id"`
	ProviderID string `json:"providerID"`
	Name       string `json:"name"`
}

type Provider struct {
	ID     string           `json:"id"`
	Name   string           `json:"name"`
	Models map[string]Model `json:"models"`
}

type providersResponse struct {
	All       []Provider `json:"all"`
	Connected []string   `json:"connected"`
}

type ModelOption struct {
	Ref      string `json:"ref"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

func (c *Client) Models() ([]ModelOption, error) {
	var res providersResponse
	if err := c.decode("GET", "/provider", nil, &res); err != nil {
		return nil, err
	}

	connected := make(map[string]bool, len(res.Connected))
	for _, id := range res.Connected {
		connected[id] = true
	}

	var out []ModelOption
	for _, p := range res.All {
		if !connected[p.ID] {
			continue
		}
		for id, m := range p.Models {
			name := m.Name
			if name == "" {
				name = id
			}
			out = append(out, ModelOption{Ref: p.ID + "/" + id, Name: name, Provider: p.Name})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Provider != out[j].Provider {
			return out[i].Provider < out[j].Provider
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

type AgentInfo struct {
	Name  string    `json:"name"`
	Model *ModelRef `json:"model,omitempty"`
}

func (c *Client) Agents() ([]AgentInfo, error) {
	var out []AgentInfo
	err := c.decode("GET", "/agent", nil, &out)
	return out, err
}

func (m *ModelRef) Ref() string {
	if m == nil {
		return ""
	}
	return m.ProviderID + "/" + m.ModelID
}
