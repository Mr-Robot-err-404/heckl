package server

import (
	"log/slog"
	"net/http"
	"unicode/utf8"

	"github.com/Mr-Robot-err-404/heckl/internal/github"
)

const maxBlobBytes = 2 << 20

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

	pr, err := s.gh.GetPR(owner, repo, number)
	if err != nil {
		slog.Error("blob: get pr failed", "owner", owner, "repo", repo, "number", number, "err", err)
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	res := blobResponse{
		OldFile: s.fileAt(r, owner, repo, pr.Base, prevPath),
		NewFile: s.fileAt(r, owner, repo, pr.Head, path),
	}
	if res.OldFile == nil && res.NewFile == nil {
		jsonError(w, "no readable contents for "+path, http.StatusNotFound)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	jsonOK(w, res)
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
