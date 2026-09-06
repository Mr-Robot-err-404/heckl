package main

import (
	"context"
	"database/sql"

	"github.com/Mr-Robot-err-404/heckl/internal/config"
	"github.com/Mr-Robot-err-404/heckl/internal/store"
	_ "modernc.org/sqlite"
)

func runMigrate(args []string) {
	if len(args) == 0 {
		fatal("usage: heckl migrate <up|up-to|down|down-to|redo|status|reset|version> [version]")
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
