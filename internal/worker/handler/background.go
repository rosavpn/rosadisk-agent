package handler

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"rosadisk-agent/internal/storage"
	"rosadisk-agent/internal/worker/event"
)

var errInvalidRequest = errors.New("invalid request type")

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

type ScrubCheckHandler struct {
	logger *zap.Logger
}

func NewScrubCheckHandler(logger *zap.Logger) *ScrubCheckHandler {
	return &ScrubCheckHandler{
		logger: logger,
	}
}

func (h *ScrubCheckHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.ScrubCheckRequest)
	if !ok {
		h.logger.Error("invalid scrub check request type")
		return nil, errInvalidRequest
	}

	h.logger.Info("handling scrub check event")

	mounts, err := storage.ListMounts()
	if err != nil {
		h.logger.Error("failed to list mounts for scrub", zap.Error(err))
		return nil, err
	}

	for _, m := range mounts {
		req.EventBus.PublishAsync(event.ActionScrubDisk, event.ScrubDiskRequest{
			Mountpoint: m.Mountpoint,
			UUID:       m.UUID,
			Label:      m.Label,
		})
	}

	return map[string]string{"status": "scrub jobs dispatched"}, nil
}

type BalanceCheckHandler struct {
	logger *zap.Logger
}

func NewBalanceCheckHandler(logger *zap.Logger) *BalanceCheckHandler {
	return &BalanceCheckHandler{
		logger: logger,
	}
}

func (h *BalanceCheckHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.BalanceCheckRequest)
	if !ok {
		h.logger.Error("invalid balance check request type")
		return nil, errInvalidRequest
	}

	h.logger.Info("handling balance check event")

	mounts, err := storage.ListMounts()
	if err != nil {
		h.logger.Error("failed to list mounts for balance", zap.Error(err))
		return nil, err
	}

	for _, m := range mounts {
		req.EventBus.PublishAsync(event.ActionBalanceDisk, event.BalanceDiskRequest{
			Mountpoint: m.Mountpoint,
			UUID:       m.UUID,
			Label:      m.Label,
		})
	}

	return map[string]string{"status": "balance jobs dispatched"}, nil
}
