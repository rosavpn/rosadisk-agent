package worker

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"rosadisk-agent/internal/worker/event"
	"rosadisk-agent/internal/worker/handler"
	"rosadisk-agent/internal/worker/scheduler"

	"go.uber.org/zap"
)

type Worker struct {
	logger     *zap.Logger
	db         *sql.DB
	eventChan  chan event.Event
	dispatcher *Dispatcher
	consumer   *ConsumerPool
	scheduler  *scheduler.Scheduler
	wg         sync.WaitGroup
}

func NewWorker(logger *zap.Logger, db *sql.DB) *Worker {
	eventChan := make(chan event.Event, 100)
	handlers := handler.RegisterAll(logger, db)
	dispatcher := NewDispatcher(logger, handlers)
	consumer := NewConsumerPool(4, eventChan, dispatcher, logger)

	w := &Worker{
		logger:     logger,
		db:         db,
		eventChan:  eventChan,
		dispatcher: dispatcher,
		consumer:   consumer,
	}

	w.scheduler = scheduler.NewScheduler(db, eventChan, logger)

	return w
}

func (w *Worker) Start() {
	w.consumer.Start()
	w.scheduler.Start()
	w.logger.Info("worker started")
}

func (w *Worker) Shutdown(ctx context.Context) error {
	w.scheduler.Stop()
	w.consumer.Stop()
	close(w.eventChan)
	w.logger.Info("worker shutdown complete")
	return nil
}

func (w *Worker) Publish(action event.ActionType, data interface{}) <-chan event.Result {
	resultChan := make(chan event.Result, 1)
	evt := event.Event{
		Action:    action,
		Data:      data,
		Timestamp: time.Now(),
		Result:    resultChan,
	}
	w.eventChan <- evt
	return resultChan
}
