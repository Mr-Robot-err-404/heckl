-- name: CreatePRReviewSession :one
INSERT INTO pr_review_sessions (owner, repo, pr_number, head_sha, opencode_session_id, summary, status, error, duration_ms, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListRecentPRReviewSessions :many
SELECT
    s.*,
    (SELECT COUNT(*) FROM concerns c WHERE c.session_id = s.id) AS concern_count,
    (SELECT COUNT(*) FROM concerns c WHERE c.session_id = s.id AND c.severity = 'high') AS high_count,
    (SELECT COUNT(*) FROM concerns c WHERE c.session_id = s.id AND c.severity = 'medium') AS medium_count,
    (SELECT COUNT(*) FROM concerns c WHERE c.session_id = s.id AND c.severity = 'low') AS low_count
FROM pr_review_sessions s
ORDER BY s.created_at DESC, s.id DESC
LIMIT ? OFFSET ?;

-- name: ListRecentPRReviewSessionsByRepo :many
SELECT
    s.*,
    (SELECT COUNT(*) FROM concerns c WHERE c.session_id = s.id) AS concern_count,
    (SELECT COUNT(*) FROM concerns c WHERE c.session_id = s.id AND c.severity = 'high') AS high_count,
    (SELECT COUNT(*) FROM concerns c WHERE c.session_id = s.id AND c.severity = 'medium') AS medium_count,
    (SELECT COUNT(*) FROM concerns c WHERE c.session_id = s.id AND c.severity = 'low') AS low_count
FROM pr_review_sessions s
WHERE s.owner = ? AND s.repo = ?
ORDER BY s.created_at DESC, s.id DESC
LIMIT ? OFFSET ?;

-- name: GetPRReviewSession :one
SELECT * FROM pr_review_sessions WHERE id = ?;

-- name: ListPRReviewSessionsByPR :many
SELECT * FROM pr_review_sessions WHERE owner = ? AND repo = ? AND pr_number = ? ORDER BY created_at DESC;

-- name: ListRepoReviewSummary :many
SELECT
    s.pr_number,
    s.status,
    s.created_at,
    (SELECT COUNT(*) FROM concerns c WHERE c.session_id = s.id) AS concern_count,
    (SELECT COUNT(*) FROM concerns c WHERE c.session_id = s.id AND c.severity = 'high') AS high_count
FROM pr_review_sessions s
WHERE s.owner = ? AND s.repo = ?
  AND s.id = (
      SELECT s2.id FROM pr_review_sessions s2
      WHERE s2.owner = s.owner AND s2.repo = s.repo AND s2.pr_number = s.pr_number
      ORDER BY s2.created_at DESC, s2.id DESC
      LIMIT 1
  );

-- name: CreateConcern :one
INSERT INTO concerns (session_id, file, line, side, severity, title, body, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListConcernsBySession :many
SELECT * FROM concerns WHERE session_id = ? ORDER BY id ASC;
