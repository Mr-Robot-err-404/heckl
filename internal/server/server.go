package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/harrylawton/pr-review/internal/checkout"
	"github.com/harrylawton/pr-review/internal/github"
	"github.com/harrylawton/pr-review/internal/opencode"
	"github.com/harrylawton/pr-review/internal/orchestrator"
	"github.com/harrylawton/pr-review/internal/store"
	"github.com/harrylawton/pr-review/internal/tmux"
)

var allowedAssetProxyHosts = map[string]bool{
	"github.com":                                true,
	"user-images.githubusercontent.com":         true,
	"private-user-images.githubusercontent.com": true,
}

type Server struct {
	gh           *github.Client
	store        *store.Store
	oc           *opencode.Client
	orchestrator *orchestrator.Orchestrator
	checkout     *checkout.Manager
	tmux         *tmux.Tmux
	remoteHost   string
	mux          *http.ServeMux
}

func New(gh *github.Client, store *store.Store, oc *opencode.Client, orc *orchestrator.Orchestrator, co *checkout.Manager, remoteHost string) *Server {
	s := &Server{gh: gh, store: store, oc: oc, orchestrator: orc, checkout: co, tmux: tmux.Init(), remoteHost: remoteHost, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Static(files http.FileSystem) {
	assets := http.FileServer(files)

	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		name := path.Clean("/" + r.URL.Path)
		if f, err := files.Open(name); err == nil {
			f.Close()
			assets.ServeHTTP(w, r)
			return
		}
		if path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}

		index, err := files.Open("/index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer index.Close()

		info, err := index.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeContent(w, r, "index.html", info.ModTime(), index)
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	s.mux.ServeHTTP(sw, r)

	level := slog.LevelInfo
	switch {
	case sw.status >= 400:
		level = slog.LevelWarn
	case !strings.HasPrefix(r.URL.Path, "/api/"):
		level = slog.LevelDebug
	}

	slog.Log(r.Context(), level, "request",
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

func (sw *statusWriter) Unwrap() http.ResponseWriter { return sw.ResponseWriter }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/orgs", s.handleListOrgs)
	s.mux.HandleFunc("GET /api/repos", s.handleListRepos)
	s.mux.HandleFunc("GET /api/repos/{owner}", s.handleListReposByOwner)
	s.mux.HandleFunc("POST /api/repos", s.handleAddRepo)
	s.mux.HandleFunc("DELETE /api/repos/{owner}/{name}", s.handleDeleteRepo)
	s.mux.HandleFunc("GET /api/prs/{owner}/{repo}", s.handleListPRs)
	s.mux.HandleFunc("GET /api/prs/{owner}/{repo}/{number}", s.handleGetPR)
	s.mux.HandleFunc("GET /api/prs/{owner}/{repo}/{number}/comments", s.handlePRComments)
	s.mux.HandleFunc("GET /api/diff/{owner}/{repo}/{number}", s.handleDiff)
	s.mux.HandleFunc("GET /api/asset", s.handleAsset)
	s.mux.HandleFunc("GET /api/reviews/history", s.handleReviewHistory)
	s.mux.HandleFunc("GET /api/reviews/stream", s.handleReviewsStream)
	s.mux.HandleFunc("GET /api/theme", s.handleGetTheme)
	s.mux.HandleFunc("PUT /api/theme", s.handleSetTheme)
	s.mux.HandleFunc("GET /api/agents", s.handleListAgents)
	s.mux.HandleFunc("GET /api/agents/config", s.handleAgentConfig)
	s.mux.HandleFunc("PUT /api/agents/config", s.handleSaveAgentConfig)
	s.mux.HandleFunc("POST /api/review/{owner}/{repo}/{number}", s.handleReview)
	s.mux.HandleFunc("POST /api/review/{owner}/{repo}/{number}/agent/{agent}", s.handleRerunAgent)
	s.mux.HandleFunc("GET /api/review/{owner}/{repo}/{number}/stream", s.handleReviewStream)
	s.mux.HandleFunc("POST /api/tmux/{owner}/{repo}/{number}", s.handleTmuxSession)
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

	RequestedReviewers []userResponse           `json:"requestedReviewers,omitempty"`
	Approvals          []userResponse           `json:"approvals,omitempty"`
	ChangesRequested   []userResponse           `json:"changesRequested,omitempty"`
	ViewerApproved     bool                     `json:"viewerApproved"`
	ViewerHasReviewed  bool                     `json:"viewerHasReviewed"`
	Review             *store.RepoReviewSummary `json:"review,omitempty"`
}

type userResponse struct {
	Login  string `json:"login"`
	Avatar string `json:"avatar"`
}

type reviewerThreadResponse struct {
	User  userResponse          `json:"user"`
	Notes []github.ReviewerNote `json:"notes"`
}

func toUsers(in []github.User) []userResponse {
	out := make([]userResponse, 0, len(in))
	for _, u := range in {
		out = append(out, userResponse{Login: u.Login, Avatar: u.AvatarURL})
	}
	return out
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
	slog.Debug("github: list prs", "owner", owner, "repo", repo, "count", len(remote), "duration_ms", time.Since(start).Milliseconds())

	numbers := make([]int, 0, len(remote))
	for _, pr := range remote {
		numbers = append(numbers, pr.Number)
	}

	var (
		wg        sync.WaitGroup
		viewer    string
		reviews   map[int][]github.Review
		summaries map[int]*store.RepoReviewSummary
	)

	wg.Go(func() {
		login, err := s.gh.Viewer()
		if err != nil {
			slog.Warn("github: viewer lookup failed", "err", err)
			return
		}
		viewer = login
	})
	wg.Go(func() {
		reviews = s.gh.ReviewsForPRs(owner, repo, numbers)
	})
	wg.Go(func() {
		stored, err := s.store.RepoReviewSummary(r.Context(), owner, repo)
		if err != nil {
			slog.Error("store: repo review summary failed", "owner", owner, "repo", repo, "err", err)
			stored = map[int]*store.RepoReviewSummary{}
		}
		summaries = stored
	})
	wg.Wait()

	out := make([]prResponse, 0, len(remote))
	for _, pr := range remote {
		item := toPRResponse(owner, repo, pr)
		item.RequestedReviewers = toUsers(pr.RequestedReviewers)

		verdict := github.Verdict(reviews[pr.Number], viewer)
		item.Approvals = toUsers(verdict.Approvals)
		item.ChangesRequested = toUsers(verdict.ChangesRequested)
		item.ViewerApproved = verdict.ViewerApproved
		item.ViewerHasReviewed = verdict.ViewerHasReviewed
		item.Review = summaries[pr.Number]

		out = append(out, item)
	}

	jsonOK(w, out)
}

func (s *Server) handlePRComments(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		jsonError(w, "invalid pr number", http.StatusBadRequest)
		return
	}

	start := time.Now()

	var (
		wg          sync.WaitGroup
		reviews     []github.Review
		reviewsErr  error
		comments    []github.ReviewComment
		commentsErr error
		viewer      string
	)
	wg.Go(func() {
		reviews, reviewsErr = s.gh.ListPRReviews(owner, repo, number)
	})
	wg.Go(func() {
		comments, commentsErr = s.gh.ListPRReviewComments(owner, repo, number)
	})
	wg.Go(func() {
		if login, err := s.gh.Viewer(); err == nil {
			viewer = login
		}
	})
	wg.Wait()

	if commentsErr != nil {
		slog.Error("github: list pr comments failed", "owner", owner, "repo", repo, "number", number, "err", commentsErr)
		jsonError(w, commentsErr.Error(), http.StatusBadGateway)
		return
	}
	if reviewsErr != nil {
		slog.Warn("github: list pr reviews failed", "owner", owner, "repo", repo, "number", number, "err", reviewsErr)
		reviews = nil
	}

	threads := github.GroupReviewerNotes(reviews, comments, viewer)
	out := make([]reviewerThreadResponse, 0, len(threads))
	for _, t := range threads {
		out = append(out, reviewerThreadResponse{
			User:  userResponse{Login: t.User.Login, Avatar: t.User.AvatarURL},
			Notes: t.Notes,
		})
	}
	slog.Debug("github: pr comments", "owner", owner, "repo", repo, "number", number, "reviewers", len(out), "duration_ms", time.Since(start).Milliseconds())

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

	var (
		wg       sync.WaitGroup
		pr       *github.PR
		prErr    error
		files    []github.PRFile
		filesErr error
	)
	wg.Go(func() {
		pr, prErr = s.gh.GetPR(owner, repo, number)
	})
	wg.Go(func() {
		files, filesErr = s.gh.GetPRFiles(owner, repo, number)
	})
	wg.Wait()

	if prErr != nil {
		slog.Error("github: get pr failed", "owner", owner, "repo", repo, "number", number, "err", prErr)
		jsonError(w, prErr.Error(), http.StatusBadGateway)
		return
	}
	if filesErr != nil {
		slog.Error("github: get pr files failed", "owner", owner, "repo", repo, "number", number, "err", filesErr)
		jsonError(w, filesErr.Error(), http.StatusBadGateway)
		return
	}
	slog.Debug("github: get pr", "owner", owner, "repo", repo, "number", number, "files", len(files), "duration_ms", time.Since(start).Milliseconds())

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

	diff, err := s.gh.GetPRDiff(owner, repo, number)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(diff)
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
