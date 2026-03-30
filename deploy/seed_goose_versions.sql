-- Run this once on an existing database before deploying goose-based migrations.
-- This tells goose that all historical migrations have already been applied.
--
-- Usage: sqlite3 /var/lib/roasti/data.db < deploy/seed_goose_versions.sql

CREATE TABLE IF NOT EXISTS goose_db_version (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    version_id INTEGER NOT NULL,
    is_applied BOOLEAN NOT NULL,
    tstamp TIMESTAMP DEFAULT (datetime('now'))
);

INSERT OR IGNORE INTO goose_db_version (version_id, is_applied) VALUES
    (0, 1),
    (1, 1),
    (2, 1),
    (3, 1),
    (4, 1),
    (5, 1),
    (6, 1),
    (7, 1),
    (8, 1),
    (9, 1),
    (10, 1),
    (11, 1);
