package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func InitDB(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		return nil, fmt.Errorf("failed to set journal mode: %w", err)
	}

	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS subvolumes (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			fs_uuid TEXT NOT NULL,
			path TEXT NOT NULL UNIQUE,
			compression INTEGER DEFAULT 0,
			quota_enabled INTEGER DEFAULT 0,
			quota_limit INTEGER,
			snapshot_enabled INTEGER DEFAULT 0,
			snapshot_frequency TEXT,
			snapshot_retention INTEGER,
			backup_incremental_enabled INTEGER DEFAULT 0,
			backup_incremental_frequency TEXT,
			backup_full_enabled INTEGER DEFAULT 0,
			backup_full_frequency TEXT,
			defrag INTEGER DEFAULT 0,
			nfs INTEGER DEFAULT 0,
			smb INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS global_config (
			id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
			data TEXT NOT NULL DEFAULT '{}'
		)`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}

	return nil
}
