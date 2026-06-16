package scheduler

import (
	"context"
	"database/sql"
	"fmt"
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
	lastRun   map[string]string
}

func NewScheduler(db *sql.DB, eventChan chan<- event.Event, logger *zap.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		db:        db,
		eventChan: eventChan,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		lastRun:   make(map[string]string),
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
	s.checkVolumeJob(event.ActionSnapshot, cfg.Snapshot, now)
	s.checkVolumeJob(event.ActionDefrag, cfg.Defrag, now)
	s.checkDiskJob(event.ActionScrub, cfg.Scrub, now)
	s.checkDiskJob(event.ActionBalance, cfg.Balance, now)
}

func (s *Scheduler) checkVolumeJob(action event.ActionType, schedule config.VolumeJobSchedule, now time.Time) {
	if !schedule.Enabled {
		return
	}

	minute := now.Minute()
	timeHHMM := now.Format("15:04")
	weekday := now.Weekday().String()
	day := now.Day()

	weekdayLower := toLowerWeekday(weekday)

	if schedule.HourlyMinute == minute {
		key := fmt.Sprintf("%s:hourly:%d", action, now.Hour())
		if s.lastRun[key] != now.Format("2006-01-02") {
			s.lastRun[key] = now.Format("2006-01-02")
			s.emitEvent(action, event.BackupRequest{})
			return
		}
	}

	if schedule.Time == timeHHMM {
		dailyKey := fmt.Sprintf("%s:daily", action)
		if s.lastRun[dailyKey] != now.Format("2006-01-02") {
			s.lastRun[dailyKey] = now.Format("2006-01-02")
			s.emitEvent(action, event.BackupRequest{})
			return
		}
	}

	if schedule.Time == timeHHMM && schedule.WeeklyDay != "" && schedule.WeeklyDay == weekdayLower {
		_, weekNum := now.ISOWeek()
		weeklyKey := fmt.Sprintf("%s:weekly:%d", action, weekNum)
		if s.lastRun[weeklyKey] != now.Format("2006") {
			s.lastRun[weeklyKey] = now.Format("2006")
			s.emitEvent(action, event.BackupRequest{})
			return
		}
	}

	if schedule.Time == timeHHMM && schedule.MonthlyDay > 0 && schedule.MonthlyDay == day {
		monthlyKey := fmt.Sprintf("%s:monthly", action)
		if s.lastRun[monthlyKey] != now.Format("2006-01") {
			s.lastRun[monthlyKey] = now.Format("2006-01")
			s.emitEvent(action, event.BackupRequest{})
			return
		}
	}
}

func (s *Scheduler) checkDiskJob(action event.ActionType, schedule config.DiskJobSchedule, now time.Time) {
	if !schedule.Enabled {
		return
	}

	timeHHMM := now.Format("15:04")
	weekday := toLowerWeekday(now.Weekday().String())
	day := now.Day()

	switch schedule.Frequency {
	case "weekly":
		if schedule.Time == timeHHMM && schedule.DayOfWeek == weekday {
			_, weekNum := now.ISOWeek()
			key := fmt.Sprintf("%s:weekly:%d", action, weekNum)
			if s.lastRun[key] != now.Format("2006") {
				s.lastRun[key] = now.Format("2006")
				s.emitEvent(action, event.ScrubRequest{})
			}
		}

	case "monthly":
		if schedule.Time == timeHHMM && schedule.DayOfMonth == day {
			key := fmt.Sprintf("%s:monthly", action)
			if s.lastRun[key] != now.Format("2006-01") {
				s.lastRun[key] = now.Format("2006-01")
				s.emitEvent(action, event.ScrubRequest{})
			}
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

func toLowerWeekday(goWeekday string) string {
	m := map[string]string{
		"Monday":    "monday",
		"Tuesday":   "tuesday",
		"Wednesday": "wednesday",
		"Thursday":  "thursday",
		"Friday":    "friday",
		"Saturday":  "saturday",
		"Sunday":    "sunday",
	}
	if v, ok := m[goWeekday]; ok {
		return v
	}
	return ""
}
