-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL UNIQUE,
    avatar_id TEXT,
    bio TEXT,
    created_at DATETIME NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS users;
