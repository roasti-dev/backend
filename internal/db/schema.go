package db

import (
	"database/sql"
	"fmt"
	"strings"
)

func InitSchema(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL UNIQUE,
			avatar_id TEXT,
			bio TEXT,
			created_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS recipes (
			id TEXT PRIMARY KEY,
			author_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL,
			image_url TEXT,
			image_id TEXT,
			brew_method TEXT NOT NULL,
			difficulty TEXT NOT NULL,
			roast_level TEXT,
			beans TEXT,
			public BOOLEAN NOT NULL DEFAULT 0
			-- FOREIGN KEY(author_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS brew_steps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			recipe_id TEXT NOT NULL,
			step_order INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			duration_seconds INTEGER,
			image_id TEXT,
			FOREIGN KEY(recipe_id) REFERENCES recipes(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS revoked_tokens (
         	token_hash TEXT PRIMARY KEY,
         	revoked_at DATETIME NOT NULL,
         	expires_at DATETIME GENERATED ALWAYS AS (datetime(revoked_at, '+90 days')) VIRTUAL
	    );`,

		`ALTER TABLE recipes ADD COLUMN created_at DATETIME;
		 ALTER TABLE recipes ADD COLUMN updated_at DATETIME;
		 UPDATE recipes SET created_at = datetime('now'), updated_at = datetime('now');`,

		`ALTER TABLE recipes ADD COLUMN image_id TEXT;
		 ALTER TABLE brew_steps ADD COLUMN image_id TEXT;`,

		`ALTER TABLE brew_steps RENAME COLUMN description TO description_old;
		 ALTER TABLE brew_steps ADD COLUMN description TEXT NOT NULL DEFAULT '';
		 UPDATE brew_steps SET description = description_old;
		 ALTER TABLE brew_steps DROP COLUMN description_old;`,

		`CREATE TABLE IF NOT EXISTS uploads (
    	 	id TEXT PRIMARY KEY,
    	 	path TEXT NOT NULL,
    	 	mime_type TEXT NOT NULL,
    	 	created_at DATETIME NOT NULL,
    	 	confirmed BOOLEAN NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS likes (
			id          TEXT PRIMARY KEY,
			user_id     TEXT NOT NULL,
			target_id   TEXT NOT NULL,
			target_type TEXT NOT NULL,
			created_at  DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE (user_id, target_id, target_type)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_likes_target ON likes (target_id, target_type);`,
		`ALTER TABLE recipes ADD COLUMN origin_recipe_id TEXT REFERENCES recipes(id) ON DELETE SET NULL;`,
		`ALTER TABLE recipes ADD COLUMN note TEXT;`,
		`ALTER TABLE recipes DROP COLUMN likes_count;`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			msg := err.Error()
			if !strings.Contains(msg, "duplicate column name") &&
				!strings.Contains(msg, "no such column") {
				return fmt.Errorf("init schema: %w", err)
			}
		}
	}

	return nil
}
