package server

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"rosadisk-agent/internal/config"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/storage"
)

const maxSnapshotWorkers = 5

func (s *Server) handleBackup(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling backup event")
	return map[string]string{"status": "backup completed (dummy)"}, nil
}

func (s *Server) handleSnapshot(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling snapshot event")

	cfg, err := config.GetConfig(s.DB)
	if err != nil {
		s.logger.Error("failed to read config for snapshot", zap.Error(err))
		return nil, err
	}

	allSubvols, err := database.ListSubvolumes(s.DB)
	if err != nil {
		s.logger.Error("failed to list subvolumes for snapshot", zap.Error(err))
		return nil, err
	}

	now := time.Now()
	matching := make([]database.SubvolumeRecord, 0)

	for _, sv := range allSubvols {
		if !sv.SnapshotEnabled {
			continue
		}
		if snapshotShouldRun(sv, cfg, now) {
			matching = append(matching, sv)
		}
	}

	if len(matching) == 0 {
		s.logger.Info("no subvolumes match snapshot schedule")
		return []map[string]string{}, nil
	}

	s.logger.Info("snapshot candidates found",
		zap.Int("count", len(matching)),
	)

	sem := make(chan struct{}, maxSnapshotWorkers)
	var wg sync.WaitGroup
	results := make([]map[string]string, len(matching))
	var mu sync.Mutex

	for i, sv := range matching {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, subvol database.SubvolumeRecord) {
			defer wg.Done()
			defer func() { <-sem }()

			result := s.runSnapshotJob(subvol)
			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, sv)
	}

	wg.Wait()

	s.logger.Info("snapshot batch completed",
		zap.Int("total", len(matching)),
	)

	return results, nil
}

func (s *Server) runSnapshotJob(subvol database.SubvolumeRecord) map[string]string {
	s.dbMu.Lock()
	logID, err := database.InsertJobLog(s.DB, database.JobLogRecord{
		JobType:     "snapshot",
		SubvolumeID: subvol.ID,
		TargetName:  subvol.Name,
		Status:      "running",
		StartedAt:   time.Now(),
	})
	s.dbMu.Unlock()
	if err != nil {
		s.logger.Error("failed to insert snapshot log",
			zap.Error(err),
			zap.String("subvolume", subvol.ID),
		)
		return map[string]string{
			"subvolume_id": subvol.ID,
			"name":         subvol.Name,
			"status":       "failed",
			"error":        "failed to insert log",
		}
	}

	mountpoint, err := storage.FindMountpointByUUID(subvol.FsUUID)
	if err != nil {
		s.logger.Error("filesystem not mounted for snapshot",
			zap.Error(err),
			zap.String("subvolume", subvol.ID),
		)
		s.dbMu.Lock()
		failJobLog(s.DB, logID, err.Error())
		s.dbMu.Unlock()
		return map[string]string{
			"subvolume_id": subvol.ID,
			"name":         subvol.Name,
			"status":       "failed",
			"error":        err.Error(),
		}
	}

	snapshotPath, err := storage.CreateSnapshotBtrfs(mountpoint, subvol.Path, subvol.Name, subvol.ID, subvol.SnapshotFrequency)
	if err != nil {
		s.logger.Error("failed to create snapshot",
			zap.Error(err),
			zap.String("subvolume", subvol.ID),
		)
		s.dbMu.Lock()
		failJobLog(s.DB, logID, err.Error())
		s.dbMu.Unlock()
		return map[string]string{
			"subvolume_id": subvol.ID,
			"name":         subvol.Name,
			"status":       "failed",
			"error":        err.Error(),
		}
	}

	snapshotID := uuid.New().String()
	snapshotName := snapshotPath
	if idx := strings.LastIndex(snapshotPath, "/"); idx >= 0 {
		snapshotName = snapshotPath[idx+1:]
	}

	s.dbMu.Lock()
	err = database.InsertSnapshot(s.DB, database.SnapshotRecord{
		ID:          snapshotID,
		SubvolumeID: subvol.ID,
		Name:        snapshotName,
		Path:        snapshotPath,
		Frequency:   subvol.SnapshotFrequency,
		CreatedAt:   time.Now(),
	})
	s.dbMu.Unlock()
	if err != nil {
		s.logger.Error("failed to persist snapshot record",
			zap.Error(err),
			zap.String("subvolume", subvol.ID),
		)
		_ = storage.DeleteSnapshotBtrfs(snapshotPath)
		s.dbMu.Lock()
		failJobLog(s.DB, logID, err.Error())
		s.dbMu.Unlock()
		return map[string]string{
			"subvolume_id": subvol.ID,
			"name":         subvol.Name,
			"status":       "failed",
			"error":        err.Error(),
		}
	}

	s.dbMu.Lock()
	successJobLog(s.DB, logID, fmt.Sprintf("snapshot created: %s", snapshotPath))
	enforceSnapshotRetention(s.DB, subvol)
	s.dbMu.Unlock()

	s.logger.Info("snapshot created",
		zap.String("subvolume", subvol.ID),
		zap.String("path", snapshotPath),
	)

	return map[string]string{
		"subvolume_id": subvol.ID,
		"name":         subvol.Name,
		"path":         snapshotPath,
		"status":       "success",
	}
}

