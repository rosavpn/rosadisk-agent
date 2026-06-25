package handler

import (
	"context"
	"time"

	"go.uber.org/zap"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/storage"
	"rosadisk-agent/internal/worker/event"
)

type ScrubDiskHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewScrubDiskHandler(logger *zap.Logger, db *database.Database) *ScrubDiskHandler {
	return &ScrubDiskHandler{
		logger: logger,
		db:     db,
	}
}

func (h *ScrubDiskHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.ScrubDiskRequest)
	if !ok {
		h.logger.Error("invalid scrub disk request type")
		return nil, errInvalidRequest
	}

	h.logger.Info("handling per-disk scrub event",
		zap.String("mountpoint", req.Mountpoint),
		zap.String("uuid", req.UUID),
	)

	logID, err := h.db.InsertJobLog(database.JobLogRecord{
		JobType:    "scrub",
		Mountpoint: req.Mountpoint,
		TargetName: req.Label,
		Status:     "running",
		StartedAt:  time.Now(),
	})
	if err != nil {
		h.logger.Error("failed to insert scrub log", zap.Error(err))
		return nil, err
	}

	output, err := storage.StartScrub(req.Mountpoint)
	status := "success"
	errMsg := ""
	if err != nil {
		status = "failed"
		errMsg = err.Error()
		h.logger.Error("scrub failed",
			zap.Error(err),
			zap.String("mountpoint", req.Mountpoint),
		)
	} else {
		h.logger.Info("scrub completed",
			zap.String("mountpoint", req.Mountpoint),
			zap.String("uuid", req.UUID),
		)
	}

	if err := h.db.UpdateJobLog(logID, status, output, errMsg); err != nil {
		h.logger.Error("failed to update scrub log", zap.Error(err))
	}

	return map[string]string{
		"mountpoint": req.Mountpoint,
		"uuid":       req.UUID,
		"status":     status,
	}, nil
}

type BalanceDiskHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewBalanceDiskHandler(logger *zap.Logger, db *database.Database) *BalanceDiskHandler {
	return &BalanceDiskHandler{
		logger: logger,
		db:     db,
	}
}

func (h *BalanceDiskHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.BalanceDiskRequest)
	if !ok {
		h.logger.Error("invalid balance disk request type")
		return nil, errInvalidRequest
	}

	h.logger.Info("handling per-disk balance event",
		zap.String("mountpoint", req.Mountpoint),
		zap.String("uuid", req.UUID),
	)

	logID, err := h.db.InsertJobLog(database.JobLogRecord{
		JobType:    "balance",
		Mountpoint: req.Mountpoint,
		TargetName: req.Label,
		Status:     "running",
		StartedAt:  time.Now(),
	})
	if err != nil {
		h.logger.Error("failed to insert balance log", zap.Error(err))
		return nil, err
	}

	output, err := storage.StartBalance(req.Mountpoint)
	status := "success"
	errMsg := ""
	if err != nil {
		status = "failed"
		errMsg = err.Error()
		h.logger.Error("balance failed",
			zap.Error(err),
			zap.String("mountpoint", req.Mountpoint),
		)
	} else {
		h.logger.Info("balance completed",
			zap.String("mountpoint", req.Mountpoint),
			zap.String("uuid", req.UUID),
		)
	}

	if err := h.db.UpdateJobLog(logID, status, output, errMsg); err != nil {
		h.logger.Error("failed to update balance log", zap.Error(err))
	}

	return map[string]string{
		"mountpoint": req.Mountpoint,
		"uuid":       req.UUID,
		"status":     status,
	}, nil
}
