package database

import (
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

func (db *Database) InsertSnapshot(r SnapshotRecord) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.DB.Exec(`
		INSERT INTO snapshots (id, subvolume_id, name, path, frequency, size)
		VALUES (?, ?, ?, ?, ?, ?)
	`, r.ID, r.SubvolumeID, r.Name, r.Path, r.Frequency, r.Size)
	if err != nil {
		return fmt.Errorf("failed to insert snapshot: %w", err)
	}

	return nil
}

func (db *Database) ListSnapshotsBySubvolume(subvolumeID string) ([]SnapshotRecord, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	rows, err := db.DB.Query(`
		SELECT id, subvolume_id, name, path, frequency, size, created_at
		FROM snapshots
		WHERE subvolume_id = ?
		ORDER BY created_at DESC
	`, subvolumeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query snapshots: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []SnapshotRecord
	for rows.Next() {
		var r SnapshotRecord
		var createdAt string

		err := rows.Scan(&r.ID, &r.SubvolumeID, &r.Name, &r.Path, &r.Frequency, &r.Size, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}

		r.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
		if err != nil {
			r.CreatedAt = time.Now()
		}

		records = append(records, r)
	}

	return records, nil
}

func (db *Database) DeleteSnapshot(id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.DB.Exec("DELETE FROM snapshots WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	return nil
}
