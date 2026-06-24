package worker

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/worker/event"
	"rosadisk-agent/internal/worker/scheduler"
)

type Worker struct {
	logger    *zap.Logger
	db        *database.Database
	eventBus  *EventBus
	scheduler *scheduler.Scheduler
	wg        sync.WaitGroup
}

func NewWorker(logger *zap.Logger, db *database.Database) *Worker {
	eventBus := NewEventBus(logger, db)
	scheduler := scheduler.NewScheduler(db, eventBus, logger)

	return &Worker{
		logger:    logger,
		db:        db,
		eventBus:  eventBus,
		scheduler: scheduler,
	}
}

func (w *Worker) Start() {
	w.eventBus.Start()
	w.scheduler.Start()
	w.logger.Info("worker started")
}

func (w *Worker) Shutdown(ctx context.Context) error {
	w.scheduler.Stop()
	err := w.eventBus.Shutdown(ctx)
	w.logger.Info("worker shutdown complete")
	return err
}

func (w *Worker) PublishSync(action event.ActionType, data interface{}) event.Result {
	return w.eventBus.PublishSync(action, data)
}

func (w *Worker) PublishAsync(action event.ActionType, data interface{}) {
	w.eventBus.PublishAsync(action, data)
}

func (w *Worker) PublishConcurrent(action event.ActionType, data interface{}) {
	w.eventBus.PublishConcurrent(action, data)
}
