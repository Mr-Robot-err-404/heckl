-- name: ListAgents :many
SELECT * FROM agents ORDER BY name ASC;

-- name: UpsertAgent :one
INSERT INTO agents (name, model, prompt, updated_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
    model = excluded.model,
    prompt = excluded.prompt,
    updated_at = excluded.updated_at
RETURNING *;
