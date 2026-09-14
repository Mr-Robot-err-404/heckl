-- name: GetJobRun :one
SELECT * FROM job_runs WHERE name = ?;

-- name: ListJobRuns :many
SELECT * FROM job_runs ORDER BY name ASC;

-- name: RecordJobRun :one
INSERT INTO job_runs (name, last_run_at, duration_ms, status, detail)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
    last_run_at = excluded.last_run_at,
    duration_ms = excluded.duration_ms,
    status = excluded.status,
    detail = excluded.detail
RETURNING *;
