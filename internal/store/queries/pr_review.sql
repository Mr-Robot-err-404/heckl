-- name: CreatePRReviewSession :one
INSERT INTO pr_review_sessions (owner, repo, pr_number, head_sha, opencode_session_id, summary, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetPRReviewSession :one
SELECT * FROM pr_review_sessions WHERE id = ?;

-- name: ListPRReviewSessionsByPR :many
SELECT * FROM pr_review_sessions WHERE owner = ? AND repo = ? AND pr_number = ? ORDER BY created_at DESC;

-- name: CreateConcern :one
INSERT INTO concerns (session_id, file, line, side, severity, title, body, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListConcernsBySession :many
SELECT * FROM concerns WHERE session_id = ? ORDER BY id ASC;
