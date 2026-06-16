package server

import (
	"context"
	"time"

	"go.uber.org/zap"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/storage"
)

func (s *Server) handleBackup(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling backup event")
	return map[string]string{"status": "backup completed (dummy)"}, nil
}

func (s *Server) handleSnapshot(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling snapshot event")
	return map[string]string{"status": "snapshot completed (dummy)"}, nil
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
