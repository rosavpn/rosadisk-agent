package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"rosadisk-agent/internal/config"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/storage"
	"rosadisk-agent/internal/worker/event"
)

type BackupCheckHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewBackupCheckHandler(logger *zap.Logger, db *database.Database) *BackupCheckHandler {
	return &BackupCheckHandler{
		logger: logger,
		db:     db,
	}
}

func (h *BackupCheckHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.BackupCheckRequest)
	if !ok {
		h.logger.Error("invalid backup check request type")
		return nil, errInvalidRequest
	}

	h.logger.Info("handling backup check event")

	now := time.Now()
	nowHHMM := now.Format("15:04")
	weekday := strings.ToLower(now.Weekday().String())
	day := now.Day()

	subvolumes, err := h.db.ListSubvolumes()
	if err != nil {
		h.logger.Error("failed to list subvolumes for backup", zap.Error(err))
		return nil, err
	}

	fullCount := 0
	incCount := 0

	for _, sv := range subvolumes {
		if !sv.BackupFullEnabled {
			continue
		}

		mountpoint, err := storage.FindMountpointByUUID(sv.FsUUID)
		if err != nil {
			h.logger.Warn("filesystem not mounted for backup subvolume",
				zap.String("subvolume_id", sv.ID),
				zap.String("fs_uuid", sv.FsUUID),
				zap.Error(err),
			)
			continue
		}

		if backupDue(sv.BackupFullFrequency, req.Schedule, nowHHMM, weekday, day) {
			req.EventBus.PublishAsync(event.ActionBackupFull, event.BackupFullRequest{
				ID:         sv.ID,
				Name:       sv.Name,
				FsUUID:     sv.FsUUID,
				SubvolPath: sv.Path,
				Mountpoint: mountpoint,
				Frequency:  sv.BackupFullFrequency,
				EventBus:   req.EventBus,
			})
			fullCount++
			continue
		}

		if sv.BackupIncrementalEnabled && backupDue(sv.BackupIncrementalFrequency, req.Schedule, nowHHMM, weekday, day) {
			req.EventBus.PublishAsync(event.ActionBackupIncremental, event.BackupIncrementalRequest{
				ID:         sv.ID,
				Name:       sv.Name,
				FsUUID:     sv.FsUUID,
				SubvolPath: sv.Path,
				Mountpoint: mountpoint,
				Frequency:  sv.BackupIncrementalFrequency,
				EventBus:   req.EventBus,
			})
			incCount++
		}
	}

	return map[string]interface{}{
		"status":            "backup jobs dispatched",
		"full_count":        fullCount,
		"incremental_count": incCount,
	}, nil
}

type BackupIncrementalHandler struct {
	logger *zap.Logger
}

func NewBackupIncrementalHandler(logger *zap.Logger) *BackupIncrementalHandler {
	return &BackupIncrementalHandler{
		logger: logger,
	}
}

func (h *BackupIncrementalHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.BackupIncrementalRequest)
	if !ok {
		h.logger.Error("invalid backup incremental request type")
		return nil, errInvalidRequest
	}

	h.logger.Info("backup incremental",
		zap.String("subvolume_id", req.ID),
		zap.String("name", req.Name),
		zap.String("mountpoint", req.Mountpoint),
	)

	req.EventBus.PublishConcurrent(event.ActionBackupUpload, event.BackupUploadRequest{
		ID:         req.ID,
		Name:       req.Name,
		SubvolPath: req.SubvolPath,
		Mountpoint: req.Mountpoint,
		BackupType: "incremental",
	})

	req.EventBus.PublishConcurrent(event.ActionBackupCleanup, event.BackupCleanupRequest{
		ID:         req.ID,
		Name:       req.Name,
		SubvolPath: req.SubvolPath,
		Mountpoint: req.Mountpoint,
	})

	return map[string]string{"status": "success"}, nil
}

type BackupFullHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewBackupFullHandler(logger *zap.Logger, db *database.Database) *BackupFullHandler {
	return &BackupFullHandler{
		logger: logger,
		db:     db,
	}
}

