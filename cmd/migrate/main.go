package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"

	"github.com/harrylawton/pr-review/internal/store"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func main() {
	dbPath := flag.String("db", "pr-review.db", "path to sqlite database")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("usage: migrate [-db path] <up|down|status|reset|version>")
	}
	command := args[0]

	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	goose.SetBaseFS(store.Migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatal(err)
	}

	if err := goose.RunContext(context.Background(), command, db, "schema"); err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}
