-- +goose Up
CREATE TABLE comments_new (
    id          TEXT PRIMARY KEY,
    target_id   TEXT NOT NULL,
    target_type TEXT NOT NULL CHECK (target_type IN ('post', 'recipe')),
    author_id   TEXT NOT NULL,
    text        TEXT NOT NULL,
    parent_id   TEXT,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL,
    deleted_at  DATETIME,
    FOREIGN KEY (author_id) REFERENCES users(id),
    FOREIGN KEY (parent_id) REFERENCES comments(id)
);

INSERT INTO comments_new SELECT id, target_id, target_type, author_id, text, NULL, created_at, created_at, NULL FROM comments;

DROP TABLE comments;
ALTER TABLE comments_new RENAME TO comments;

CREATE INDEX IF NOT EXISTS idx_comments_target ON comments (target_id, target_type);
CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments (parent_id);

-- +goose Down
CREATE TABLE comments_old (
    id          TEXT PRIMARY KEY,
    target_id   TEXT NOT NULL,
    target_type TEXT NOT NULL,
    author_id   TEXT NOT NULL,
    text        TEXT NOT NULL,
    created_at  DATETIME NOT NULL,
    FOREIGN KEY (author_id) REFERENCES users(id)
);

INSERT INTO comments_old SELECT id, target_id, target_type, author_id, text, created_at FROM comments;

DROP TABLE comments;
ALTER TABLE comments_old RENAME TO comments;

CREATE INDEX IF NOT EXISTS idx_comments_target ON comments (target_id, target_type);
