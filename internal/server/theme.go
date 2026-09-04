package server

import (
	"encoding/json"
	"net/http"
)

const defaultTheme = "gruvbox"

var themes = []string{
	"gruvbox",
	"github-dark",
	"catppuccin-mocha",
	"kanagawa-wave",
	"tokyo-night",
	"everforest",
	"night-owl",
}

func knownTheme(name string) bool {
	for _, t := range themes {
		if t == name {
			return true
		}
	}
	return false
}

func (s *Server) handleGetTheme(w http.ResponseWriter, r *http.Request) {
	name, err := s.store.GetTheme(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !knownTheme(name) {
		name = defaultTheme
	}
	jsonOK(w, map[string]any{"theme": name, "available": themes})
}

func (s *Server) handleSetTheme(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Theme string `json:"theme"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if !knownTheme(body.Theme) {
		jsonError(w, "unknown theme: "+body.Theme, http.StatusBadRequest)
		return
	}
	if err := s.store.SetTheme(r.Context(), body.Theme); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]any{"theme": body.Theme, "available": themes})
}
