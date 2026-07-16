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
