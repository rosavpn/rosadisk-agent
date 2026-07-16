package handler

import (
	"context"

	"go.uber.org/zap"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/worker/event"
)

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
