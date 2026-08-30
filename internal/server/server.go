package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/harrylawton/pr-review/internal/github"
	"github.com/harrylawton/pr-review/internal/store"
)

var allowedAssetProxyHosts = map[string]bool{
	"github.com":                        true,
	"user-images.githubusercontent.com": true,
	"private-user-images.githubusercontent.com": true,
}

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

func (s *Server) Static(fs http.FileSystem) {
	s.mux.Handle("/", http.FileServer(fs))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	s.mux.ServeHTTP(sw, r)
	slog.Info("request",
		"method", r.Method,
		"path", r.URL.Path,
		"status", sw.status,
		"duration_ms", time.Since(start).Milliseconds(),
	)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/orgs", s.handleListOrgs)
	s.mux.HandleFunc("GET /api/repos", s.handleListRepos)
	s.mux.HandleFunc("GET /api/repos/{owner}", s.handleListReposByOwner)
	s.mux.HandleFunc("POST /api/repos", s.handleAddRepo)
	s.mux.HandleFunc("DELETE /api/repos/{owner}/{name}", s.handleDeleteRepo)
	s.mux.HandleFunc("GET /api/prs/{owner}/{repo}", s.handleListPRs)
	s.mux.HandleFunc("GET /api/prs/{owner}/{repo}/{number}", s.handleGetPR)
	s.mux.HandleFunc("GET /api/diff/{owner}/{repo}/{number}", s.handleDiff)
	s.mux.HandleFunc("GET /api/asset", s.handleAsset)
}

type prResponse struct {
	Owner     string `json:"Owner"`
	Repo      string `json:"Repo"`
	Number    int    `json:"Number"`
	Title     string `json:"Title"`
	Body      string `json:"Body"`
	State     string `json:"State"`
	Author    string `json:"Author"`
	HtmlUrl   string `json:"HtmlUrl"`
	Draft     bool   `json:"Draft"`
	CreatedAt string `json:"CreatedAt"`
	UpdatedAt string `json:"UpdatedAt"`
}

type prFileResponse struct {
	Sha       string `json:"Sha"`
	Filename  string `json:"Filename"`
	Status    string `json:"Status"`
	Additions int    `json:"Additions"`
	Deletions int    `json:"Deletions"`
	Changes   int    `json:"Changes"`
	Patch     string `json:"Patch"`
}

func toPRResponse(owner, repo string, pr github.PR) prResponse {
	return prResponse{
		Owner:     owner,
		Repo:      repo,
		Number:    pr.Number,
		Title:     pr.Title,
		Body:      pr.Body,
		State:     pr.State,
		Author:    pr.User.Login,
		HtmlUrl:   pr.HTMLURL,
		Draft:     pr.Draft,
		CreatedAt: pr.CreatedAt,
		UpdatedAt: pr.UpdatedAt,
	}
}

func (s *Server) handleListPRs(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")

	start := time.Now()
	remote, err := s.gh.ListRepoPRs(owner, repo)
	if err != nil {
		slog.Error("github: list prs failed", "owner", owner, "repo", repo, "err", err)
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}
	slog.Info("github: list prs", "owner", owner, "repo", repo, "count", len(remote), "duration_ms", time.Since(start).Milliseconds())

	out := make([]prResponse, 0, len(remote))
	for _, pr := range remote {
		out = append(out, toPRResponse(owner, repo, pr))
	}

	jsonOK(w, out)
}

func (s *Server) handleGetPR(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		jsonError(w, "invalid pr number", http.StatusBadRequest)
		return
	}

	start := time.Now()
	pr, err := s.gh.GetPR(owner, repo, number)
	if err != nil {
		slog.Error("github: get pr failed", "owner", owner, "repo", repo, "number", number, "err", err)
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	files, err := s.gh.GetPRFiles(owner, repo, number)
	if err != nil {
		slog.Error("github: get pr files failed", "owner", owner, "repo", repo, "number", number, "err", err)
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}
	slog.Info("github: get pr", "owner", owner, "repo", repo, "number", number, "files", len(files), "duration_ms", time.Since(start).Milliseconds())

	fileOut := make([]prFileResponse, 0, len(files))
	for _, f := range files {
		fileOut = append(fileOut, prFileResponse{
			Sha:       f.SHA,
			Filename:  f.Filename,
			Status:    f.Status,
			Additions: f.Additions,
			Deletions: f.Deletions,
			Changes:   f.Changes,
			Patch:     f.Patch,
		})
	}

	jsonOK(w, map[string]any{"pr": toPRResponse(owner, repo, *pr), "files": fileOut})
}

func (s *Server) handleListOrgs(w http.ResponseWriter, r *http.Request) {
	orgs, err := s.store.ListOrgs(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if orgs == nil {
		orgs = []string{}
	}
	jsonOK(w, orgs)
}

func (s *Server) handleListReposByOwner(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repos, err := s.store.ListReposByOwner(r.Context(), owner)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if repos == nil {
		repos = []*store.Repo{}
	}
	jsonOK(w, repos)
}

func (s *Server) handleListRepos(w http.ResponseWriter, r *http.Request) {
	repos, err := s.store.ListRepos(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if repos == nil {
		repos = []*store.Repo{}
	}
	jsonOK(w, repos)
}

func (s *Server) handleAddRepo(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Owner string `json:"owner"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Owner == "" || body.Name == "" {
		jsonError(w, "owner and name required", http.StatusBadRequest)
		return
	}
	repo, err := s.store.AddRepo(r.Context(), body.Owner, body.Name)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, repo)
}

func (s *Server) handleDeleteRepo(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	name := r.PathValue("name")
	if err := s.store.DeleteRepo(r.Context(), owner, name); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		http.Error(w, "invalid pr number", http.StatusBadRequest)
		return
	}

	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/pulls/%d",
		owner, repo, number,
	)
	req, err := http.NewRequestWithContext(r.Context(), "GET", url, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+s.gh.Token())
	req.Header.Set("Accept", "application/vnd.github.diff")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		http.Error(w, fmt.Sprintf("github: %d", resp.StatusCode), resp.StatusCode)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	io.Copy(w, resp.Body)
}

func (s *Server) handleAsset(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("url")
	if raw == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}

	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "https") || !allowedAssetProxyHosts[parsed.Host] {
		slog.Warn("asset proxy: rejected url", "url", raw)
		http.Error(w, "url not allowed", http.StatusForbidden)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), "GET", raw, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Authorization", "Bearer "+s.gh.Token())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("asset proxy: fetch failed", "url", raw, "err", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Error("asset proxy: upstream error", "url", raw, "status", resp.StatusCode)
		http.Error(w, fmt.Sprintf("github: %d", resp.StatusCode), resp.StatusCode)
		return
	}

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	io.Copy(w, resp.Body)
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
