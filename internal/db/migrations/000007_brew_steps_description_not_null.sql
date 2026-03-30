-- +goose Up
ALTER TABLE brew_steps RENAME COLUMN description TO description_old;
ALTER TABLE brew_steps ADD COLUMN description TEXT NOT NULL DEFAULT '';
UPDATE brew_steps SET description = COALESCE(description_old, '');
ALTER TABLE brew_steps DROP COLUMN description_old;

-- +goose Down
ALTER TABLE brew_steps RENAME COLUMN description TO description_new;
ALTER TABLE brew_steps ADD COLUMN description TEXT;
UPDATE brew_steps SET description = description_new;
ALTER TABLE brew_steps DROP COLUMN description_new;
