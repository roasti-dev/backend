-- +goose Up
CREATE TABLE IF NOT EXISTS beans (
    id          TEXT PRIMARY KEY,
    author_id   TEXT NOT NULL,
    name        TEXT NOT NULL,
    roast_type  TEXT NOT NULL,
    roaster     TEXT NOT NULL,
    country     TEXT,
    region      TEXT,
    farm        TEXT,
    process     TEXT,
    descriptors TEXT,
    q_score     REAL,
    url         TEXT,
    image_id    TEXT,
    created_at  DATETIME NOT NULL,
    deleted_at  DATETIME,
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_beans_author ON beans (author_id);

ALTER TABLE recipes ADD COLUMN bean_id TEXT REFERENCES beans(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE recipes DROP COLUMN bean_id;
DROP INDEX IF EXISTS idx_beans_author;
DROP TABLE IF EXISTS beans;
