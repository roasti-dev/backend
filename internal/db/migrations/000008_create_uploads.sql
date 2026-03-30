-- +goose Up
CREATE TABLE IF NOT EXISTS uploads (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    confirmed BOOLEAN NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS uploads;
