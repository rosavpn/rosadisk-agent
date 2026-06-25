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

type DefragCheckHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewDefragCheckHandler(logger *zap.Logger, db *database.Database) *DefragCheckHandler {
	return &DefragCheckHandler{
		logger: logger,
		db:     db,
	}
}

func (h *DefragCheckHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.DefragCheckRequest)
	if !ok {
		h.logger.Error("invalid defrag check request type")
		return nil, errInvalidRequest
	}

	h.logger.Info("handling defrag check event")

	now := time.Now()
	nowHHMM := now.Format("15:04")
	weekday := strings.ToLower(now.Weekday().String())
	day := now.Day()

	subvolumes, err := h.db.ListSubvolumes()
	if err != nil {
		h.logger.Error("failed to list subvolumes for defrag", zap.Error(err))
		return nil, err
	}

	count := 0
	for _, sv := range subvolumes {
		if !sv.Defrag {
			continue
		}

		if !defragDue(sv.DefragFrequency, req.Schedule, nowHHMM, weekday, day) {
			continue
		}

		if sv.BackupIncrementalEnabled || sv.BackupFullEnabled {
			h.logger.Debug("skipping defrag for subvolume with backup enabled",
				zap.String("subvolume_id", sv.ID),
				zap.String("name", sv.Name),
			)
			continue
		}

		mountpoint, err := storage.FindMountpointByUUID(sv.FsUUID)
		if err != nil {
			h.logger.Warn("filesystem not mounted for defrag subvolume",
				zap.String("subvolume_id", sv.ID),
				zap.String("fs_uuid", sv.FsUUID),
				zap.Error(err),
			)
			continue
		}

		h.logger.Info("dispatching defrag for subvolume",
			zap.String("subvolume_id", sv.ID),
			zap.String("name", sv.Name),
		)

		req.EventBus.PublishAsync(event.ActionDefragSubvolume, event.DefragSubvolumeRequest{
			ID:         sv.ID,
			Name:       sv.Name,
			FsUUID:     sv.FsUUID,
			SubvolPath: sv.Path,
			Mountpoint: mountpoint,
		})
		count++
	}

	return map[string]interface{}{"status": "defrag jobs dispatched", "count": count}, nil
}

type DefragSubvolumeHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewDefragSubvolumeHandler(logger *zap.Logger, db *database.Database) *DefragSubvolumeHandler {
	return &DefragSubvolumeHandler{
		logger: logger,
		db:     db,
	}
}

func (h *DefragSubvolumeHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.DefragSubvolumeRequest)
	if !ok {
		h.logger.Error("invalid defrag subvolume request type")
		return nil, errInvalidRequest
	}

	h.logger.Info("running defrag on subvolume",
		zap.String("subvolume_id", req.ID),
		zap.String("name", req.Name),
	)

	logID, err := h.db.InsertJobLog(database.JobLogRecord{
		JobType:     string(event.ActionDefragSubvolume),
		Mountpoint:  req.Mountpoint,
		SubvolumeID: req.ID,
		TargetName:  req.Name,
		Status:      "running",
		StartedAt:   time.Now(),
	})
	if err != nil {
		h.logger.Error("failed to insert job log", zap.Error(err))
		return nil, err
	}

	output, err := storage.DefragmentBtrfs(req.SubvolPath)
	status := "success"
	var errMsg string
	if err != nil {
		status = "failed"
		errMsg = err.Error()
		h.logger.Error("defrag failed", zap.String("subvolume_id", req.ID), zap.Error(err))
	}

	if updateErr := h.db.UpdateJobLog(logID, status, output, errMsg); updateErr != nil {
		h.logger.Error("failed to update job log", zap.Error(updateErr))
	}

	return map[string]string{"status": status, "output": output}, nil
}

func defragDue(frequency string, schedule event.DefragSchedule, nowHHMM, weekday string, day int) bool {
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