func failJobLog(db *sql.DB, logID int64, errMsg string) {
	_ = database.UpdateJobLog(db, logID, "failed", "", errMsg)
}

func successJobLog(db *sql.DB, logID int64, output string) {
	_ = database.UpdateJobLog(db, logID, "success", output, "")
}

func enforceSnapshotRetention(db *sql.DB, subvol database.SubvolumeRecord) {
	if subvol.SnapshotRetention <= 0 {
		return
	}

	snapshots, err := database.ListSnapshotsBySubvolume(db, subvol.ID)
	if err != nil {
		return
	}

	var freqSnapshots []database.SnapshotRecord
	for _, s := range snapshots {
		if s.Frequency == subvol.SnapshotFrequency {
			freqSnapshots = append(freqSnapshots, s)
		}
	}

	if len(freqSnapshots) <= subvol.SnapshotRetention {
		return
	}

	toDelete := len(freqSnapshots) - subvol.SnapshotRetention
	for i := 0; i < toDelete; i++ {
		_ = storage.DeleteSnapshotBtrfs(freqSnapshots[i].Path)
		_ = database.DeleteSnapshotRecord(db, freqSnapshots[i].ID)
	}
}

func snapshotShouldRun(subvol database.SubvolumeRecord, cfg config.GlobalConfig, now time.Time) bool {
	minute := now.Minute()
	timeHHMM := now.Format("15:04")
	weekday := strings.ToLower(now.Weekday().String())
	day := now.Day()

	switch subvol.SnapshotFrequency {
	case "hourly":
		return cfg.Snapshot.HourlyMinute == minute
	case "daily":
		return cfg.Snapshot.Time == timeHHMM
	case "weekly":
		return cfg.Snapshot.Time == timeHHMM && cfg.Snapshot.WeeklyDay == weekday
	case "monthly":
		return cfg.Snapshot.Time == timeHHMM && cfg.Snapshot.MonthlyDay == day
	}
	return false
}

func (s *Server) handleDefrag(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling defrag event")
	return map[string]string{"status": "defrag completed (dummy)"}, nil
}

func (s *Server) handleScrub(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling scrub event")

	mounts, err := storage.ListMounts()
	if err != nil {
		s.logger.Error("failed to list mounts for scrub", zap.Error(err))
		return nil, err
	}

	results := make([]map[string]string, 0)

	for _, m := range mounts {
		logID, err := database.InsertJobLog(s.DB, database.JobLogRecord{
			JobType:    "scrub",
			Mountpoint: m.Mountpoint,
			TargetName: m.Label,
			Status:     "running",
			StartedAt:  time.Now(),
		})
		if err != nil {
			s.logger.Error("failed to insert scrub log", zap.Error(err))
			continue
		}

		output, err := storage.StartScrub(m.Mountpoint)
		status := "success"
		errMsg := ""
		if err != nil {
			status = "failed"
			errMsg = err.Error()
			s.logger.Error("scrub failed",
				zap.Error(err),
				zap.String("mountpoint", m.Mountpoint),
			)
		} else {
			s.logger.Info("scrub completed",
				zap.String("mountpoint", m.Mountpoint),
				zap.String("uuid", m.UUID),
			)
		}

		if err := database.UpdateJobLog(s.DB, logID, status, output, errMsg); err != nil {
			s.logger.Error("failed to update scrub log", zap.Error(err))
		}

		results = append(results, map[string]string{
			"mountpoint": m.Mountpoint,
			"uuid":       m.UUID,
			"status":     status,
		})
	}

	return results, nil
}

func (s *Server) handleBalance(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling balance event")

	mounts, err := storage.ListMounts()
	if err != nil {
		s.logger.Error("failed to list mounts for balance", zap.Error(err))
		return nil, err
	}

	results := make([]map[string]string, 0)

	for _, m := range mounts {
		logID, err := database.InsertJobLog(s.DB, database.JobLogRecord{
			JobType:    "balance",
			Mountpoint: m.Mountpoint,
			TargetName: m.Label,
			Status:     "running",
			StartedAt:  time.Now(),
		})
		if err != nil {
			s.logger.Error("failed to insert balance log", zap.Error(err))
			continue
		}

		output, err := storage.StartBalance(m.Mountpoint)
		status := "success"
		errMsg := ""
		if err != nil {
			status = "failed"
			errMsg = err.Error()
			s.logger.Error("balance failed",
				zap.Error(err),
				zap.String("mountpoint", m.Mountpoint),
			)
		} else {
			s.logger.Info("balance completed",
				zap.String("mountpoint", m.Mountpoint),
				zap.String("uuid", m.UUID),
			)
		}

		if err := database.UpdateJobLog(s.DB, logID, status, output, errMsg); err != nil {
			s.logger.Error("failed to update balance log", zap.Error(err))
		}

		results = append(results, map[string]string{
			"mountpoint": m.Mountpoint,
			"uuid":       m.UUID,
			"status":     status,
		})
	}

	return results, nil
}
