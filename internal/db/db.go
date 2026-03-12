package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func NewSQLite(file string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", file)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return db, nil
}

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
			FOREIGN KEY(recipe_id) REFERENCES recipes(id) ON DELETE CASCADE
		);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}

	_, err := db.Exec(`ALTER TABLE recipes ADD COLUMN public BOOLEAN NOT NULL DEFAULT 0;`)
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return err
		}
	}

	return nil
}
