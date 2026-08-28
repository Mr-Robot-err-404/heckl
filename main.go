package main

import (
	"log"
	"os"

	"github.com/harrylawton/pr-review/internal/store"
)

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN not set")
	}

	db, err := store.Open("pr-review.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("store ready")
}
