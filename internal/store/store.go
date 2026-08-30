package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema/*.sql
var Migrations embed.FS

type Store struct {
	db      *sql.DB
	queries *Queries
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	return &Store{db: db, queries: New(db)}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) AddRepo(ctx context.Context, owner, name string) (*Repo, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	return s.queries.AddRepo(ctx, AddRepoParams{Owner: owner, Name: name, AddedAt: now})
}

func (s *Store) ListRepos(ctx context.Context) ([]*Repo, error) {
	return s.queries.ListRepos(ctx)
}

func (s *Store) ListReposByOwner(ctx context.Context, owner string) ([]*Repo, error) {
	return s.queries.ListReposByOwner(ctx, owner)
}

func (s *Store) ListOrgs(ctx context.Context) ([]string, error) {
	return s.queries.ListOrgs(ctx)
}

func (s *Store) DeleteRepo(ctx context.Context, owner, name string) error {
	return s.queries.DeleteRepo(ctx, DeleteRepoParams{Owner: owner, Name: name})
}
