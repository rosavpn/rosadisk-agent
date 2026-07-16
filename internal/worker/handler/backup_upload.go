package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"go.uber.org/zap"
	"rosadisk-agent/internal/config"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/worker/event"
)

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
