package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/Mr-Robot-err-404/heckl/internal/github"
	"github.com/Mr-Robot-err-404/heckl/internal/store"
)

const recentPRLimit = 12

type recentPRFilter struct {
	Organizations []string `json:"organizations"`
	Repositories  []string `json:"repositories"`
}

type recentPRsResponse struct {
	PullRequests []prResponse `json:"pullRequests"`
	Unavailable  []string     `json:"unavailable,omitempty"`
}

func normalizeRecentPRFilter(filter recentPRFilter) (recentPRFilter, error) {
	normalize := func(values []string, repository bool) ([]string, error) {
		kind := "organization"
		if repository {
			kind = "repository"
		}
		seen := make(map[string]bool, len(values))
		out := make([]string, 0, len(values))
		for _, raw := range values {
			value := strings.TrimSpace(raw)
			parts := strings.Split(value, "/")
			valid := value != "" && ((!repository && len(parts) == 1) || (repository && len(parts) == 2 && parts[0] != "" && parts[1] != ""))
			if !valid {
				return nil, fmt.Errorf("invalid %s: %q", kind, raw)
			}
			if !seen[value] {
				seen[value] = true
				out = append(out, value)
			}
		}
		sort.Strings(out)
		return out, nil
	}

	organizations, err := normalize(filter.Organizations, false)
	if err != nil {
		return recentPRFilter{}, err
	}
	repositories, err := normalize(filter.Repositories, true)
	if err != nil {
		return recentPRFilter{}, err
	}
	return recentPRFilter{Organizations: organizations, Repositories: repositories}, nil
}

func (s *Server) recentPRFilter(ctxReq *http.Request) (recentPRFilter, error) {
	raw, err := s.store.GetRecentPRFilter(ctxReq.Context())
	if err != nil || raw == "" {
		return recentPRFilter{}, err
	}
	var filter recentPRFilter
	if err := json.Unmarshal([]byte(raw), &filter); err != nil {
		return recentPRFilter{}, fmt.Errorf("decode recent PR filter: %w", err)
	}
	return normalizeRecentPRFilter(filter)
}

func (s *Server) handleGetRecentPRFilter(w http.ResponseWriter, r *http.Request) {
	filter, err := s.recentPRFilter(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if filter.Organizations == nil {
		filter.Organizations = []string{}
	}
	if filter.Repositories == nil {
		filter.Repositories = []string{}
	}
	jsonOK(w, filter)
}

func (s *Server) handleSetRecentPRFilter(w http.ResponseWriter, r *http.Request) {
	var body recentPRFilter
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	filter, err := normalizeRecentPRFilter(body)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	raw, err := json.Marshal(filter)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.store.SetRecentPRFilter(r.Context(), string(raw)); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, filter)
}

func (s *Server) handleListRecentPRs(w http.ResponseWriter, r *http.Request) {
	filter, err := s.recentPRFilter(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	repos, err := s.store.ListRepos(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	excludedOrgs := make(map[string]bool, len(filter.Organizations))
	for _, owner := range filter.Organizations {
		excludedOrgs[owner] = true
	}
	excludedRepos := make(map[string]bool, len(filter.Repositories))
	for _, repo := range filter.Repositories {
		excludedRepos[repo] = true
	}

	type result struct {
		repo *store.Repo
		prs  []github.PR
		err  error
	}
	results := make([]result, 0, len(repos))
	for _, repo := range repos {
		if excludedOrgs[repo.Owner] || excludedRepos[repo.Owner+"/"+repo.Name] {
			continue
		}
		results = append(results, result{repo: repo})
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for i := range results {
		wg.Add(1)
		go func(result *result) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			result.prs, result.err = s.gh.ListRepoPRs(r.Context(), result.repo.Owner, result.repo.Name)
		}(&results[i])
	}
	wg.Wait()

	response := recentPRsResponse{PullRequests: []prResponse{}}
	for _, result := range results {
		if result.err != nil {
			name := result.repo.Owner + "/" + result.repo.Name
			slog.Warn("github: recent PR list failed", "repo", name, "err", result.err)
			response.Unavailable = append(response.Unavailable, name)
			continue
		}
		for _, pr := range result.prs {
			response.PullRequests = append(response.PullRequests, toPRResponse(result.repo.Owner, result.repo.Name, pr))
		}
	}
	sort.Slice(response.PullRequests, func(i, j int) bool {
		return response.PullRequests[i].UpdatedAt > response.PullRequests[j].UpdatedAt
	})
	if len(response.PullRequests) > recentPRLimit {
		response.PullRequests = response.PullRequests[:recentPRLimit]
	}
	s.enrichPRs(r.Context(), response.PullRequests)
	sort.Strings(response.Unavailable)
	jsonOK(w, response)
}
