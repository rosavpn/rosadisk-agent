package handler

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"rosadisk-agent/internal/storage"
	"rosadisk-agent/internal/worker/event"
)

type MountHandler struct {
	logger *zap.Logger
}

func NewMountHandler(logger *zap.Logger) *MountHandler {
	return &MountHandler{
		logger: logger,
	}
}

func (h *MountHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	h.logger.Info("handling mount list event")

	storageMounts, err := storage.ListMounts()
	if err != nil {
		h.logger.Error("failed to list mounts", zap.Error(err))
		return nil, err
	}

	mounts := make([]event.MountInfo, len(storageMounts))
	for i, m := range storageMounts {
		mounts[i] = event.MountInfo{
			UUID:       m.UUID,
			Label:      m.Label,
			Mountpoint: m.Mountpoint,
			Devices:    m.Devices,
			Used:       m.Used,
		}
	}

	h.logger.Info("mount list completed", zap.Int("count", len(mounts)))

	return mounts, nil
}

type MountCreateHandler struct {
	logger *zap.Logger
}

func NewMountCreateHandler(logger *zap.Logger) *MountCreateHandler {
	return &MountCreateHandler{
		logger: logger,
	}
}

func (h *MountCreateHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	h.logger.Info("handling mount create event")

	req, ok := data.(event.MountRequest)
	if !ok {
		h.logger.Error("invalid request type for mount create")
		return nil, fmt.Errorf("invalid request type")
	}

	mount, err := storage.MountByUUID(req.UUID)
	if err != nil {
		h.logger.Error("failed to mount filesystem", zap.Error(err))
		return nil, err
	}

	result := event.MountInfo{
		UUID:       mount.UUID,
		Label:      mount.Label,
		Mountpoint: mount.Mountpoint,
		Devices:    mount.Devices,
		Used:       mount.Used,
	}

	h.logger.Info("filesystem mounted", zap.String("uuid", mount.UUID), zap.String("mountpoint", mount.Mountpoint))

	return result, nil
}
