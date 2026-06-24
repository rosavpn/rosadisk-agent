package database

import (
	"database/sql"
	"fmt"
	"time"
)

type SubvolumeRecord struct {
	ID                         string
	Name                       string
	FsUUID                     string
	Path                       string
	Compression                bool
	QuotaEnabled               bool
	QuotaLimit                 int64
	SnapshotEnabled            bool
	SnapshotFrequency          string
	SnapshotRetention          int
	BackupIncrementalEnabled   bool
	BackupIncrementalFrequency string
	BackupFullEnabled          bool
	BackupFullFrequency        string
	Defrag                     bool
	NFS                        bool
	SMB                        bool
	CreatedAt                  time.Time
}

type CreateSubvolumeRecord struct {
	ID                         string
	Name                       string
	FsUUID                     string
	Path                       string
	Compression                bool
	QuotaEnabled               bool
	QuotaLimit                 int64
	SnapshotEnabled            bool
	SnapshotFrequency          string
	SnapshotRetention          int
	BackupIncrementalEnabled   bool
	BackupIncrementalFrequency string
	BackupFullEnabled          bool
	BackupFullFrequency        string
	Defrag                     bool
	NFS                        bool
	SMB                        bool
}

func (db *Database) ListSubvolumes() ([]SubvolumeRecord, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	rows, err := db.DB.Query(`
		SELECT id, name, fs_uuid, path, compression, quota_enabled, quota_limit,
		       snapshot_enabled, snapshot_frequency, snapshot_retention,
		       backup_incremental_enabled, backup_incremental_frequency,
		       backup_full_enabled, backup_full_frequency,
		       defrag, nfs, smb, created_at
		FROM subvolumes
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query subvolumes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []SubvolumeRecord
	for rows.Next() {
		var r SubvolumeRecord
		var snapshotFreq sql.NullString
		var snapshotRetention sql.NullInt64
		var backupIncEnabled sql.NullBool
		var backupIncFreq sql.NullString
		var backupFullEnabled sql.NullBool
		var backupFullFreq sql.NullString
		var createdAt string

		err := rows.Scan(
			&r.ID, &r.Name, &r.FsUUID, &r.Path, &r.Compression,
			&r.QuotaEnabled, &r.QuotaLimit,
			&r.SnapshotEnabled, &snapshotFreq, &snapshotRetention,
			&backupIncEnabled, &backupIncFreq,
			&backupFullEnabled, &backupFullFreq,
			&r.Defrag, &r.NFS, &r.SMB, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subvolume: %w", err)
		}

		if snapshotFreq.Valid {
			r.SnapshotFrequency = snapshotFreq.String
		}
		if snapshotRetention.Valid {
			r.SnapshotRetention = int(snapshotRetention.Int64)
		}
		if backupIncEnabled.Valid {
			r.BackupIncrementalEnabled = backupIncEnabled.Bool
			r.BackupIncrementalFrequency = backupIncFreq.String
		}
		if backupFullEnabled.Valid {
			r.BackupFullEnabled = backupFullEnabled.Bool
			r.BackupFullFrequency = backupFullFreq.String
		}

		r.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
		if err != nil {
			r.CreatedAt = time.Now()
		}

		records = append(records, r)
	}

	return records, nil
}

func (db *Database) GetSubvolume(id string) (*SubvolumeRecord, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var r SubvolumeRecord
	var snapshotFreq sql.NullString
	var snapshotRetention sql.NullInt64
	var backupIncEnabled sql.NullBool
	var backupIncFreq sql.NullString
	var backupFullEnabled sql.NullBool
	var backupFullFreq sql.NullString
	var createdAt string

	err := db.DB.QueryRow(`
		SELECT id, name, fs_uuid, path, compression, quota_enabled, quota_limit,
		       snapshot_enabled, snapshot_frequency, snapshot_retention,
		       backup_incremental_enabled, backup_incremental_frequency,
		       backup_full_enabled, backup_full_frequency,
		       defrag, nfs, smb, created_at
		FROM subvolumes
		WHERE id = ?
	`, id).Scan(
		&r.ID, &r.Name, &r.FsUUID, &r.Path, &r.Compression,
		&r.QuotaEnabled, &r.QuotaLimit,
		&r.SnapshotEnabled, &snapshotFreq, &snapshotRetention,
		&backupIncEnabled, &backupIncFreq,
		&backupFullEnabled, &backupFullFreq,
		&r.Defrag, &r.NFS, &r.SMB, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("subvolume not found")
		}
		return nil, fmt.Errorf("failed to query subvolume: %w", err)
	}

	if snapshotFreq.Valid {
		r.SnapshotFrequency = snapshotFreq.String
	}
	if snapshotRetention.Valid {
		r.SnapshotRetention = int(snapshotRetention.Int64)
	}
	if backupIncEnabled.Valid {
		r.BackupIncrementalEnabled = backupIncEnabled.Bool
		r.BackupIncrementalFrequency = backupIncFreq.String
	}
	if backupFullEnabled.Valid {
		r.BackupFullEnabled = backupFullEnabled.Bool
		r.BackupFullFrequency = backupFullFreq.String
	}

	r.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
	if err != nil {
		r.CreatedAt = time.Now()
	}

	return &r, nil
}

func (db *Database) InsertSubvolumeRecord(r CreateSubvolumeRecord) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.DB.Exec(`
		INSERT INTO subvolumes (
			id, name, fs_uuid, path, compression, quota_enabled, quota_limit,
			snapshot_enabled, snapshot_frequency, snapshot_retention,
			backup_incremental_enabled, backup_incremental_frequency,
			backup_full_enabled, backup_full_frequency,
			defrag, nfs, smb
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		r.ID, r.Name, r.FsUUID, r.Path, r.Compression,
		r.QuotaEnabled, r.QuotaLimit,
		r.SnapshotEnabled, r.SnapshotFrequency, r.SnapshotRetention,
		r.BackupIncrementalEnabled, r.BackupIncrementalFrequency,
		r.BackupFullEnabled, r.BackupFullFrequency,
		r.Defrag, r.NFS, r.SMB,
	)
	if err != nil {
		return fmt.Errorf("failed to persist subvolume: %w", err)
	}

	return nil
}

func (db *Database) DeleteSubvolumeRecord(id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.DB.Exec("DELETE FROM subvolumes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to remove subvolume from database: %w", err)
	}

	return nil
}
