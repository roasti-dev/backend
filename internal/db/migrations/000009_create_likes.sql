-- +goose Up
CREATE TABLE IF NOT EXISTS likes (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    target_id   TEXT NOT NULL,
    target_type TEXT NOT NULL,
    created_at  DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (user_id, target_id, target_type)
);

CREATE INDEX IF NOT EXISTS idx_likes_target ON likes (target_id, target_type);

ALTER TABLE recipes ADD COLUMN likes_count INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE recipes DROP COLUMN likes_count;
DROP INDEX IF EXISTS idx_likes_target;
DROP TABLE IF EXISTS likes;
