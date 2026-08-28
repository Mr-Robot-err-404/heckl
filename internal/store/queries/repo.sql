-- name: AddRepo :one
INSERT INTO repos (owner, name, added_at)
VALUES (?, ?, ?)
ON CONFLICT(owner, name) DO UPDATE SET added_at = excluded.added_at
RETURNING *;

-- name: ListRepos :many
SELECT * FROM repos ORDER BY added_at DESC;

-- name: DeleteRepo :exec
DELETE FROM repos WHERE owner = ? AND name = ?;
