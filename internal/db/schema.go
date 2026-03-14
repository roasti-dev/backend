package db

import (
	"database/sql"
	"fmt"
	"strings"
)

func InitSchema(db *sql.DB) error {
	queries := []string{
		// `CREATE TABLE IF NOT EXISTS users (
		// 	id TEXT PRIMARY KEY,
		// )`,
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
			description TEXT NOT NULL,
			duration_seconds INTEGER,
			image_id TEXT,
			FOREIGN KEY(recipe_id) REFERENCES recipes(id) ON DELETE CASCADE
		);`,
		`ALTER TABLE recipes ADD COLUMN created_at DATETIME;
		 ALTER TABLE recipes ADD COLUMN updated_at DATETIME;
		 UPDATE recipes SET created_at = datetime('now'), updated_at = datetime('now');`,
		`ALTER TABLE recipes ADD COLUMN image_id TEXT;
		 ALTER TABLE brew_steps ADD COLUMN image_id TEXT;`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			if !strings.Contains(err.Error(), "duplicate column name") {
				return fmt.Errorf("init schema: %w", err)
			}
		}
	}

	return nil
}
