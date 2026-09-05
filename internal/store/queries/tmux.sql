-- name: UpsertTmuxSession :one
INSERT INTO tmux_sessions (owner, repo, pr_number, name, worktree, head_sha, windows, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(owner, repo, pr_number) DO UPDATE SET
    name = excluded.name,
    worktree = excluded.worktree,
    head_sha = excluded.head_sha,
    windows = excluded.windows,
    created_at = excluded.created_at
RETURNING *;

-- name: GetTmuxSession :one
SELECT * FROM tmux_sessions
WHERE owner = ? AND repo = ? AND pr_number = ?;

-- name: DeleteTmuxSession :exec
DELETE FROM tmux_sessions
WHERE owner = ? AND repo = ? AND pr_number = ?;

-- name: ListTmuxSessions :many
SELECT * FROM tmux_sessions ORDER BY created_at DESC;
