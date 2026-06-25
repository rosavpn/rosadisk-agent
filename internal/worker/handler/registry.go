package handler

import (
	"context"

	"go.uber.org/zap"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/worker/event"
)

type Handler interface {
	Handle(ctx context.Context, data interface{}) (interface{}, error)
}

type HandlerFunc func(ctx context.Context, data interface{}) (interface{}, error)

func (f HandlerFunc) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	return f(ctx, data)
}

func RegisterAll(logger *zap.Logger, db *database.Database) map[event.ActionType]Handler {
	handlers := make(map[event.ActionType]Handler)

	handlers[event.ActionDiskList] = NewDiskHandler(logger)
	handlers[event.ActionFilesystemList] = NewFilesystemHandler(logger)
	handlers[event.ActionFilesystemCreate] = NewFilesystemCreateHandler(logger)
	handlers[event.ActionMountList] = NewMountHandler(logger)
	handlers[event.ActionMountCreate] = NewMountCreateHandler(logger)
	handlers[event.ActionSubvolumeList] = NewSubvolumeHandler(logger, db)
	handlers[event.ActionSubvolumeCreate] = NewSubvolumeCreateHandler(logger, db)
	handlers[event.ActionSubvolumeGet] = NewSubvolumeGetHandler(logger, db)
	handlers[event.ActionSubvolumeDelete] = NewSubvolumeDeleteHandler(logger, db)
	handlers[event.ActionBackup] = NewBackupHandler(logger)
	handlers[event.ActionSnapshotCheck] = NewSnapshotCheckHandler(logger, db)
	handlers[event.ActionDefrag] = NewDefragHandler(logger)
	handlers[event.ActionScrubCheck] = NewScrubCheckHandler(logger)
	handlers[event.ActionBalanceCheck] = NewBalanceCheckHandler(logger)
	handlers[event.ActionScrubDisk] = NewScrubDiskHandler(logger, db)
	handlers[event.ActionBalanceDisk] = NewBalanceDiskHandler(logger, db)
	handlers[event.ActionSnapshotSubvolume] = NewSnapshotSubvolumeHandler(logger, db)
	handlers[event.ActionSnapshotCleanup] = NewSnapshotCleanupHandler(logger, db)
	handlers[event.ActionSnapshotList] = NewSnapshotListHandler(logger, db)

	logger.Info("all handlers registered")
	return handlers
}
