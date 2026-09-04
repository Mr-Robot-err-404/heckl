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

func (s *Store) ListAgentConfigs(ctx context.Context) (map[string]AgentConfig, error) {
	rows, err := s.queries.ListAgents(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]AgentConfig, len(rows))
	for _, r := range rows {
		out[r.Name] = AgentConfig{Name: r.Name, Model: r.Model, Prompt: r.Prompt}
	}
	return out, nil
}

func (s *Store) SaveAgentConfigs(ctx context.Context, configs []AgentConfig) error {
	now := time.Now().UTC().Format(time.RFC3339)
	for _, c := range configs {
		if _, err := s.queries.UpsertAgent(ctx, UpsertAgentParams{
			Name:      c.Name,
			Model:     c.Model,
			Prompt:    c.Prompt,
			UpdatedAt: now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateReviewSession(ctx context.Context, in NewReviewSession) (*ReviewSession, error) {
	status := in.Status
	if status == "" {
		status = ReviewStatusDone
	}
	row, err := s.queries.CreatePRReviewSession(ctx, CreatePRReviewSessionParams{
		Owner:             in.Owner,
		Repo:              in.Repo,
		PrNumber:          int64(in.PRNumber),
		HeadSha:           in.HeadSHA,
		OpencodeSessionID: in.OpencodeSessionID,
		Summary:           in.Summary,
		Status:            status,
		Error:             in.Error,
		DurationMs:        in.DurationMS,
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return nil, err
	}
	return toReviewSession(row), nil
}

func (s *Store) ListRecentReviewSessions(ctx context.Context, q HistoryQuery) ([]*RecentReviewSession, error) {
	rows, err := s.recentRows(ctx, q)
	if err != nil {
		return nil, err
	}
	out := make([]*RecentReviewSession, len(rows))
	for i, r := range rows {
		out[i] = &RecentReviewSession{
			ReviewSession: toReviewSession(&PrReviewSession{
				ID:                r.ID,
				Owner:             r.Owner,
				Repo:              r.Repo,
				PrNumber:          r.PrNumber,
				HeadSha:           r.HeadSha,
				OpencodeSessionID: r.OpencodeSessionID,
				Summary:           r.Summary,
				Status:            r.Status,
				Error:             r.Error,
				DurationMs:        r.DurationMs,
				CreatedAt:         r.CreatedAt,
			}),
			ConcernCount: r.ConcernCount,
			HighCount:    r.HighCount,
			MediumCount:  r.MediumCount,
			LowCount:     r.LowCount,
		}
	}
	return out, nil
}

func (s *Store) RepoReviewSummary(ctx context.Context, owner, repo string) (map[int]*RepoReviewSummary, error) {
	rows, err := s.queries.ListRepoReviewSummary(ctx, ListRepoReviewSummaryParams{Owner: owner, Repo: repo})
	if err != nil {
		return nil, err
	}
	out := make(map[int]*RepoReviewSummary, len(rows))
	for _, r := range rows {
		out[int(r.PrNumber)] = &RepoReviewSummary{
			PRNumber:     int(r.PrNumber),
			Status:       r.Status,
			CreatedAt:    r.CreatedAt,
			ConcernCount: r.ConcernCount,
			HighCount:    r.HighCount,
		}
	}
	return out, nil
}

func (s *Store) recentRows(ctx context.Context, q HistoryQuery) ([]*ListRecentPRReviewSessionsRow, error) {
	if q.Owner == "" || q.Repo == "" {
		return s.queries.ListRecentPRReviewSessions(ctx, ListRecentPRReviewSessionsParams{
			Limit:  int64(q.Limit),
			Offset: int64(q.Offset),
		})
	}

	scoped, err := s.queries.ListRecentPRReviewSessionsByRepo(ctx, ListRecentPRReviewSessionsByRepoParams{
		Owner:  q.Owner,
		Repo:   q.Repo,
		Limit:  int64(q.Limit),
		Offset: int64(q.Offset),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*ListRecentPRReviewSessionsRow, len(scoped))
	for i, r := range scoped {
		row := ListRecentPRReviewSessionsRow(*r)
		out[i] = &row
	}
	return out, nil
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

type NewConcern struct {
	SessionID int64
	Agent     string
	File      string
	Line      *int
	Side      string
	Severity  string
	Title     string
	Body      string
}

func (s *Store) CreateConcern(ctx context.Context, c NewConcern) (*ReviewConcern, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	row, err := s.queries.CreateConcern(ctx, CreateConcernParams{
		SessionID: c.SessionID,
		Agent:     c.Agent,
		File:      c.File,
		Line:      intPtrToNullInt64(c.Line),
		Side:      c.Side,
		Severity:  c.Severity,
		Title:     c.Title,
		Body:      c.Body,
		CreatedAt: now,
	})
	if err != nil {
		return nil, err
	}
	return toConcern(row), nil
}

func (s *Store) SaveSessionAgent(ctx context.Context, a SessionAgent) error {
	status := a.Status
	if status == "" {
		status = ReviewStatusDone
	}
	_, err := s.queries.UpsertReviewAgent(ctx, UpsertReviewAgentParams{
		SessionID:         a.SessionID,
		Name:              a.Name,
		Status:            status,
		Error:             a.Error,
		Summary:           a.Summary,
		OpencodeSessionID: a.OpencodeSessionID,
		DurationMs:        a.DurationMS,
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
	})
	return err
}

func (s *Store) ListSessionAgents(ctx context.Context, sessionID int64) ([]SessionAgent, error) {
	rows, err := s.queries.ListReviewAgentsBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	out := make([]SessionAgent, len(rows))
	for i, r := range rows {
		out[i] = SessionAgent{
			SessionID:         r.SessionID,
			Name:              r.Name,
			Status:            r.Status,
			Error:             r.Error,
			Summary:           r.Summary,
			OpencodeSessionID: r.OpencodeSessionID,
			DurationMS:        r.DurationMs,
		}
	}
	return out, nil
}

func (s *Store) SetReviewSessionSummary(ctx context.Context, sessionID int64, summary string) error {
	return s.queries.UpdatePRReviewSessionSummary(ctx, UpdatePRReviewSessionSummaryParams{
		Summary: summary,
		ID:      sessionID,
	})
}

func (s *Store) DeleteConcernsByAgent(ctx context.Context, sessionID int64, agent string) error {
	return s.queries.DeleteConcernsBySessionAgent(ctx, DeleteConcernsBySessionAgentParams{
		SessionID: sessionID,
		Agent:     agent,
	})
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
		Status:            r.Status,
		Error:             r.Error,
		DurationMS:        r.DurationMs,
		CreatedAt:         r.CreatedAt,
	}
}

func toConcern(r *Concern) *ReviewConcern {
	return &ReviewConcern{
		ID:        r.ID,
		SessionID: r.SessionID,
		Agent:     r.Agent,
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
