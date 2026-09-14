-- +goose Up
CREATE TABLE IF NOT EXISTS recent_pr_filter (
    id         INTEGER PRIMARY KEY CHECK(id = 1),
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- +goose Down
DROP TABLE recent_pr_filter;
