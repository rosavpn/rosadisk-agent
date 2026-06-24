package handler

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"rosadisk-agent/internal/storage"
	"rosadisk-agent/internal/worker/event"
)

type FilesystemHandler struct {
	logger *zap.Logger
}

func NewFilesystemHandler(logger *zap.Logger) *FilesystemHandler {
	return &FilesystemHandler{
		logger: logger,
	}
}

func (h *FilesystemHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	h.logger.Info("handling filesystem list event")

	storageFS, err := storage.ListFilesystems()
	if err != nil {
		h.logger.Error("failed to list filesystems", zap.Error(err))
		return nil, err
	}

	filesystems := make([]event.FilesystemInfo, len(storageFS))
	for i, fs := range storageFS {
		filesystems[i] = event.FilesystemInfo{
			UUID:        fs.UUID,
			Label:       fs.Label,
			Size:        fs.Size,
			Devices:     fs.Devices,
			RaidProfile: fs.RaidProfile,
		}
	}

	h.logger.Info("filesystem list completed", zap.Int("count", len(filesystems)))

	return filesystems, nil
}

type FilesystemCreateHandler struct {
	logger *zap.Logger
}

func NewFilesystemCreateHandler(logger *zap.Logger) *FilesystemCreateHandler {
	return &FilesystemCreateHandler{
		logger: logger,
	}
}

func (h *FilesystemCreateHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	h.logger.Info("handling filesystem create event")

	req, ok := data.(event.CreateFilesystemRequest)
	if !ok {
		h.logger.Error("invalid request type for filesystem create")
		return nil, fmt.Errorf("invalid request type")
	}

	fs, err := storage.CreateFilesystem(req.Devices, req.Label, req.RaidProfile)
	if err != nil {
		h.logger.Error("failed to create filesystem", zap.Error(err))
		return nil, err
	}

	result := event.FilesystemInfo{
		UUID:        fs.UUID,
		Label:       fs.Label,
		Size:        fs.Size,
		Devices:     fs.Devices,
		RaidProfile: fs.RaidProfile,
	}

	h.logger.Info("filesystem created", zap.String("uuid", fs.UUID))

	return result, nil
}
