package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"rosadisk-agent/internal/config"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/worker/event"
)

type eventBus interface {
	event.AsyncEventPublisher
	event.ConcurrentEventPublisher
}

type Scheduler struct {
	db       *database.Database
	eventBus eventBus
	logger   *zap.Logger
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	lastRun  map[string]string
}

func NewScheduler(db *database.Database, eventBus eventBus, logger *zap.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		db:       db,
		eventBus: eventBus,
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
		lastRun:  make(map[string]string),
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

	now := time.Now()

	s.checkVolumeJob(event.ActionBackup, cfg.Backup, now)
	s.checkSnapshotJob(cfg.Snapshot)
	s.checkVolumeJob(event.ActionDefrag, cfg.Defrag, now)
	s.checkDiskJob(event.ActionScrubCheck, cfg.Scrub, now)
	s.checkDiskJob(event.ActionBalanceCheck, cfg.Balance, now)
}

func (s *Scheduler) checkVolumeJob(action event.ActionType, schedule config.VolumeJobSchedule, now time.Time) {
	if !schedule.Enabled {
		return
	}

	minute := now.Minute()
	timeHHMM := now.Format("15:04")
	weekday := strings.ToLower(now.Weekday().String())
	day := now.Day()

	var shouldEmit bool
	var key string

	if schedule.HourlyMinute == minute {
		key = fmt.Sprintf("%s:hourly:%d", action, now.Hour())
		if s.lastRun[key] != now.Format("2006-01-02") {
			shouldEmit = true
		}
	} else if schedule.Time == timeHHMM {
		if schedule.WeeklyDay != "" && schedule.WeeklyDay == weekday {
			_, weekNum := now.ISOWeek()
			key = fmt.Sprintf("%s:weekly:%d", action, weekNum)
			if s.lastRun[key] != now.Format("2006") {
				shouldEmit = true
			}
		} else if schedule.MonthlyDay > 0 && schedule.MonthlyDay == day {
			key = fmt.Sprintf("%s:monthly", action)
			if s.lastRun[key] != now.Format("2006-01") {
				shouldEmit = true
			}
		} else {
			key = fmt.Sprintf("%s:daily", action)
			if s.lastRun[key] != now.Format("2006-01-02") {
				shouldEmit = true
			}
		}
	}

	if shouldEmit {
		s.lastRun[key] = getTimeKey(now, key)
		s.eventBus.PublishAsync(action, s.getVolumeRequest(action))
	}
}

func (s *Scheduler) checkDiskJob(action event.ActionType, schedule config.DiskJobSchedule, now time.Time) {
	if !schedule.Enabled {
		return
	}

	timeHHMM := now.Format("15:04")
	weekday := strings.ToLower(now.Weekday().String())
	day := now.Day()

	var shouldEmit bool
	var key string

	switch schedule.Frequency {
	case "weekly":
		if schedule.Time == timeHHMM && schedule.DayOfWeek == weekday {
			_, weekNum := now.ISOWeek()
			key = fmt.Sprintf("%s:weekly:%d", action, weekNum)
			if s.lastRun[key] != now.Format("2006") {
				shouldEmit = true
			}
		}
	case "monthly":
		if schedule.Time == timeHHMM && schedule.DayOfMonth == day {
			key = fmt.Sprintf("%s:monthly", action)
			if s.lastRun[key] != now.Format("2006-01") {
				shouldEmit = true
			}
		}
	}

	if shouldEmit {
		s.lastRun[key] = getTimeKey(now, key)
		s.eventBus.PublishAsync(action, s.getDiskRequest(action))
	}
}

func (s *Scheduler) checkSnapshotJob(schedule config.VolumeJobSchedule) {
	if !schedule.Enabled {
		return
	}

	s.eventBus.PublishAsync(event.ActionSnapshotCheck, event.SnapshotCheckRequest{EventBus: s.eventBus})
}

func (s *Scheduler) getVolumeRequest(action event.ActionType) interface{} {
	switch action {
	case event.ActionBackup:
		return event.BackupRequest{}
	case event.ActionDefrag:
		return event.DefragRequest{}
	default:
		return event.BackupRequest{}
	}
}

func (s *Scheduler) getDiskRequest(action event.ActionType) interface{} {
	switch action {
	case event.ActionScrubCheck:
		return event.ScrubCheckRequest{EventBus: s.eventBus}
	case event.ActionBalanceCheck:
		return event.BalanceCheckRequest{EventBus: s.eventBus}
	default:
		return event.ScrubCheckRequest{EventBus: s.eventBus}
	}
}

func getTimeKey(now time.Time, key string) string {
	if strings.Contains(key, "hourly") {
		return now.Format("2006-01-02")
	}
	if strings.Contains(key, "weekly") {
		return now.Format("2006")
	}
	if strings.Contains(key, "monthly") {
		return now.Format("2006-01")
	}
	return now.Format("2006-01-02")
}
