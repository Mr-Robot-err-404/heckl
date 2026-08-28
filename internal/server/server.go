package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/harrylawton/pr-review/internal/github"
	"github.com/harrylawton/pr-review/internal/store"
)

type Server struct {
	gh    *github.Client
	store *store.Store
	mux   *http.ServeMux
}

func New(gh *github.Client, store *store.Store) *Server {
	s := &Server{gh: gh, store: store, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/prs/{owner}/{repo}", s.handleListPRs)
	s.mux.HandleFunc("GET /api/prs/{owner}/{repo}/{number}", s.handleGetPR)
}

func (s *Server) handleListPRs(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")

	prs, err := s.store.ListPRs(r.Context(), owner, repo)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(prs) == 0 {
		remote, err := s.gh.ListRepoPRs(owner, repo)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadGateway)
			return
		}
		for _, pr := range remote {
			files, err := s.gh.GetPRFiles(owner, repo, pr.Number)
			if err != nil {
				jsonError(w, err.Error(), http.StatusBadGateway)
				return
			}
			if _, err := s.store.SyncPR(r.Context(), owner, repo, &pr, files); err != nil {
				jsonError(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		prs, err = s.store.ListPRs(r.Context(), owner, repo)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	jsonOK(w, prs)
}

func (s *Server) handleGetPR(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		jsonError(w, "invalid pr number", http.StatusBadRequest)
		return
	}

	pr, err := s.store.GetPR(r.Context(), owner, repo, number)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			jsonError(w, "not found", http.StatusNotFound)
			return
		}
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	files, err := s.store.GetPRFiles(r.Context(), pr.ID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{"pr": pr, "files": files})
}


func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
