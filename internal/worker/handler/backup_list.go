package handler

import (
	"context"
	"time"

	"go.uber.org/zap"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/worker/event"
)

type BackupListHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewBackupListHandler(logger *zap.Logger, db *database.Database) *BackupListHandler {
	return &BackupListHandler{
		logger: logger,
		db:     db,
	}
}

func (h *BackupListHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.BackupListRequest)
	if !ok {
		h.logger.Error("invalid backup list request type")
		return nil, errInvalidRequest
	}

	records, err := h.db.ListBackupsBySubvolume(req.SubvolumeID)
	if err != nil {
		h.logger.Error("failed to list backups", zap.Error(err))
		return nil, err
	}

	backups := make([]event.BackupListResponse, len(records))
	for i, r := range records {
		var completedAt *string
		if r.CompletedAt != nil {
			t := r.CompletedAt.Format(time.RFC3339)
			completedAt = &t
		}
		backups[i] = event.BackupListResponse{
			ID:            r.ID,
			SubvolumeID:   r.SubvolumeID,
			Type:          r.Type,
			ParentID:      r.ParentID,
			SnapshotPath:  r.SnapshotPath,
			Path:          r.Path,
			Size:          r.Size,
			UploadDetails: r.UploadDetails,
			Status:        r.Status,
			Error:         r.Error,
			CreatedAt:     r.CreatedAt.Format(time.RFC3339),
			CompletedAt:   completedAt,
		}
	}

	return backups, nil
}
