-- +goose Up
CREATE TABLE IF NOT EXISTS recipes (
    id TEXT PRIMARY KEY,
    author_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    image_url TEXT,
    brew_method TEXT NOT NULL,
    difficulty TEXT NOT NULL,
    roast_level TEXT,
    beans TEXT,
    public BOOLEAN NOT NULL DEFAULT 0,
    FOREIGN KEY(author_id) REFERENCES users(id)
);

-- +goose Down
DROP TABLE IF EXISTS recipes;
