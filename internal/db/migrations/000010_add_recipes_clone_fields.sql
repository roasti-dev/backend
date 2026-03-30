-- +goose Up
ALTER TABLE recipes ADD COLUMN origin_recipe_id TEXT REFERENCES recipes(id) ON DELETE SET NULL;
ALTER TABLE recipes ADD COLUMN note TEXT;

-- +goose Down
ALTER TABLE recipes DROP COLUMN origin_recipe_id;
ALTER TABLE recipes DROP COLUMN note;
