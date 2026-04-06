-- +goose Up
CREATE TABLE IF NOT EXISTS notifications (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    type       TEXT NOT NULL,
    actor_id   TEXT NOT NULL,
    entity_id  TEXT NOT NULL,
    read_at    DATETIME,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (user_id)  REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications (user_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_notifications_user_created;
DROP TABLE IF EXISTS notifications;
