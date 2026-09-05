-- +goose Up
CREATE TABLE IF NOT EXISTS tmux_sessions (
    owner         TEXT NOT NULL,
    repo          TEXT NOT NULL,
    pr_number     INTEGER NOT NULL,
    name          TEXT NOT NULL,
    worktree      TEXT NOT NULL,
    head_sha      TEXT NOT NULL,
    windows       INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT NOT NULL,
    PRIMARY KEY (owner, repo, pr_number)
);

-- +goose Down
DROP TABLE tmux_sessions;
