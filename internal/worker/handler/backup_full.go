package handler

import (
	"context"

	"go.uber.org/zap"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/worker/event"
)

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
