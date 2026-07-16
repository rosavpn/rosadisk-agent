package handler

import (
	"context"

	"go.uber.org/zap"
	"rosadisk-agent/internal/worker/event"
)

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
