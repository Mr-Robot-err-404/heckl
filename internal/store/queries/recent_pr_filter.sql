-- name: GetRecentPRFilter :one
SELECT value FROM recent_pr_filter WHERE id = 1;

-- name: SetRecentPRFilter :exec
INSERT INTO recent_pr_filter (id, value, updated_at)
VALUES (1, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    value = excluded.value,
    updated_at = excluded.updated_at;
