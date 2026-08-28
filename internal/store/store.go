package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	gh "github.com/harrylawton/pr-review/internal/github"
	_ "modernc.org/sqlite"
)

//go:embed schema/001_init.sql
var schema string

type Store struct {
	db      *sql.DB
	queries *Queries
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("store: migrate: %w", err)
	}
	return &Store{db: db, queries: New(db)}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) SyncPR(ctx context.Context, owner, repo string, pr *gh.PR, files []gh.PRFile) (*Pr, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	draft := int64(0)
	if pr.Draft {
		draft = 1
	}

	record, err := s.queries.UpsertPR(ctx, UpsertPRParams{
		Owner:     owner,
		Repo:      repo,
		Number:    int64(pr.Number),
		Title:     pr.Title,
		Body:      pr.Body,
		State:     pr.State,
		Author:    pr.User.Login,
		HtmlUrl:   pr.HTMLURL,
		Draft:     draft,
		CreatedAt: pr.CreatedAt,
		UpdatedAt: pr.UpdatedAt,
		SyncedAt:  now,
	})
	if err != nil {
		return nil, fmt.Errorf("store: upsert pr: %w", err)
	}

	if err := s.queries.DeletePRFiles(ctx, record.ID); err != nil {
		return nil, fmt.Errorf("store: delete files: %w", err)
	}

	for _, f := range files {
		if _, err := s.queries.InsertPRFile(ctx, InsertPRFileParams{
			PrID:      record.ID,
			Sha:       f.SHA,
			Filename:  f.Filename,
			Status:    f.Status,
			Additions: int64(f.Additions),
			Deletions: int64(f.Deletions),
			Changes:   int64(f.Changes),
			Patch:     f.Patch,
		}); err != nil {
			return nil, fmt.Errorf("store: insert file %s: %w", f.Filename, err)
		}
	}

	return record, nil
}

func (s *Store) ListPRs(ctx context.Context, owner, repo string) ([]*Pr, error) {
	return s.queries.ListPRs(ctx, ListPRsParams{Owner: owner, Repo: repo})
}

func (s *Store) GetPR(ctx context.Context, owner, repo string, number int) (*Pr, error) {
	return s.queries.GetPR(ctx, GetPRParams{Owner: owner, Repo: repo, Number: int64(number)})
}

func (s *Store) GetPRFiles(ctx context.Context, prID int64) ([]*PrFile, error) {
	return s.queries.ListPRFiles(ctx, prID)
}
