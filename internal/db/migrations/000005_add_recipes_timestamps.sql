-- +goose Up
ALTER TABLE recipes ADD COLUMN created_at DATETIME;
ALTER TABLE recipes ADD COLUMN updated_at DATETIME;
UPDATE recipes SET created_at = datetime('now'), updated_at = datetime('now');

-- +goose Down
ALTER TABLE recipes DROP COLUMN created_at;
ALTER TABLE recipes DROP COLUMN updated_at;
