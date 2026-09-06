package main

import (
	"context"
	"database/sql"

	"github.com/harrylawton/pr-review/internal/config"
	"github.com/harrylawton/pr-review/internal/store"
	_ "modernc.org/sqlite"
)

func runMigrate(args []string) {
	if len(args) == 0 {
		fatal("usage: pr-review migrate <up|up-to|down|down-to|redo|status|reset|version> [version]")
	}

	cfg, err := config.Load()
	if err != nil {
		fatal(err.Error())
	}

	db, err := sql.Open("sqlite", cfg.Server.Database)
	if err != nil {
		fatal(err.Error())
	}
	defer db.Close()

	if err := store.RunGoose(context.Background(), db, args[0], args[1:]...); err != nil {
		fatal(err.Error())
	}
}
