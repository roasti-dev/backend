-- +goose Up
ALTER TABLE recipes DROP COLUMN likes_count;

-- +goose Down
ALTER TABLE recipes ADD COLUMN likes_count INTEGER NOT NULL DEFAULT 0;
