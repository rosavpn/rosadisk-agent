package handler

import (
	"context"
	"time"

	"go.uber.org/zap"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/storage"
)

type BackupHandler struct {
	logger *zap.Logger
}

func NewBackupHandler(logger *zap.Logger) *BackupHandler {
	return &BackupHandler{
		logger: logger,
	}
}

func (h *BackupHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	h.logger.Info("handling backup event")
	return map[string]string{"status": "backup completed (dummy)"}, nil
}

type SnapshotHandler struct {
	logger *zap.Logger
}

func NewSnapshotHandler(logger *zap.Logger) *SnapshotHandler {
	return &SnapshotHandler{
		logger: logger,
	}
}

func (h *SnapshotHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	h.logger.Info("handling snapshot event")
	return map[string]string{"status": "snapshot completed (dummy)"}, nil
}

type DefragHandler struct {
	logger *zap.Logger
}

func NewDefragHandler(logger *zap.Logger) *DefragHandler {
	return &DefragHandler{
		logger: logger,
	}
}

func (h *DefragHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	h.logger.Info("handling defrag event")
	return map[string]string{"status": "defrag completed (dummy)"}, nil
}

type ScrubHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewScrubHandler(logger *zap.Logger, db *database.Database) *ScrubHandler {
	return &ScrubHandler{
		logger: logger,
		db:     db,
	}
}

func (h *ScrubHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	h.logger.Info("handling scrub event")

	mounts, err := storage.ListMounts()
	if err != nil {
		h.logger.Error("failed to list mounts for scrub", zap.Error(err))
		return nil, err
	}

	results := make([]map[string]string, 0)

	for _, m := range mounts {
		logID, err := h.db.InsertJobLog(database.JobLogRecord{
			JobType:    "scrub",
			Mountpoint: m.Mountpoint,
			TargetName: m.Label,
			Status:     "running",
			StartedAt:  time.Now(),
		})
		if err != nil {
			h.logger.Error("failed to insert scrub log", zap.Error(err))
			continue
		}

		output, err := storage.StartScrub(m.Mountpoint)
		status := "success"
		errMsg := ""
		if err != nil {
			status = "failed"
			errMsg = err.Error()
			h.logger.Error("scrub failed",
				zap.Error(err),
				zap.String("mountpoint", m.Mountpoint),
			)
		} else {
			h.logger.Info("scrub completed",
				zap.String("mountpoint", m.Mountpoint),
				zap.String("uuid", m.UUID),
			)
		}

		if err := h.db.UpdateJobLog(logID, status, output, errMsg); err != nil {
			h.logger.Error("failed to update scrub log", zap.Error(err))
		}

		results = append(results, map[string]string{
			"mountpoint": m.Mountpoint,
			"uuid":       m.UUID,
			"status":     status,
		})
	}

	return results, nil
}

type BalanceHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewBalanceHandler(logger *zap.Logger, db *database.Database) *BalanceHandler {
	return &BalanceHandler{
		logger: logger,
		db:     db,
	}
}

func (h *BalanceHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	h.logger.Info("handling balance event")

	mounts, err := storage.ListMounts()
	if err != nil {
		h.logger.Error("failed to list mounts for balance", zap.Error(err))
		return nil, err
	}

	results := make([]map[string]string, 0)

	for _, m := range mounts {
		logID, err := h.db.InsertJobLog(database.JobLogRecord{
			JobType:    "balance",
			Mountpoint: m.Mountpoint,
			TargetName: m.Label,
			Status:     "running",
			StartedAt:  time.Now(),
		})
		if err != nil {
			h.logger.Error("failed to insert balance log", zap.Error(err))
			continue
		}

		output, err := storage.StartBalance(m.Mountpoint)
		status := "success"
		errMsg := ""
		if err != nil {
			status = "failed"
			errMsg = err.Error()
			h.logger.Error("balance failed",
				zap.Error(err),
				zap.String("mountpoint", m.Mountpoint),
			)
		} else {
			h.logger.Info("balance completed",
				zap.String("mountpoint", m.Mountpoint),
				zap.String("uuid", m.UUID),
			)
		}

		if err := h.db.UpdateJobLog(logID, status, output, errMsg); err != nil {
			h.logger.Error("failed to update balance log", zap.Error(err))
		}

		results = append(results, map[string]string{
			"mountpoint": m.Mountpoint,
			"uuid":       m.UUID,
			"status":     status,
		})
	}

	return results, nil
}
