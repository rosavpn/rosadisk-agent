package handler

import (
	"context"
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
	h.logger.Info("handling defrag check event")

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

		logID, err := h.db.InsertJobLog(database.JobLogRecord{
			JobType:     string(event.ActionDefragSubvolume),
			Mountpoint:  mountpoint,
			SubvolumeID: sv.ID,
			TargetName:  sv.Name,
			Status:      "running",
			StartedAt:   time.Now(),
		})
		if err != nil {
			h.logger.Error("failed to insert job log", zap.Error(err))
			continue
		}

		output, err := storage.DefragmentBtrfs(sv.Path)
		status := "success"
		var errMsg string
		if err != nil {
			status = "failed"
			errMsg = err.Error()
			h.logger.Error("defrag failed", zap.String("subvolume_id", sv.ID), zap.Error(err))
		}

		if updateErr := h.db.UpdateJobLog(logID, status, output, errMsg); updateErr != nil {
			h.logger.Error("failed to update job log", zap.Error(updateErr))
		}
		count++
	}

	return map[string]interface{}{"status": "defrag check completed", "count": count}, nil
}
