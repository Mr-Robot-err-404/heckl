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

func (s *Store) CreateReviewSession(ctx context.Context, owner, repo string, prNumber int, headSHA, opencodeSessionID, summary string) (*ReviewSession, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	row, err := s.queries.CreatePRReviewSession(ctx, CreatePRReviewSessionParams{
		Owner:             owner,
		Repo:              repo,
		PrNumber:          int64(prNumber),
		HeadSha:           headSHA,
		OpencodeSessionID: opencodeSessionID,
		Summary:           summary,
		CreatedAt:         now,
	})
	if err != nil {
		return nil, err
	}
	return toReviewSession(row), nil
}

func (s *Store) GetReviewSession(ctx context.Context, id int64) (*ReviewSession, error) {
	row, err := s.queries.GetPRReviewSession(ctx, id)
	if err != nil {
		return nil, err
	}
	return toReviewSession(row), nil
}

func (s *Store) ListReviewSessions(ctx context.Context, owner, repo string, prNumber int) ([]*ReviewSession, error) {
	rows, err := s.queries.ListPRReviewSessionsByPR(ctx, ListPRReviewSessionsByPRParams{
		Owner:    owner,
		Repo:     repo,
		PrNumber: int64(prNumber),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*ReviewSession, len(rows))
	for i, r := range rows {
		out[i] = toReviewSession(r)
	}
	return out, nil
}

func (s *Store) CreateConcern(ctx context.Context, sessionID int64, file string, line *int, side, severity, title, body string) (*ReviewConcern, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	row, err := s.queries.CreateConcern(ctx, CreateConcernParams{
		SessionID: sessionID,
		File:      file,
		Line:      intPtrToNullInt64(line),
		Side:      side,
		Severity:  severity,
		Title:     title,
		Body:      body,
		CreatedAt: now,
	})
	if err != nil {
		return nil, err
	}
	return toConcern(row), nil
}

func (s *Store) ListConcerns(ctx context.Context, sessionID int64) ([]*ReviewConcern, error) {
	rows, err := s.queries.ListConcernsBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	out := make([]*ReviewConcern, len(rows))
	for i, r := range rows {
		out[i] = toConcern(r)
	}
	return out, nil
}

func toReviewSession(r *PrReviewSession) *ReviewSession {
	return &ReviewSession{
		ID:                r.ID,
		Owner:             r.Owner,
		Repo:              r.Repo,
		PRNumber:          int(r.PrNumber),
		HeadSHA:           r.HeadSha,
		OpencodeSessionID: r.OpencodeSessionID,
		Summary:           r.Summary,
		CreatedAt:         r.CreatedAt,
	}
}

func toConcern(r *Concern) *ReviewConcern {
	return &ReviewConcern{
		ID:        r.ID,
		SessionID: r.SessionID,
		File:      r.File,
		Line:      nullInt64ToIntPtr(r.Line),
		Side:      r.Side,
		Severity:  r.Severity,
		Title:     r.Title,
		Body:      r.Body,
		CreatedAt: r.CreatedAt,
	}
}

func intPtrToNullInt64(v *int) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*v), Valid: true}
}

func nullInt64ToIntPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}
