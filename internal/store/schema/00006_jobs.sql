-- +goose Up
CREATE TABLE IF NOT EXISTS job_runs (
    name        TEXT PRIMARY KEY,
    last_run_at TEXT NOT NULL,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    status      TEXT NOT NULL,
    detail      TEXT NOT NULL DEFAULT ''
);

-- +goose Down
DROP TABLE job_runs;
