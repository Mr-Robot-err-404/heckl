-- name: UpsertPR :one
INSERT INTO prs (owner, repo, number, title, body, state, author, html_url, draft, created_at, updated_at, synced_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(owner, repo, number) DO UPDATE SET
    title      = excluded.title,
    body       = excluded.body,
    state      = excluded.state,
    draft      = excluded.draft,
    updated_at = excluded.updated_at,
    synced_at  = excluded.synced_at
RETURNING *;

-- name: ListPRs :many
SELECT * FROM prs
WHERE owner = ? AND repo = ?
ORDER BY updated_at DESC;

-- name: GetPR :one
SELECT * FROM prs
WHERE owner = ? AND repo = ? AND number = ?
LIMIT 1;

-- name: InsertPRFile :one
INSERT INTO pr_files (pr_id, sha, filename, status, additions, deletions, changes, patch)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: DeletePRFiles :exec
DELETE FROM pr_files WHERE pr_id = ?;

-- name: ListPRFiles :many
SELECT * FROM pr_files
WHERE pr_id = ?
ORDER BY filename;
