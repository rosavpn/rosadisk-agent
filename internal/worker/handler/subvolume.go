package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/storage"
	"rosadisk-agent/internal/worker/event"
)

type SubvolumeHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewSubvolumeHandler(logger *zap.Logger, db *database.Database) *SubvolumeHandler {
	return &SubvolumeHandler{
		logger: logger,
		db:     db,
	}
}

func (h *SubvolumeHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	h.logger.Info("handling subvolume list event")

	dbRecords, err := h.db.ListSubvolumes()
	if err != nil {
		h.logger.Error("failed to list subvolumes", zap.Error(err))
		return nil, err
	}

	subvolumes := make([]event.SubvolumeInfo, len(dbRecords))
	for i, r := range dbRecords {
		subvolumes[i] = event.SubvolumeInfo{
			ID:          r.ID,
			Name:        r.Name,
			FsUUID:      r.FsUUID,
			Path:        r.Path,
			Compression: r.Compression,
			Quota: event.QuotaConfig{
				Enabled: r.QuotaEnabled,
				Limit:   r.QuotaLimit,
			},
			Snapshots: event.SnapshotConfig{
				Enabled:   r.SnapshotEnabled,
				Frequency: r.SnapshotFrequency,
				Retention: r.SnapshotRetention,
			},
			Backups: event.BackupConfig{
				Incremental: toBackupSchedule(r.BackupIncrementalEnabled, r.BackupIncrementalFrequency),
				Full:        toBackupSchedule(r.BackupFullEnabled, r.BackupFullFrequency),
			},
			Defrag:    r.Defrag,
			NFS:       r.NFS,
			SMB:       r.SMB,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
		}
	}

	h.logger.Info("subvolume list completed", zap.Int("count", len(subvolumes)))

	return subvolumes, nil
}

type SubvolumeCreateHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewSubvolumeCreateHandler(logger *zap.Logger, db *database.Database) *SubvolumeCreateHandler {
	return &SubvolumeCreateHandler{
		logger: logger,
		db:     db,
	}
}

