-- +goose Up
CREATE TABLE IF NOT EXISTS brew_steps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    recipe_id TEXT NOT NULL,
    step_order INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    duration_seconds INTEGER,
    FOREIGN KEY(recipe_id) REFERENCES recipes(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS brew_steps;
