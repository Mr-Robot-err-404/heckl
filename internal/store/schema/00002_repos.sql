-- +goose Up
CREATE TABLE IF NOT EXISTS repos (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    owner       TEXT NOT NULL,
    name        TEXT NOT NULL,
    added_at    TEXT NOT NULL,
    UNIQUE(owner, name)
);

-- +goose Down
DROP TABLE repos;
