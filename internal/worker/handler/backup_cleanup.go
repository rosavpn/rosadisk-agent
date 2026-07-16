package handler

import (
	"context"

	"go.uber.org/zap"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/storage"
	"rosadisk-agent/internal/worker/event"
)

type BackupCleanupHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewBackupCleanupHandler(logger *zap.Logger, db *database.Database) *BackupCleanupHandler {
	return &BackupCleanupHandler{
		logger: logger,
		db:     db,
	}
}

func (h *BackupCleanupHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.BackupCleanupRequest)
	if !ok {
		h.logger.Error("invalid backup cleanup request type")
		return nil, errInvalidRequest
	}

	h.logger.Info("running backup cleanup", zap.String("subvolume_id", req.ID))

	backups, err := h.db.ListBackupsBySubvolume(req.ID)
	if err != nil {
		h.logger.Error("failed to list backups for cleanup", zap.Error(err))
		return nil, err
	}

	if len(backups) <= 1 {
		return map[string]interface{}{"deleted": 0}, nil
	}

	deleted := 0
	for i := 1; i < len(backups); i++ {
		snapshotPath := backups[i].SnapshotPath
		if err := storage.DeleteSubvolumeBtrfs(snapshotPath); err != nil {
			h.logger.Warn("failed to delete old backup snapshot",
				zap.String("path", snapshotPath),
				zap.Error(err),
			)
			continue
		}
		h.logger.Info("deleted old backup snapshot", zap.String("path", snapshotPath))
		deleted++
	}

	return map[string]interface{}{"deleted": deleted}, nil
}
