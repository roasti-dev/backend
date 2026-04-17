-- +goose Up
CREATE TABLE IF NOT EXISTS follows (
    id          TEXT PRIMARY KEY,
    follower_id TEXT NOT NULL,
    target_id   TEXT NOT NULL,
    target_type TEXT NOT NULL,
    created_at  DATETIME NOT NULL,
    UNIQUE (follower_id, target_id, target_type),
    FOREIGN KEY (follower_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_follows_follower ON follows (follower_id, target_type);
CREATE INDEX IF NOT EXISTS idx_follows_target ON follows (target_id, target_type);

-- +goose Down
DROP INDEX IF EXISTS idx_follows_target;
DROP INDEX IF EXISTS idx_follows_follower;
DROP TABLE IF EXISTS follows;
