-- +goose Up
CREATE TABLE IF NOT EXISTS opencode_sessions (
    session_id  TEXT PRIMARY KEY,
    owner       TEXT NOT NULL,
    repo        TEXT NOT NULL,
    pr_number   INTEGER NOT NULL,
    agent       TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL
);

-- +goose Down
DROP TABLE opencode_sessions;
