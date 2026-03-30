-- +goose Up
ALTER TABLE recipes ADD COLUMN image_id TEXT;
ALTER TABLE brew_steps ADD COLUMN image_id TEXT;

-- +goose Down
ALTER TABLE recipes DROP COLUMN image_id;
ALTER TABLE brew_steps DROP COLUMN image_id;