func (h *SubvolumeCreateHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	h.logger.Info("handling subvolume create event")

	req, ok := data.(event.CreateSubvolumeRequest)
	if !ok {
		h.logger.Error("invalid request type for subvolume create")
		return nil, fmt.Errorf("invalid request type")
	}

	mountpoint, err := storage.FindMountpointByUUID(req.FsUUID)
	if err != nil {
		h.logger.Error("filesystem not mounted", zap.Error(err))
		return nil, err
	}

	subvolPath, err := storage.CreateSubvolumeBtrfs(storage.CreateSubvolumeBtrfsRequest{
		Mountpoint:  mountpoint,
		Name:        req.Name,
		Compression: req.Compression,
		QuotaLimit:  quotaEnabledLimit(req.Quota.Enabled, req.Quota.Limit),
	})
	if err != nil {
		h.logger.Error("failed to create btrfs subvolume", zap.Error(err))
		return nil, err
	}

	id := uuid.New().String()

	dbRecord := database.CreateSubvolumeRecord{
		ID:                         id,
		Name:                       req.Name,
		FsUUID:                     req.FsUUID,
		Path:                       subvolPath,
		Compression:                req.Compression,
		QuotaEnabled:               req.Quota.Enabled,
		QuotaLimit:                 req.Quota.Limit,
		SnapshotEnabled:            req.Snapshots.Enabled,
		SnapshotFrequency:          req.Snapshots.Frequency,
		SnapshotRetention:          req.Snapshots.Retention,
		BackupIncrementalEnabled:   req.Backups.Incremental.Enabled,
		BackupIncrementalFrequency: req.Backups.Incremental.Frequency,
		BackupFullEnabled:          req.Backups.Full.Enabled,
		BackupFullFrequency:        req.Backups.Full.Frequency,
		Defrag:                     req.Defrag,
		NFS:                        req.NFS,
		SMB:                        req.SMB,
	}

	if err := h.db.InsertSubvolumeRecord(dbRecord); err != nil {
		_ = storage.DeleteSubvolumeBtrfs(subvolPath)
		h.logger.Error("failed to persist subvolume", zap.Error(err))
		return nil, err
	}

	result := event.SubvolumeInfo{
		ID:          id,
		Name:        req.Name,
		FsUUID:      req.FsUUID,
		Path:        subvolPath,
		Compression: req.Compression,
		Quota:       req.Quota,
		Snapshots:   req.Snapshots,
		Backups:     req.Backups,
		Defrag:      req.Defrag,
		NFS:         req.NFS,
		SMB:         req.SMB,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	h.logger.Info("subvolume created", zap.String("id", id), zap.String("path", subvolPath))

	return result, nil
}

type SubvolumeGetHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewSubvolumeGetHandler(logger *zap.Logger, db *database.Database) *SubvolumeGetHandler {
	return &SubvolumeGetHandler{
		logger: logger,
		db:     db,
	}
}

func (h *SubvolumeGetHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	h.logger.Info("handling subvolume get event")

	req, ok := data.(event.SubvolumeGetRequest)
	if !ok {
		h.logger.Error("invalid request type for subvolume get")
		return nil, fmt.Errorf("invalid request type")
	}

	r, err := h.db.GetSubvolume(req.ID)
	if err != nil {
		h.logger.Error("failed to get subvolume", zap.Error(err))
		return nil, err
	}

	result := event.SubvolumeInfo{
		ID:          r.ID,
		Name:        r.Name,
		FsUUID:      r.FsUUID,
		Path:        r.Path,
		Compression: r.Compression,
		Quota: event.QuotaConfig{
			Enabled: r.QuotaEnabled,
			Limit:   r.QuotaLimit,
		},
		Snapshots: event.SnapshotConfig{
			Enabled:   r.SnapshotEnabled,
			Frequency: r.SnapshotFrequency,
			Retention: r.SnapshotRetention,
		},
		Backups: event.BackupConfig{
			Incremental: toBackupSchedule(r.BackupIncrementalEnabled, r.BackupIncrementalFrequency),
			Full:        toBackupSchedule(r.BackupFullEnabled, r.BackupFullFrequency),
		},
		Defrag:    r.Defrag,
		NFS:       r.NFS,
		SMB:       r.SMB,
		CreatedAt: r.CreatedAt.Format(time.RFC3339),
	}

	h.logger.Info("subvolume retrieved", zap.String("id", r.ID))

	return result, nil
}

type SubvolumeDeleteHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewSubvolumeDeleteHandler(logger *zap.Logger, db *database.Database) *SubvolumeDeleteHandler {
	return &SubvolumeDeleteHandler{
		logger: logger,
		db:     db,
	}
}

func (h *SubvolumeDeleteHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	h.logger.Info("handling subvolume delete event")

	req, ok := data.(event.SubvolumeDeleteRequest)
	if !ok {
		h.logger.Error("invalid request type for subvolume delete")
		return nil, fmt.Errorf("invalid request type")
	}

	r, err := h.db.GetSubvolume(req.ID)
	if err != nil {
		return nil, err
	}

	if err := storage.DeleteSubvolumeBtrfs(r.Path); err != nil {
		h.logger.Error("failed to delete btrfs subvolume", zap.Error(err))
		return nil, err
	}

	if err := h.db.DeleteSubvolumeRecord(req.ID); err != nil {
		h.logger.Error("failed to remove subvolume from database", zap.Error(err))
		return nil, err
	}

	h.logger.Info("subvolume deleted", zap.String("id", req.ID))

	return nil, nil
}

func quotaEnabledLimit(enabled bool, limit int64) int64 {
	if !enabled {
		return 0
	}
	return limit
}

func toBackupSchedule(enabled bool, freq string) event.BackupSchedule {
	return event.BackupSchedule{
		Enabled:   enabled,
		Frequency: freq,
	}
}
