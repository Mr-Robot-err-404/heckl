package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/harrylawton/pr-review/internal/checkout"
	"github.com/harrylawton/pr-review/internal/github"
	"github.com/harrylawton/pr-review/internal/opencode"
	"github.com/harrylawton/pr-review/internal/orchestrator"
	"github.com/harrylawton/pr-review/internal/reviewer"
	"github.com/harrylawton/pr-review/internal/server"
	"github.com/harrylawton/pr-review/internal/store"
)

//go:embed all:dist
var static embed.FS

func main() {
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		log.Fatal("gh auth token failed — run `gh auth login` first")
	}
	token := strings.TrimSpace(string(out))

	db, err := store.Open("pr-review.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	projectDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	oc, err := opencode.Setup("", projectDir)
	if err != nil {
		log.Fatal("opencode setup failed:", err)
	}

	gh := github.New(token)
	co := checkout.New(filepath.Join(projectDir, "data"))
	rev := reviewer.New(oc, co, db)
	sessionPath := func(id string) string { return opencode.SessionPath(projectDir, id) }
	orc := orchestrator.New(context.Background(), slog.Default(), gh, rev, db, sessionPath)

	srv := server.New(gh, db, orc)

	dist, err := fs.Sub(static, "dist")
	if err != nil {
		log.Fatal(err)
	}
	srv.Static(http.FS(dist))

	addr := ":7331"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatal(err)
	}
}
