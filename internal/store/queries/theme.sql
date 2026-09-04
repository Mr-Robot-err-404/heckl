-- name: GetTheme :one
SELECT name FROM theme WHERE id = 1;

-- name: SetTheme :exec
INSERT INTO theme (id, name, updated_at)
VALUES (1, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    updated_at = excluded.updated_at;
