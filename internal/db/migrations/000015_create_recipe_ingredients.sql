-- +goose Up
CREATE TABLE IF NOT EXISTS recipe_ingredients (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    recipe_id TEXT    NOT NULL,
    position  INTEGER NOT NULL,
    name      TEXT    NOT NULL,
    amount    REAL,
    unit      TEXT,
    FOREIGN KEY(recipe_id) REFERENCES recipes(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS recipe_ingredients;
