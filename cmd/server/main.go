package main

import (
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/harrylawton/pr-review/internal/github"
	"github.com/harrylawton/pr-review/internal/server"
	"github.com/harrylawton/pr-review/internal/store"
)

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

	gh := github.New(token)
	srv := server.New(gh, db)

	addr := ":7331"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatal(err)
	}
}
