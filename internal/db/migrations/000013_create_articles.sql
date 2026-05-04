-- +goose Up
CREATE TABLE IF NOT EXISTS articles (
    id         TEXT PRIMARY KEY,
    author_id  TEXT NOT NULL,
    title      TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS article_blocks (
    id          TEXT PRIMARY KEY,
    article_id     TEXT NOT NULL,
    block_order INTEGER NOT NULL,
    type        TEXT NOT NULL,
    images      TEXT,
    text        TEXT,
    recipe_id   TEXT,
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE
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

CREATE INDEX IF NOT EXISTS idx_articles_author ON articles (author_id);
CREATE INDEX IF NOT EXISTS idx_article_blocks_article ON article_blocks (article_id);
CREATE INDEX IF NOT EXISTS idx_comments_target ON comments (target_id, target_type);
CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments (parent_id);

-- +goose Down
DROP INDEX IF EXISTS idx_comments_parent;
DROP INDEX IF EXISTS idx_comments_target;
DROP INDEX IF EXISTS idx_article_blocks_article;
DROP INDEX IF EXISTS idx_articles_author;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS article_blocks;
DROP TABLE IF EXISTS articles;