func (h *BackupFullHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.BackupFullRequest)
	if !ok {
		h.logger.Error("invalid backup full request type")
		return nil, errInvalidRequest
	}

	now := time.Now()
	nowStr := now.Format("02012006-1504")
	backupDir := filepath.Join(req.Mountpoint, ".rosadisk", "backups", fmt.Sprintf("%s-%s", req.Name, req.ID))
	backupFile := fmt.Sprintf("full-%s.enc", nowStr)
	backupPath := filepath.Join(backupDir, backupFile)
	snapshotName := fmt.Sprintf("snapshot-full-%s", nowStr)
	snapshotPath := filepath.Join(backupDir, snapshotName)

	h.logger.Info("creating full backup",
		zap.String("subvolume_id", req.ID),
		zap.String("snapshot_path", snapshotPath),
		zap.String("backup_path", backupPath),
	)

	if err := storage.CreateBackupSnapshot(req.SubvolPath, snapshotPath); err != nil {
		h.logger.Error("failed to create backup snapshot", zap.Error(err))
		return nil, err
	}

	backupID := uuid.New().String()
	if err := h.db.InsertBackup(database.CreateBackupRecord{
		ID:           backupID,
		SubvolumeID:  req.ID,
		Type:         "full",
		ParentID:     nil,
		SnapshotName: snapshotName,
		Path:         backupPath,
	}); err != nil {
		h.logger.Error("failed to insert backup record", zap.Error(err))
		return nil, err
	}

	if err := storage.SendBackup(snapshotPath, backupPath, "/var/lib/rosadisk-agent/e2ee_key"); err != nil {
		h.db.CompleteBackup(backupID, 0, "", "failed", err.Error())
		h.logger.Error("full backup failed", zap.Error(err))
		return nil, err
	}

	fileInfo, _ := os.Stat(backupPath)
	var fileSize int64
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	cfg, _ := config.GetConfig(h.db)
	uploadDetails := map[string]string{
		"type":     cfg.BackupStorage.Type,
		"filename": backupFile,
	}
	if cfg.BackupStorage.Type == "local" {
		if p, ok := cfg.BackupStorage.Options["path"]; ok {
			uploadDetails["path"] = filepath.Join(p, fmt.Sprintf("%s-%s", req.Name, req.ID), backupFile)
		}
	}
	if cfg.BackupStorage.Type == "s3" {
		if ep, ok := cfg.BackupStorage.Options["endpoint"]; ok {
			uploadDetails["endpoint"] = ep
		}
		if b, ok := cfg.BackupStorage.Options["bucket"]; ok {
			uploadDetails["bucket"] = b
		}
		uploadDetails["key"] = fmt.Sprintf("%s-%s/%s", req.Name, req.ID, backupFile)
	}
	uploadDetailsJSON, _ := json.Marshal(uploadDetails)

	if err := h.db.CompleteBackup(backupID, fileSize, string(uploadDetailsJSON), "completed", ""); err != nil {
		h.logger.Error("failed to update backup record", zap.Error(err))
	}

	req.EventBus.PublishConcurrent(event.ActionBackupUpload, event.BackupUploadRequest{
		ID:         req.ID,
		Name:       req.Name,
		SubvolPath: req.SubvolPath,
		Mountpoint: req.Mountpoint,
		BackupType: "full",
	})

	req.EventBus.PublishConcurrent(event.ActionBackupCleanup, event.BackupCleanupRequest{
		ID:         req.ID,
		Name:       req.Name,
		SubvolPath: req.SubvolPath,
		Mountpoint: req.Mountpoint,
	})

	return map[string]string{"status": "success", "backup_id": backupID}, nil
}

type BackupUploadHandler struct {
	logger *zap.Logger
}

func NewBackupUploadHandler(logger *zap.Logger) *BackupUploadHandler {
	return &BackupUploadHandler{
		logger: logger,
	}
}

func (h *BackupUploadHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.BackupUploadRequest)
	if !ok {
		h.logger.Error("invalid backup upload request type")
		return nil, errInvalidRequest
	}

	h.logger.Info("backup upload",
		zap.String("subvolume_id", req.ID),
		zap.String("name", req.Name),
		zap.String("backup_type", req.BackupType),
		zap.String("mountpoint", req.Mountpoint),
	)

	return map[string]string{"status": "success"}, nil
}

type BackupCleanupHandler struct {
	logger *zap.Logger
}

func NewBackupCleanupHandler(logger *zap.Logger) *BackupCleanupHandler {
	return &BackupCleanupHandler{
		logger: logger,
	}
}

func (h *BackupCleanupHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.BackupCleanupRequest)
	if !ok {
		h.logger.Error("invalid backup cleanup request type")
		return nil, errInvalidRequest
	}

	h.logger.Info("backup cleanup",
		zap.String("subvolume_id", req.ID),
		zap.String("name", req.Name),
		zap.String("mountpoint", req.Mountpoint),
	)

	return map[string]string{"status": "success"}, nil
}

func backupDue(frequency string, schedule event.BackupCheckSchedule, nowHHMM, weekday string, day int) bool {
	switch strings.ToLower(frequency) {
	case "daily":
		return schedule.Time == nowHHMM
	case "weekly":
		return schedule.Time == nowHHMM && schedule.WeeklyDay == weekday
	case "monthly":
		return schedule.Time == nowHHMM && schedule.MonthlyDay == day
	default:
		return false
	}
}
