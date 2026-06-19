package database

import (
	"database/sql"
	"fmt"
	"time"
)

type SnapshotRecord struct {
	ID          string
	SubvolumeID string
	Name        string
	Path        string
	Frequency   string
	Size        int64
	CreatedAt   time.Time
}

func InsertSnapshot(db *sql.DB, r SnapshotRecord) error {
	_, err := db.Exec(`
		INSERT INTO snapshots (id, subvolume_id, name, path, frequency, size, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, r.ID, r.SubvolumeID, r.Name, r.Path, r.Frequency, r.Size, r.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert snapshot: %w", err)
	}
	return nil
}

func ListSnapshotsBySubvolume(db *sql.DB, subvolumeID string) ([]SnapshotRecord, error) {
	rows, err := db.Query(`
		SELECT id, subvolume_id, name, path, frequency, size, created_at
		FROM snapshots
		WHERE subvolume_id = ?
		ORDER BY created_at ASC
	`, subvolumeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query snapshots: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []SnapshotRecord
	for rows.Next() {
		var r SnapshotRecord
		err := rows.Scan(&r.ID, &r.SubvolumeID, &r.Name, &r.Path, &r.Frequency, &r.Size, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}
		records = append(records, r)
	}
	return records, nil
}

func DeleteSnapshotRecord(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM snapshots WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot record: %w", err)
	}
	return nil
}
