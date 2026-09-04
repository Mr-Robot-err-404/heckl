-- +goose Up
CREATE TABLE IF NOT EXISTS theme (
    id            INTEGER PRIMARY KEY CHECK(id = 1),
    name          TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

-- +goose Down
DROP TABLE theme;
