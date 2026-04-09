package db

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"foundry/backend/appdata"

	_ "modernc.org/sqlite"
)

var instance *sql.DB

// Open opens the SQLite database and runs migrations.
func Open() error {
	dbPath := filepath.Join(appdata.Path(), "foundry.db")
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", dbPath)

	var err error
	instance, err = sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("db: open: %w", err)
	}

	if err := instance.Ping(); err != nil {
		return fmt.Errorf("db: ping: %w", err)
	}

	if err := migrate(); err != nil {
		return fmt.Errorf("db: migrate: %w", err)
	}

	return nil
}

// Close closes the database connection.
func Close() error {
	if instance != nil {
		return instance.Close()
	}
	return nil
}

// DB returns the package-level database handle.
func DB() *sql.DB {
	return instance
}

func migrate() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS installations (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			path_hash     TEXT    NOT NULL UNIQUE,
			project_path  TEXT    NOT NULL,
			project_name  TEXT    NOT NULL,
			repository    TEXT    NOT NULL,
			site_name     TEXT    NOT NULL,
			db_name       TEXT    NOT NULL,
			installed_at  TEXT    NOT NULL,
			updated_at    TEXT    NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS installed_features (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			installation_id INTEGER NOT NULL REFERENCES installations(id) ON DELETE CASCADE,
			feature_id      TEXT    NOT NULL,
			feature_name    TEXT    NOT NULL,
			config_values   TEXT    NOT NULL DEFAULT '{}',
			installed_at    TEXT    NOT NULL
		)`,
	}

	for _, stmt := range statements {
		if _, err := instance.Exec(stmt); err != nil {
			return fmt.Errorf("db: migrate: %w", err)
		}
	}

	return nil
}
