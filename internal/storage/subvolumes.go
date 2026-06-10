package storage

import (
	"database/sql"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CreateSubvolumeRequest struct {
	Name        string
	FsUUID      string
	Compression bool
	Quota       QuotaConfig
	Snapshots   SnapshotConfig
	Backups     BackupConfig
	Defrag      bool
	NFS         bool
	SMB         bool
}

func ListSubvolumes(db *sql.DB) ([]SubvolumeInfo, error) {
	rows, err := db.Query(`
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

	var subvolumes []SubvolumeInfo
	for rows.Next() {
		var sv SubvolumeInfo
		var quotaLimit sql.NullInt64
		var snapshotFreq sql.NullString
		var snapshotRetention sql.NullInt64
		var backupIncEnabled sql.NullBool
		var backupIncFreq sql.NullString
		var backupFullEnabled sql.NullBool
		var backupFullFreq sql.NullString
		var createdAt string

		err := rows.Scan(
			&sv.ID, &sv.Name, &sv.FsUUID, &sv.Path, &sv.Compression,
			&sv.Quota.Enabled, &quotaLimit,
			&sv.Snapshots.Enabled, &snapshotFreq, &snapshotRetention,
			&backupIncEnabled, &backupIncFreq,
			&backupFullEnabled, &backupFullFreq,
			&sv.Defrag, &sv.NFS, &sv.SMB, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subvolume: %w", err)
		}

		if quotaLimit.Valid {
			sv.Quota.Limit = &quotaLimit.Int64
		}
		if snapshotFreq.Valid {
			sv.Snapshots.Frequency = snapshotFreq.String
		}
		if snapshotRetention.Valid {
			sv.Snapshots.Retention = int(snapshotRetention.Int64)
		}
		if backupIncEnabled.Valid {
			sv.Backups.Incremental = &BackupSchedule{
				Enabled:   backupIncEnabled.Bool,
				Frequency: backupIncFreq.String,
			}
		}
		if backupFullEnabled.Valid {
			sv.Backups.Full = &BackupSchedule{
				Enabled:   backupFullEnabled.Bool,
				Frequency: backupFullFreq.String,
			}
		}

		sv.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
		if err != nil {
			sv.CreatedAt = time.Now()
		}

		subvolumes = append(subvolumes, sv)
	}

	return subvolumes, nil
}

func GetSubvolume(db *sql.DB, id string) (*SubvolumeInfo, error) {
	var sv SubvolumeInfo
	var quotaLimit sql.NullInt64
	var snapshotFreq sql.NullString
	var snapshotRetention sql.NullInt64
	var backupIncEnabled sql.NullBool
	var backupIncFreq sql.NullString
	var backupFullEnabled sql.NullBool
	var backupFullFreq sql.NullString
	var createdAt string

	err := db.QueryRow(`
		SELECT id, name, fs_uuid, path, compression, quota_enabled, quota_limit,
		       snapshot_enabled, snapshot_frequency, snapshot_retention,
		       backup_incremental_enabled, backup_incremental_frequency,
		       backup_full_enabled, backup_full_frequency,
		       defrag, nfs, smb, created_at
		FROM subvolumes
		WHERE id = ?
	`, id).Scan(
		&sv.ID, &sv.Name, &sv.FsUUID, &sv.Path, &sv.Compression,
		&sv.Quota.Enabled, &quotaLimit,
		&sv.Snapshots.Enabled, &snapshotFreq, &snapshotRetention,
		&backupIncEnabled, &backupIncFreq,
		&backupFullEnabled, &backupFullFreq,
		&sv.Defrag, &sv.NFS, &sv.SMB, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("subvolume not found")
		}
		return nil, fmt.Errorf("failed to query subvolume: %w", err)
	}

	if quotaLimit.Valid {
		sv.Quota.Limit = &quotaLimit.Int64
	}
	if snapshotFreq.Valid {
		sv.Snapshots.Frequency = snapshotFreq.String
	}
	if snapshotRetention.Valid {
		sv.Snapshots.Retention = int(snapshotRetention.Int64)
	}
	if backupIncEnabled.Valid {
		sv.Backups.Incremental = &BackupSchedule{
			Enabled:   backupIncEnabled.Bool,
			Frequency: backupIncFreq.String,
		}
	}
	if backupFullEnabled.Valid {
		sv.Backups.Full = &BackupSchedule{
			Enabled:   backupFullEnabled.Bool,
			Frequency: backupFullFreq.String,
		}
	}

	sv.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
	if err != nil {
		sv.CreatedAt = time.Now()
	}

	return &sv, nil
}

func CreateSubvolume(db *sql.DB, req CreateSubvolumeRequest) (*SubvolumeInfo, error) {
	mountpoint, err := findMountpointByUUID(req.FsUUID)
	if err != nil {
		return nil, fmt.Errorf("filesystem not mounted: %w", err)
	}

	subvolPath := filepath.Join(mountpoint, req.Name)

	if err := validateSubvolumeName(req.Name); err != nil {
		return nil, err
	}

	cmd := exec.Command("btrfs", "subvolume", "create", subvolPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create subvolume: %w, output: %s", err, string(output))
	}

	if req.Compression {
		chattrCmd := exec.Command("chattr", "+c", subvolPath)
		if _, err := chattrCmd.CombinedOutput(); err != nil {
			_ = exec.Command("btrfs", "subvolume", "delete", subvolPath).Run()
			return nil, fmt.Errorf("failed to set compression attribute: %w", err)
		}
	}

	if req.Quota.Enabled && req.Quota.Limit != nil {
		_ = exec.Command("btrfs", "quota", "enable", mountpoint).Run()

		qgroupID := fmt.Sprintf("1/0")
		qgroupCmd := exec.Command("btrfs", "qgroup", "create", qgroupID, mountpoint)
		if _, err := qgroupCmd.CombinedOutput(); err != nil {
		}

		limitCmd := exec.Command("btrfs", "qgroup", "limit", fmt.Sprintf("%d", *req.Quota.Limit), subvolPath)
		if output, err := limitCmd.CombinedOutput(); err != nil {
			_ = exec.Command("btrfs", "subvolume", "delete", subvolPath).Run()
			return nil, fmt.Errorf("failed to set quota limit: %w, output: %s", err, string(output))
		}
	}

	id := uuid.New().String()

	sv := &SubvolumeInfo{
		ID:          id,
		Name:        req.Name,
		FsUUID:      req.FsUUID,
		Path:        subvolPath,
		Compression: req.Compression,
		Quota:       req.Quota,
		Snapshots:   req.Snapshots,
		Backups:     req.Backups,
		Defrag:      req.Defrag,
		NFS:         req.NFS,
		SMB:         req.SMB,
		CreatedAt:   time.Now(),
	}

	_, err = db.Exec(`
		INSERT INTO subvolumes (
			id, name, fs_uuid, path, compression, quota_enabled, quota_limit,
			snapshot_enabled, snapshot_frequency, snapshot_retention,
			backup_incremental_enabled, backup_incremental_frequency,
			backup_full_enabled, backup_full_frequency,
			defrag, nfs, smb
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		sv.ID, sv.Name, sv.FsUUID, sv.Path, sv.Compression,
		sv.Quota.Enabled, sv.Quota.Limit,
		sv.Snapshots.Enabled, sv.Snapshots.Frequency, sv.Snapshots.Retention,
		sv.Backups.Incremental != nil && sv.Backups.Incremental.Enabled,
		nullableString(sv.Backups.Incremental, func(s *BackupSchedule) string { return s.Frequency }),
		sv.Backups.Full != nil && sv.Backups.Full.Enabled,
		nullableString(sv.Backups.Full, func(s *BackupSchedule) string { return s.Frequency }),
		sv.Defrag, sv.NFS, sv.SMB,
	)
	if err != nil {
		_ = exec.Command("btrfs", "subvolume", "delete", subvolPath).Run()
		return nil, fmt.Errorf("failed to persist subvolume: %w", err)
	}

	return sv, nil
}

func DeleteSubvolume(db *sql.DB, id string) error {
	sv, err := GetSubvolume(db, id)
	if err != nil {
		return err
	}

	cmd := exec.Command("btrfs", "subvolume", "delete", sv.Path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete subvolume: %w, output: %s", err, string(output))
	}

	_, err = db.Exec("DELETE FROM subvolumes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to remove subvolume from database: %w", err)
	}

	return nil
}

func findMountpointByUUID(fsUUID string) (string, error) {
	mounts, err := ListMounts()
	if err != nil {
		return "", fmt.Errorf("failed to list mounts: %w", err)
	}

	for _, mount := range mounts {
		if mount.UUID == fsUUID {
			return mount.Mountpoint, nil
		}
	}

	return "", fmt.Errorf("filesystem %s is not mounted", fsUUID)
}

func validateSubvolumeName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("subvolume name is required")
	}
	if len(name) > 255 {
		return fmt.Errorf("subvolume name must be at most 255 characters")
	}
	if strings.Contains(name, "/") {
		return fmt.Errorf("subvolume name cannot contain slashes")
	}
	return nil
}

func nullableString[T any](ptr *T, fn func(*T) string) interface{} {
	if ptr == nil {
		return nil
	}
	return fn(ptr)
}
