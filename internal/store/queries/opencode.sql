-- name: TrackOpencodeSession :exec
INSERT INTO opencode_sessions (session_id, owner, repo, pr_number, agent, created_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(session_id) DO NOTHING;

-- name: ForgetOpencodeSession :exec
DELETE FROM opencode_sessions WHERE session_id = ?;

-- name: UnreachableOpencodeSessions :many
SELECT * FROM opencode_sessions
WHERE session_id NOT IN (SELECT opencode_session_id FROM pr_review_sessions WHERE opencode_session_id <> '')
  AND session_id NOT IN (SELECT opencode_session_id FROM review_agents WHERE opencode_session_id <> '')
ORDER BY created_at ASC;
