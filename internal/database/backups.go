package database

import (
	"database/sql"
	"fmt"
	"time"
)

type BackupRecord struct {
	ID            string
	SubvolumeID   string
	Type          string
	ParentID      *string
	SnapshotPath  string
	Path          string
	Size          int64
	UploadDetails *string
	Status        string
	Error         *string
	CreatedAt     time.Time
	CompletedAt   *time.Time
}

type CreateBackupRecord struct {
	ID           string
	SubvolumeID  string
	Type         string
	ParentID     *string
	SnapshotPath string
	Path         string
}

func (db *Database) InsertBackup(r CreateBackupRecord) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.DB.Exec(`
		INSERT INTO backups (id, subvolume_id, type, parent_id, snapshot_path, path, status)
		VALUES (?, ?, ?, ?, ?, ?, 'running')
	`, r.ID, r.SubvolumeID, r.Type, r.ParentID, r.SnapshotPath, r.Path)
	if err != nil {
		return fmt.Errorf("failed to insert backup: %w", err)
	}

	return nil
}

func (db *Database) CompleteBackup(id string, size int64, uploadDetails string, status string, errMsg string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.DB.Exec(`
		UPDATE backups SET size = ?, upload_details = ?, status = ?, error = ?, completed_at = ?
		WHERE id = ?
	`, size, nullString(uploadDetails), status, nullString(errMsg), time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update backup: %w", err)
	}

	return nil
}

func (db *Database) ListBackupsBySubvolume(subvolumeID string) ([]BackupRecord, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	rows, err := db.DB.Query(`
		SELECT id, subvolume_id, type, parent_id, snapshot_path, path, size, upload_details, status, error, created_at, completed_at
		FROM backups
		WHERE subvolume_id = ?
		ORDER BY created_at DESC
	`, subvolumeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query backups: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []BackupRecord
	for rows.Next() {
		var r BackupRecord
		var parentID, uploadDetails, errMsg, completedAt sql.NullString
		var createdAt string

		err := rows.Scan(&r.ID, &r.SubvolumeID, &r.Type, &parentID, &r.SnapshotPath,
			&r.Path, &r.Size, &uploadDetails, &r.Status, &errMsg, &createdAt, &completedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan backup: %w", err)
		}

		if parentID.Valid {
			r.ParentID = &parentID.String
		}
		if uploadDetails.Valid {
			r.UploadDetails = &uploadDetails.String
		}
		if errMsg.Valid {
			r.Error = &errMsg.String
		}
		r.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		if completedAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", completedAt.String)
			r.CompletedAt = &t
		}

		records = append(records, r)
	}

	return records, nil
}

func (db *Database) GetLatestBackup(subvolumeID string) (*BackupRecord, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var r BackupRecord
	var parentID, uploadDetails, errMsg, completedAt sql.NullString
	var createdAt string

	err := db.DB.QueryRow(`
		SELECT id, subvolume_id, type, parent_id, snapshot_path, path, size, upload_details, status, error, created_at, completed_at
		FROM backups
		WHERE subvolume_id = ? AND status = 'completed'
		ORDER BY created_at DESC
		LIMIT 1
	`, subvolumeID).Scan(&r.ID, &r.SubvolumeID, &r.Type, &parentID, &r.SnapshotPath,
		&r.Path, &r.Size, &uploadDetails, &r.Status, &errMsg, &createdAt, &completedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query latest backup: %w", err)
	}

	if parentID.Valid {
		r.ParentID = &parentID.String
	}
	if uploadDetails.Valid {
		r.UploadDetails = &uploadDetails.String
	}
	if errMsg.Valid {
		r.Error = &errMsg.String
	}
	r.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	if completedAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", completedAt.String)
		r.CompletedAt = &t
	}

	return &r, nil
}

func (db *Database) DeleteBackup(id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.DB.Exec("DELETE FROM backups WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	return nil
}
