CREATE TABLE IF NOT EXISTS prs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    owner       TEXT NOT NULL,
    repo        TEXT NOT NULL,
    number      INTEGER NOT NULL,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL DEFAULT '',
    state       TEXT NOT NULL,
    author      TEXT NOT NULL,
    html_url    TEXT NOT NULL,
    draft       INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    synced_at   TEXT NOT NULL,
    UNIQUE(owner, repo, number)
);

CREATE TABLE IF NOT EXISTS pr_files (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    pr_id       INTEGER NOT NULL REFERENCES prs(id) ON DELETE CASCADE,
    sha         TEXT NOT NULL,
    filename    TEXT NOT NULL,
    status      TEXT NOT NULL,
    additions   INTEGER NOT NULL DEFAULT 0,
    deletions   INTEGER NOT NULL DEFAULT 0,
    changes     INTEGER NOT NULL DEFAULT 0,
    patch       TEXT NOT NULL DEFAULT ''
);
