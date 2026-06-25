package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/storage"
	"rosadisk-agent/internal/worker/event"
)

type SnapshotCheckHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewSnapshotCheckHandler(logger *zap.Logger, db *database.Database) *SnapshotCheckHandler {
	return &SnapshotCheckHandler{
		logger: logger,
		db:     db,
	}
}

func (h *SnapshotCheckHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.SnapshotCheckRequest)
	if !ok {
		h.logger.Error("invalid snapshot check request type")
		return nil, errInvalidRequest
	}

	h.logger.Info("handling snapshot check event")

	subvolumes, err := h.db.ListSubvolumes()
	if err != nil {
		h.logger.Error("failed to list subvolumes for snapshot", zap.Error(err))
		return nil, err
	}

	now := time.Now()
	minute := now.Minute()
	timeHHMM := now.Format("15:04")
	weekday := strings.ToLower(now.Weekday().String())
	day := now.Day()

	count := 0
	for _, sv := range subvolumes {
		if !sv.SnapshotEnabled {
			continue
		}

		if !snapshotDue(sv.SnapshotFrequency, req.Snapshot, minute, timeHHMM, weekday, day) {
			continue
		}

		mountpoint, err := storage.FindMountpointByUUID(sv.FsUUID)
		if err != nil {
			h.logger.Warn("filesystem not mounted for snapshot subvolume",
				zap.String("subvolume_id", sv.ID),
				zap.String("fs_uuid", sv.FsUUID),
				zap.Error(err),
			)
			continue
		}

		req.EventBus.PublishConcurrent(event.ActionSnapshotSubvolume, event.SnapshotSubvolumeRequest{
			ID:         sv.ID,
			Name:       sv.Name,
			FsUUID:     sv.FsUUID,
			SubvolPath: sv.Path,
			Mountpoint: mountpoint,
			Frequency:  sv.SnapshotFrequency,
			Retention:  sv.SnapshotRetention,
			EventBus:   req.EventBus,
		})
		count++
	}

	return map[string]interface{}{"status": "snapshot jobs dispatched", "count": count}, nil
}

func snapshotDue(frequency string, schedule event.SnapshotSchedule, minute int, timeHHMM, weekday string, day int) bool {
	switch strings.ToLower(frequency) {
	case "hourly":
		return schedule.HourlyMinute == minute
	case "daily":
		return schedule.Time == timeHHMM
	case "weekly":
		return schedule.Time == timeHHMM && schedule.WeeklyDay == weekday
	case "monthly":
		return schedule.Time == timeHHMM && schedule.MonthlyDay == day
	default:
		return false
	}
}

type SnapshotSubvolumeHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewSnapshotSubvolumeHandler(logger *zap.Logger, db *database.Database) *SnapshotSubvolumeHandler {
	return &SnapshotSubvolumeHandler{
		logger: logger,
		db:     db,
	}
}

