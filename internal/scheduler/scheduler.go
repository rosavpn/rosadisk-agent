package scheduler

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"go.uber.org/zap"
	"rosadisk-agent/internal/config"
	"rosadisk-agent/internal/event"
)

type Scheduler struct {
	db        *sql.DB
	eventChan chan<- event.Event
	logger    *zap.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	lastRun   map[event.ActionType]string
}

func NewScheduler(db *sql.DB, eventChan chan<- event.Event, logger *zap.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		db:        db,
		eventChan: eventChan,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		lastRun:   make(map[event.ActionType]string),
	}
}

func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.run()
	s.logger.Info("scheduler started")
}

func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()
	s.logger.Info("scheduler stopped")
}

func (s *Scheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	s.checkAndEmit()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkAndEmit()
		}
	}
}

func (s *Scheduler) checkAndEmit() {
	cfg, err := config.GetConfig(s.db)
	if err != nil {
		s.logger.Error("scheduler failed to read config", zap.Error(err))
		return
	}

	now := time.Now().Format("15:04")

	jobs := []struct {
		action  event.ActionType
		enabled bool
		time    string
		data    interface{}
	}{
		{event.ActionBackup, cfg.Backup.Enabled, cfg.Backup.Time, event.BackupRequest{}},
		{event.ActionSnapshot, cfg.Snapshot.Enabled, cfg.Snapshot.Time, event.SnapshotRequest{}},
		{event.ActionDefrag, cfg.Defrag.Enabled, cfg.Defrag.Time, event.DefragRequest{}},
		{event.ActionScrub, cfg.Scrub.Enabled, cfg.Scrub.Time, event.ScrubRequest{}},
		{event.ActionBalance, cfg.Balance.Enabled, cfg.Balance.Time, event.BalanceRequest{}},
	}

	for _, job := range jobs {
		if job.enabled && job.time == now && s.lastRun[job.action] != now {
			s.lastRun[job.action] = now
			s.emitEvent(job.action, job.data)
		}
	}
}

func (s *Scheduler) emitEvent(action event.ActionType, data interface{}) {
	s.logger.Info("scheduler emitting event", zap.String("action", string(action)))

	resultChan := make(chan event.Result, 1)
	evt := event.Event{
		Action:    action,
		Data:      data,
		Timestamp: time.Now(),
		Result:    resultChan,
	}

	select {
	case s.eventChan <- evt:
		result := <-resultChan
		if result.Error != nil {
			s.logger.Error("scheduler event failed",
				zap.String("action", string(action)),
				zap.Error(result.Error),
			)
		} else {
			s.logger.Info("scheduler event completed",
				zap.String("action", string(action)),
				zap.Any("result", result.Data),
			)
		}
	case <-s.ctx.Done():
		return
	}
}
