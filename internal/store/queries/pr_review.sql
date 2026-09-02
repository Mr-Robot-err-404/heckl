-- name: CreatePRReviewSession :one
INSERT INTO pr_review_sessions (owner, repo, pr_number, head_sha, opencode_session_id, summary, status, error, duration_ms, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListRecentPRReviewSessions :many
SELECT
    s.*,
    (SELECT COUNT(*) FROM concerns c WHERE c.session_id = s.id) AS concern_count,
    (SELECT COUNT(*) FROM concerns c WHERE c.session_id = s.id AND c.severity = 'high') AS high_count
FROM pr_review_sessions s
ORDER BY s.created_at DESC, s.id DESC
LIMIT ?;

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