func (h *SnapshotSubvolumeHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.SnapshotSubvolumeRequest)
	if !ok {
		h.logger.Error("invalid snapshot subvolume request type")
		return nil, errInvalidRequest
	}

	now := time.Now()
	snapshotName := fmt.Sprintf("snapshot-%s-%s-%s",
		req.Frequency,
		now.Format("02012006"),
		now.Format("1504"),
	)
	snapshotDir := filepath.Join(req.Mountpoint, ".rosadisk", "snapshots", fmt.Sprintf("%s-%s", req.Name, req.ID))
	snapshotPath := filepath.Join(snapshotDir, snapshotName)

	h.logger.Info("creating snapshot",
		zap.String("subvolume_id", req.ID),
		zap.String("name", req.Name),
		zap.String("snapshot_path", snapshotPath),
	)

	startedAt := time.Now()
	logID, err := h.db.InsertJobLog(database.JobLogRecord{
		JobType:     "snapshot",
		Mountpoint:  req.Mountpoint,
		SubvolumeID: req.ID,
		TargetName:  snapshotName,
		Status:      "running",
		StartedAt:   startedAt,
	})
	if err != nil {
		h.logger.Error("failed to insert job log", zap.Error(err))
		return nil, err
	}

	if err := os.MkdirAll(snapshotDir, 0750); err != nil {
		status := "failed"
		errMsg := err.Error()
		if updateErr := h.db.UpdateJobLog(logID, status, "", errMsg); updateErr != nil {
			h.logger.Error("failed to update job log", zap.Error(updateErr))
		}
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	if err := storage.CreateSnapshotBtrfs(req.SubvolPath, snapshotPath); err != nil {
		status := "failed"
		errMsg := err.Error()
		if updateErr := h.db.UpdateJobLog(logID, status, "", errMsg); updateErr != nil {
			h.logger.Error("failed to update job log", zap.Error(updateErr))
		}
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	snapshotID := uuid.New().String()
	if err := h.db.InsertSnapshot(database.SnapshotRecord{
		ID:          snapshotID,
		SubvolumeID: req.ID,
		Name:        snapshotName,
		Path:        snapshotPath,
		Frequency:   req.Frequency,
		Size:        0,
		CreatedAt:   now,
	}); err != nil {
		h.logger.Error("failed to insert snapshot record", zap.Error(err))
		return nil, err
	}

	if err := h.db.UpdateJobLog(logID, "success", snapshotPath, ""); err != nil {
		h.logger.Error("failed to update job log", zap.Error(err))
	}

	req.EventBus.PublishConcurrent(event.ActionSnapshotCleanup, event.SnapshotCleanupRequest{
		ID:         req.ID,
		Name:       req.Name,
		FsUUID:     req.FsUUID,
		SubvolPath: req.SubvolPath,
		Mountpoint: req.Mountpoint,
		Retention:  req.Retention,
	})

	return map[string]string{
		"subvolume_id": req.ID,
		"snapshot":     snapshotName,
		"status":       "success",
	}, nil
}

type SnapshotCleanupHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewSnapshotCleanupHandler(logger *zap.Logger, db *database.Database) *SnapshotCleanupHandler {
	return &SnapshotCleanupHandler{
		logger: logger,
		db:     db,
	}
}

func (h *SnapshotCleanupHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.SnapshotCleanupRequest)
	if !ok {
		h.logger.Error("invalid snapshot cleanup request type")
		return nil, errInvalidRequest
	}

	h.logger.Info("running snapshot cleanup",
		zap.String("subvolume_id", req.ID),
		zap.Int("retention", req.Retention),
	)

	snapshots, err := h.db.ListSnapshotsBySubvolume(req.ID)
	if err != nil {
		h.logger.Error("failed to list snapshots for cleanup", zap.Error(err))
		return nil, err
	}

	if len(snapshots) <= req.Retention {
		h.logger.Info("within retention limit, nothing to clean up",
			zap.Int("total", len(snapshots)),
			zap.Int("retention", req.Retention),
		)
		return map[string]interface{}{"deleted": 0}, nil
	}

	deleted := 0
	for i := req.Retention; i < len(snapshots); i++ {
		snapshot := snapshots[i]

		if err := storage.DeleteSubvolumeBtrfs(snapshot.Path); err != nil {
			h.logger.Error("failed to delete snapshot",
				zap.String("path", snapshot.Path),
				zap.Error(err),
			)
			continue
		}

		if err := h.db.DeleteSnapshot(snapshot.ID); err != nil {
			h.logger.Error("failed to delete snapshot record",
				zap.String("id", snapshot.ID),
				zap.Error(err),
			)
			continue
		}

		h.logger.Info("deleted snapshot",
			zap.String("id", snapshot.ID),
			zap.String("path", snapshot.Path),
		)
		deleted++
	}

	return map[string]interface{}{"deleted": deleted}, nil
}

type SnapshotListHandler struct {
	logger *zap.Logger
	db     *database.Database
}

func NewSnapshotListHandler(logger *zap.Logger, db *database.Database) *SnapshotListHandler {
	return &SnapshotListHandler{
		logger: logger,
		db:     db,
	}
}

func (h *SnapshotListHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	req, ok := data.(event.SnapshotListRequest)
	if !ok {
		h.logger.Error("invalid snapshot list request type")
		return nil, errInvalidRequest
	}

	records, err := h.db.ListSnapshotsBySubvolume(req.SubvolumeID)
	if err != nil {
		h.logger.Error("failed to list snapshots", zap.Error(err))
		return nil, err
	}

	snapshots := make([]event.SnapshotInfo, len(records))
	for i, r := range records {
		snapshots[i] = event.SnapshotInfo{
			ID:          r.ID,
			SubvolumeID: r.SubvolumeID,
			Name:        r.Name,
			Path:        r.Path,
			Frequency:   r.Frequency,
			Size:        r.Size,
			CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		}
	}

	return snapshots, nil
}
