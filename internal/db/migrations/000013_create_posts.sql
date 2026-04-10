-- +goose Up
CREATE TABLE IF NOT EXISTS posts (
    id         TEXT PRIMARY KEY,
    author_id  TEXT NOT NULL,
    title      TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS post_blocks (
    id          TEXT PRIMARY KEY,
    post_id     TEXT NOT NULL,
    block_order INTEGER NOT NULL,
    type        TEXT NOT NULL,
    images      TEXT,
    text        TEXT,
    recipe_id   TEXT,
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS comments (
    id          TEXT PRIMARY KEY,
    target_id   TEXT NOT NULL,
    target_type TEXT NOT NULL,
    author_id   TEXT NOT NULL,
    text        TEXT NOT NULL,
    parent_id   TEXT,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL,
    deleted_at  DATETIME,
    FOREIGN KEY (author_id) REFERENCES users(id),
    FOREIGN KEY (parent_id) REFERENCES comments(id)
);

CREATE INDEX IF NOT EXISTS idx_posts_author ON posts (author_id);
CREATE INDEX IF NOT EXISTS idx_post_blocks_post ON post_blocks (post_id);
CREATE INDEX IF NOT EXISTS idx_comments_target ON comments (target_id, target_type);
CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments (parent_id);

-- +goose Down
DROP INDEX IF EXISTS idx_comments_parent;
DROP INDEX IF EXISTS idx_comments_target;
DROP INDEX IF EXISTS idx_post_blocks_post;
DROP INDEX IF EXISTS idx_posts_author;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS post_blocks;
DROP TABLE IF EXISTS posts;
