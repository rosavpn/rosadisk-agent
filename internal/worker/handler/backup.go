package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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
	db     *database.Database
}

func NewBackupIncrementalHandler(logger *zap.Logger, db *database.Database) *BackupIncrementalHandler {
	return &BackupIncrementalHandler{
		logger: logger,
		db:     db,
	}
}

func (h *BackupIncrementalHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.BackupIncrementalRequest)
	if !ok {
		h.logger.Error("invalid backup incremental request type")
		return nil, errInvalidRequest
	}

	parent, err := h.db.GetLatestBackup(req.ID)
	if err != nil {
		h.logger.Error("failed to find parent backup", zap.Error(err))
		return nil, err
	}
	if parent == nil {
		h.logger.Warn("no parent backup found for incremental, skipping",
			zap.String("subvolume_id", req.ID),
		)
		return map[string]string{"status": "skipped", "reason": "no parent backup"}, nil
	}

	return runBackup(h.logger, h.db, req.Mountpoint, req.ID, req.Name, req.SubvolPath,
		"incremental", &parent.ID, parent.SnapshotPath, req.EventBus)
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

	return runBackup(h.logger, h.db, req.Mountpoint, req.ID, req.Name, req.SubvolPath,
		"full", nil, "", req.EventBus)
}

type BackupUploadHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewBackupUploadHandler(logger *zap.Logger, db *database.Database) *BackupUploadHandler {
	return &BackupUploadHandler{
		logger: logger,
		db:     db,
	}
}

func (h *BackupUploadHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.BackupUploadRequest)
	if !ok {
		h.logger.Error("invalid backup upload request type")
		return nil, errInvalidRequest
	}

	h.logger.Info("uploading backup",
		zap.String("backup_id", req.BackupID),
		zap.String("file_path", req.FilePath),
		zap.String("backup_type", req.BackupType),
	)

	cfg, err := config.GetConfig(h.db)
	if err != nil {
		h.logger.Error("failed to read config for upload", zap.Error(err))
		return nil, err
	}

	uploadDetails := map[string]string{
		"type":     cfg.BackupStorage.Type,
		"filename": filepath.Base(req.FilePath),
	}

	switch cfg.BackupStorage.Type {
	case "local":
		if err := h.uploadLocal(req, cfg, uploadDetails); err != nil {
			return nil, err
		}
	case "s3":
		if err := h.uploadS3(req, cfg, uploadDetails); err != nil {
			return nil, err
		}
	default:
		h.logger.Warn("unknown backup storage type, skipping upload",
			zap.String("type", cfg.BackupStorage.Type),
		)
		return map[string]string{"status": "skipped"}, nil
	}

	if err := os.Remove(req.FilePath); err != nil {
		h.logger.Warn("failed to remove local backup file after upload",
			zap.String("path", req.FilePath),
			zap.Error(err),
		)
	}

	uploadDetailsJSON, _ := json.Marshal(uploadDetails)
	if err := h.db.CompleteBackup(req.BackupID, 0, string(uploadDetailsJSON), "completed", ""); err != nil {
		h.logger.Error("failed to update backup upload details", zap.Error(err))
	}

	return map[string]string{"status": "success"}, nil
}

func (h *BackupUploadHandler) uploadLocal(req event.BackupUploadRequest, cfg config.GlobalConfig, details map[string]string) error {
	destPath, ok := cfg.BackupStorage.Options["path"]
	if !ok {
		return fmt.Errorf("local backup path not configured")
	}

	fullDest := filepath.Join(destPath, fmt.Sprintf("%s-%s", req.Name, req.ID), filepath.Base(req.FilePath))
	if err := os.MkdirAll(filepath.Dir(fullDest), 0750); err != nil {
		return fmt.Errorf("failed to create local backup directory: %w", err)
	}

	// #nosec G204 -- paths are from trusted configuration
	cpCmd := exec.Command("cp", req.FilePath, fullDest)
	output, err := cpCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to copy backup to local destination: %w, output: %s", err, string(output))
	}

	details["path"] = fullDest
	h.logger.Info("backup uploaded to local destination", zap.String("path", fullDest))
	return nil
}

func (h *BackupUploadHandler) uploadS3(req event.BackupUploadRequest, cfg config.GlobalConfig, details map[string]string) error {
	endpoint := cfg.BackupStorage.Options["endpoint"]
	bucket := cfg.BackupStorage.Options["bucket"]
	region := cfg.BackupStorage.Options["region"]
	key := fmt.Sprintf("%s-%s/%s", req.Name, req.ID, filepath.Base(req.FilePath))

	accessKey, err := config.ReadS3AccessKey()
	if err != nil {
		return fmt.Errorf("failed to read s3 access key: %w", err)
	}
	secretKey, err := config.ReadS3SecretKey()
	if err != nil {
		return fmt.Errorf("failed to read s3 secret key: %w", err)
	}

	sess, err := session.NewSession(&aws.Config{
		Endpoint:         aws.String(endpoint),
		Region:           aws.String(region),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create s3 session: %w", err)
	}

	file, err := os.Open(req.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open backup file for s3 upload: %w", err)
	}
	defer func() { _ = file.Close() }()

	uploader := s3manager.NewUploader(sess)
	_, err = uploader.UploadWithContext(context.Background(), &s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to s3: %w", err)
	}

	details["endpoint"] = endpoint
	details["bucket"] = bucket
	details["key"] = key
	h.logger.Info("backup uploaded to s3",
		zap.String("bucket", bucket),
		zap.String("key", key),
	)
	return nil
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
