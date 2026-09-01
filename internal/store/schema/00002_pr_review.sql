-- +goose Up
CREATE TABLE IF NOT EXISTS pr_review_sessions (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    owner                 TEXT NOT NULL,
    repo                  TEXT NOT NULL,
    pr_number             INTEGER NOT NULL,
    head_sha              TEXT NOT NULL,
    opencode_session_id   TEXT NOT NULL,
    summary               TEXT NOT NULL DEFAULT '',
    created_at            TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS concerns (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id    INTEGER NOT NULL REFERENCES pr_review_sessions(id) ON DELETE CASCADE,
    file          TEXT NOT NULL,
    line          INTEGER,
    side          TEXT NOT NULL DEFAULT '' CHECK(side IN ('', 'additions', 'deletions')),
    severity      TEXT NOT NULL CHECK(severity IN ('low', 'medium', 'high')),
    title         TEXT NOT NULL,
    body          TEXT NOT NULL,
    created_at    TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_concerns_session_id ON concerns(session_id);

-- +goose Down
DROP TABLE concerns;
DROP TABLE pr_review_sessions;
