package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"rosadisk-agent/internal/config"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/storage"
	"rosadisk-agent/internal/worker/event"
)

func runBackup(logger *zap.Logger, db *database.Database, mountpoint, subvolumeID, subvolumeName, subvolPath,
	backupType string, parentID *string, parentSnapshot string, eventBus event.Publisher) (interface{}, error) {

	now := time.Now()
	nowStr := now.Format("02012006-1504")
	backupDir := filepath.Join(mountpoint, ".rosadisk", "backups", fmt.Sprintf("%s-%s", subvolumeName, subvolumeID))
	backupFile := fmt.Sprintf("%s-%s.enc", backupType, nowStr)
	backupPath := filepath.Join(backupDir, backupFile)
	snapshotName := fmt.Sprintf("snapshot-%s-%s", backupType, nowStr)
	snapshotPath := filepath.Join(backupDir, snapshotName)

	logger.Info("creating backup",
		zap.String("type", backupType),
		zap.String("subvolume_id", subvolumeID),
		zap.String("snapshot_path", snapshotPath),
		zap.String("backup_path", backupPath),
	)

	if err := storage.CreateBackupSnapshot(subvolPath, snapshotPath); err != nil {
		logger.Error("failed to create backup snapshot", zap.Error(err))
		return nil, err
	}

	backupID := uuid.New().String()
	if err := db.InsertBackup(database.CreateBackupRecord{
		ID:           backupID,
		SubvolumeID:  subvolumeID,
		Type:         backupType,
		ParentID:     parentID,
		SnapshotPath: snapshotPath,
		Path:         backupPath,
	}); err != nil {
		logger.Error("failed to insert backup record", zap.Error(err))
		return nil, err
	}

	if err := storage.SendBackup(snapshotPath, backupPath, "/var/lib/rosadisk-agent/e2ee_key", parentSnapshot); err != nil {
		if completeErr := db.CompleteBackup(backupID, 0, "", "failed", err.Error()); completeErr != nil {
			logger.Error("failed to update backup record after failure", zap.Error(completeErr))
		}
		logger.Error("backup failed", zap.Error(err))
		return nil, err
	}

	fileInfo, _ := os.Stat(backupPath)
	var fileSize int64
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	cfg, _ := config.GetConfig(db)
	uploadDetails := map[string]string{
		"type":     cfg.BackupStorage.Type,
		"filename": backupFile,
	}
	if cfg.BackupStorage.Type == "local" {
		if p, ok := cfg.BackupStorage.Options["path"]; ok {
			uploadDetails["path"] = filepath.Join(p, fmt.Sprintf("%s-%s", subvolumeName, subvolumeID), backupFile)
		}
	}
	if cfg.BackupStorage.Type == "s3" {
		if ep, ok := cfg.BackupStorage.Options["endpoint"]; ok {
			uploadDetails["endpoint"] = ep
		}
		if b, ok := cfg.BackupStorage.Options["bucket"]; ok {
			uploadDetails["bucket"] = b
		}
		uploadDetails["key"] = fmt.Sprintf("%s-%s/%s", subvolumeName, subvolumeID, backupFile)
	}
	uploadDetailsJSON, _ := json.Marshal(uploadDetails)

	if err := db.CompleteBackup(backupID, fileSize, string(uploadDetailsJSON), "completed", ""); err != nil {
		logger.Error("failed to update backup record", zap.Error(err))
	}

	eventBus.PublishConcurrent(event.ActionBackupUpload, event.BackupUploadRequest{
		ID:         subvolumeID,
		BackupID:   backupID,
		Name:       subvolumeName,
		FilePath:   backupPath,
		Mountpoint: mountpoint,
		BackupType: backupType,
	})

	eventBus.PublishConcurrent(event.ActionBackupCleanup, event.BackupCleanupRequest{
		ID:         subvolumeID,
		Name:       subvolumeName,
		SubvolPath: subvolPath,
		Mountpoint: mountpoint,
	})

	return map[string]string{"status": "success", "backup_id": backupID}, nil
}
