package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/harrylawton/pr-review/internal/github"
	"github.com/harrylawton/pr-review/internal/server"
	"github.com/harrylawton/pr-review/internal/store"
)

//go:embed all:dist
var static embed.FS

func readGithubSessionCookie() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(home, ".config", "pr-review", "session_cookie")
	b, err := os.ReadFile(path)
	if err != nil {
		log.Printf("no session cookie at %s — video embeds will not resolve", path)
		return ""
	}
	return strings.TrimSpace(string(b))
}

func main() {
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		log.Fatal("gh auth token failed — run `gh auth login` first")
	}
	token := strings.TrimSpace(string(out))
	sessionCookie := readGithubSessionCookie()

	db, err := store.Open("pr-review.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	gh := github.New(token)
	srv := server.New(gh, db, sessionCookie)

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
