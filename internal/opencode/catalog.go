package opencode

import (
	"net/url"
	"sort"
)

type Model struct {
	ID         string `json:"id"`
	ProviderID string `json:"providerID"`
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
}

type Provider struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ModelOption struct {
	Ref      string `json:"ref"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

func (c *Client) Models() ([]ModelOption, error) {
	var providers struct {
		Data []Provider `json:"data"`
	}
	if err := c.decode("GET", c.locationPath("/api/provider"), nil, &providers); err != nil {
		return nil, err
	}
	var models struct {
		Data []Model `json:"data"`
	}
	if err := c.decode("GET", c.locationPath("/api/model"), nil, &models); err != nil {
		return nil, err
	}
	names := make(map[string]string, len(providers.Data))
	for _, p := range providers.Data {
		names[p.ID] = p.Name
	}
	var out []ModelOption
	for _, m := range models.Data {
		if !m.Enabled {
			continue
		}
		name := m.Name
		if name == "" {
			name = m.ID
		}
		provider := names[m.ProviderID]
		if provider == "" {
			provider = m.ProviderID
		}
		out = append(out, ModelOption{Ref: m.ProviderID + "/" + m.ID, Name: name, Provider: provider})
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
	var res struct {
		Data []struct {
			ID    string    `json:"id"`
			Model *ModelRef `json:"model,omitempty"`
		} `json:"data"`
	}
	if err := c.decode("GET", c.locationPath("/api/agent"), nil, &res); err != nil {
		return nil, err
	}
	out := make([]AgentInfo, 0, len(res.Data))
	for _, agent := range res.Data {
		out = append(out, AgentInfo{Name: agent.ID, Model: agent.Model})
	}
	return out, nil
}

func (c *Client) locationPath(path string) string {
	if c.directory == "" {
		return path
	}
	return path + "?" + url.Values{"location[directory]": {c.directory}}.Encode()
}

func (m *ModelRef) Ref() string {
	if m == nil {
		return ""
	}
	ref := m.ProviderID + "/" + m.ModelID
	if m.Variant != "" {
		ref += "#" + m.Variant
	}
	return ref
}
