-- name: AddRepo :one
INSERT INTO repos (owner, name, added_at)
VALUES (?, ?, ?)
ON CONFLICT(owner, name) DO UPDATE SET added_at = excluded.added_at
RETURNING *;

-- name: ListRepos :many
SELECT * FROM repos ORDER BY added_at DESC;

-- name: DeleteRepo :exec
DELETE FROM repos WHERE owner = ? AND name = ?;

-- name: ListOrgs :many
SELECT DISTINCT owner FROM repos ORDER BY owner ASC;

-- name: ListReposByOwner :many
SELECT * FROM repos WHERE owner = ? ORDER BY name ASC;
