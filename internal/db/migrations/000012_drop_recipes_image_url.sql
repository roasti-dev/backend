-- +goose Up
ALTER TABLE recipes DROP COLUMN image_url;

-- +goose Down
ALTER TABLE recipes ADD COLUMN image_url TEXT;
