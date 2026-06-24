package handler

import (
	"context"

	"go.uber.org/zap"
	"rosadisk-agent/internal/storage"
	"rosadisk-agent/internal/worker/event"
)

type DiskHandler struct {
	logger *zap.Logger
}

func NewDiskHandler(logger *zap.Logger) *DiskHandler {
	return &DiskHandler{
		logger: logger,
	}
}

func (h *DiskHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	h.logger.Info("handling disk list event")

	storageDisks, err := storage.ListDisks()
	if err != nil {
		h.logger.Error("failed to list disks", zap.Error(err))
		return nil, err
	}

	disks := make([]event.DiskInfo, len(storageDisks))
	for i, d := range storageDisks {
		disks[i] = event.DiskInfo{
			Name:   d.Name,
			Size:   d.Size,
			Type:   d.Type,
			Vendor: d.Vendor,
			Model:  d.Model,
			FsType: d.FsType,
		}
	}

	h.logger.Info("disk list completed", zap.Int("count", len(disks)))

	return disks, nil
}
