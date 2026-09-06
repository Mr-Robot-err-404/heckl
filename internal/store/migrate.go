package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3"
)

const migrationDir = "schema"

func prepareGoose(quiet bool) error {
	goose.SetBaseFS(Migrations)
	if quiet {
		goose.SetLogger(goose.NopLogger())
	}
	return goose.SetDialect("sqlite3")
}

func Migrate(ctx context.Context, db *sql.DB) error {
	if err := prepareGoose(true); err != nil {
		return err
	}
	if err := goose.UpContext(ctx, db, migrationDir); err != nil {
		return fmt.Errorf("store: migrate: %w", err)
	}
	return nil
}

func RunGoose(ctx context.Context, db *sql.DB, command string, args ...string) error {
	if err := prepareGoose(false); err != nil {
		return err
	}
	return goose.RunContext(ctx, command, db, migrationDir, args...)
}

func PendingMigrations(db *sql.DB) (int, error) {
	if err := prepareGoose(true); err != nil {
		return 0, err
	}
	current, err := goose.EnsureDBVersion(db)
	if err != nil {
		return 0, fmt.Errorf("store: read schema version: %w", err)
	}
	pending, err := goose.CollectMigrations(migrationDir, current, goose.MaxVersion)
	if errors.Is(err, goose.ErrNoMigrationFiles) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("store: collect migrations: %w", err)
	}
	return len(pending), nil
}
