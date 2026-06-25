package database

import (
	"database/sql"
	"fmt"
	"strings"
)

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
		`CREATE TABLE IF NOT EXISTS job_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_type TEXT NOT NULL,
			mountpoint TEXT,
			subvolume_id TEXT,
			target_name TEXT,
			status TEXT NOT NULL,
			output TEXT,
			error TEXT,
			started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS snapshots (
			id TEXT PRIMARY KEY,
			subvolume_id TEXT NOT NULL,
			name TEXT NOT NULL,
			path TEXT NOT NULL UNIQUE,
			frequency TEXT NOT NULL,
			size INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}

	if _, err := db.Exec(`ALTER TABLE subvolumes ADD COLUMN defrag_frequency TEXT`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("migration defrag_frequency failed: %w", err)
		}
	}

	return nil
}
