package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/Mr-Robot-err-404/heckl/internal/github"
)

const maxBlobBytes = 2 << 20

const maxPrefetchPaths = 50

const prefetchTimeout = 2 * time.Minute

type blobFile struct {
	Name     string `json:"name"`
	Contents string `json:"contents"`
}

type blobResponse struct {
	OldFile *blobFile `json:"oldFile"`
	NewFile *blobFile `json:"newFile"`
}

// handleBlob serves both sides of a file so the diff viewer can expand context
// beyond the hunks GitHub sent in the patch.
func (s *Server) handleBlob(w http.ResponseWriter, r *http.Request) {
	owner, repo, number, ok := prPath(w, r)
	if !ok {
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		jsonError(w, "missing path", http.StatusBadRequest)
		return
	}
	prevPath := r.URL.Query().Get("prev")
	if prevPath == "" {
		prevPath = path
	}

	base, head, ok := s.sides(w, r, owner, repo, number)
	if !ok {
		return
	}

	res := blobResponse{
		OldFile: s.fileAt(r, owner, repo, base, prevPath),
		NewFile: s.fileAt(r, owner, repo, head, path),
	}
	if res.OldFile == nil && res.NewFile == nil {
		jsonError(w, "no readable contents for "+path, http.StatusNotFound)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	jsonOK(w, res)
}

func (s *Server) sides(w http.ResponseWriter, r *http.Request, owner, repo string, number int) (base, head github.PRHead, ok bool) {
	q := r.URL.Query()
	base = github.PRHead{SHA: q.Get("baseSha"), Ref: q.Get("baseRef")}
	head = github.PRHead{SHA: q.Get("headSha"), Ref: q.Get("headRef")}
	if base.SHA != "" && head.SHA != "" {
		return base, head, true
	}

	pr, err := s.gh.GetPR(owner, repo, number)
	if err != nil {
		slog.Error("blob: get pr failed", "owner", owner, "repo", repo, "number", number, "err", err)
		jsonError(w, err.Error(), http.StatusBadGateway)
		return base, head, false
	}
	return pr.Base, pr.Head, true
}

type prefetchSide struct {
	SHA   string   `json:"sha"`
	Ref   string   `json:"ref"`
	Paths []string `json:"paths"`
}

type prefetchRequest struct {
	Base prefetchSide `json:"base"`
	Head prefetchSide `json:"head"`
}

// handlePrefetch warms both sides of a PR's changed files in the background.
func (s *Server) handlePrefetch(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")

	var req prefetchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid prefetch body", http.StatusBadRequest)
		return
	}

	for _, side := range []prefetchSide{req.Base, req.Head} {
		if side.SHA == "" || len(side.Paths) == 0 {
			continue
		}
		paths := side.Paths
		if len(paths) > maxPrefetchPaths {
			paths = paths[:maxPrefetchPaths]
		}
		go s.prefetch(owner, repo, side.SHA, side.Ref, paths)
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) prefetch(owner, repo, sha, ref string, paths []string) {
	ctx, cancel := context.WithTimeout(context.Background(), prefetchTimeout)
	defer cancel()

	start := time.Now()
	if err := s.checkout.Prefetch(ctx, owner, repo, sha, ref, paths); err != nil {
		slog.Warn("prefetch failed", "repo", repo, "sha", sha, "paths", len(paths), "err", err)
		return
	}
	slog.Debug("prefetch", "repo", repo, "sha", sha, "paths", len(paths), "duration_ms", time.Since(start).Milliseconds())
}

func (s *Server) fileAt(r *http.Request, owner, repo string, side github.PRHead, path string) *blobFile {
	raw, err := s.checkout.FileAt(r.Context(), owner, repo, side.SHA, side.Ref, path)
	if err != nil {
		slog.Debug("blob: read failed", "repo", repo, "sha", side.SHA, "path", path, "err", err)
		return nil
	}
	if len(raw) > maxBlobBytes || !utf8.Valid(raw) {
		return nil
	}
	return &blobFile{Name: path, Contents: string(raw)}
}
