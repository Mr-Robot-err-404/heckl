-- +goose Up
CREATE TABLE IF NOT EXISTS agents (
    name          TEXT PRIMARY KEY,
    model         TEXT NOT NULL DEFAULT '',
    prompt        TEXT NOT NULL DEFAULT '',
    updated_at    TEXT NOT NULL
);

-- +goose Down
DROP TABLE agents;
