package handler

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
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
}

func NewBackupFullHandler(logger *zap.Logger) *BackupFullHandler {
	return &BackupFullHandler{
		logger: logger,
	}
}

func (h *BackupFullHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.BackupFullRequest)
	if !ok {
		h.logger.Error("invalid backup full request type")
		return nil, errInvalidRequest
	}

	h.logger.Info("backup full",
		zap.String("subvolume_id", req.ID),
		zap.String("name", req.Name),
		zap.String("mountpoint", req.Mountpoint),
	)

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

	return map[string]string{"status": "success"}, nil
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
