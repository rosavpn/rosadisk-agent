package worker

import (
	"context"
	"database/sql"
	"time"

	"go.uber.org/zap"
	"rosadisk-agent/internal/worker/event"
	"rosadisk-agent/internal/worker/handler"
)

type SyncEventPublisher interface {
	PublishSync(action event.ActionType, data interface{}) event.Result
}

type AsyncEventPublisher interface {
	PublishAsync(action event.ActionType, data interface{})
}

type ConcurrentEventPublisher interface {
	PublishConcurrent(action event.ActionType, data interface{})
}

type EventBus struct {
	logger         *zap.Logger
	syncChan       chan event.Event
	asyncChan      chan event.Event
	concurrentChan chan event.Event
	syncPool       *ConsumerPool
	asyncPool      *ConsumerPool
	concurrentPool *ConsumerPool
	dispatcher     *Dispatcher
}

func NewEventBus(logger *zap.Logger, db *sql.DB) *EventBus {
	syncChan := make(chan event.Event, 100)
	asyncChan := make(chan event.Event, 100)
	concurrentChan := make(chan event.Event, 100)

	handlers := handler.RegisterAll(logger, db)
	dispatcher := NewDispatcher(logger, handlers)

	syncPool := NewConsumerPool(1, syncChan, dispatcher, logger)
	asyncPool := NewConsumerPool(1, asyncChan, dispatcher, logger)
	concurrentPool := NewConsumerPool(5, concurrentChan, dispatcher, logger)

	return &EventBus{
		logger:         logger,
		syncChan:       syncChan,
		asyncChan:      asyncChan,
		concurrentChan: concurrentChan,
		syncPool:       syncPool,
		asyncPool:      asyncPool,
		concurrentPool: concurrentPool,
		dispatcher:     dispatcher,
	}
}

func (bus *EventBus) Start() {
	bus.syncPool.Start()
	bus.asyncPool.Start()
	bus.concurrentPool.Start()
	bus.logger.Info("event bus started")
}

func (bus *EventBus) Shutdown(ctx context.Context) error {
	bus.syncPool.Stop()
	bus.asyncPool.Stop()
	bus.concurrentPool.Stop()
	close(bus.syncChan)
	close(bus.asyncChan)
	close(bus.concurrentChan)
	bus.logger.Info("event bus shutdown complete")
	return nil
}

func (bus *EventBus) PublishSync(action event.ActionType, data interface{}) event.Result {
	resultChan := make(chan event.Result, 1)
	evt := event.Event{
		Action:    action,
		Data:      data,
		Timestamp: time.Now(),
		Result:    resultChan,
	}
	bus.syncChan <- evt
	return <-resultChan
}

func (bus *EventBus) PublishAsync(action event.ActionType, data interface{}) {
	evt := event.Event{
		Action:    action,
		Data:      data,
		Timestamp: time.Now(),
		Result:    nil,
	}
	bus.asyncChan <- evt
}

func (bus *EventBus) PublishConcurrent(action event.ActionType, data interface{}) {
	evt := event.Event{
		Action:    action,
		Data:      data,
		Timestamp: time.Now(),
		Result:    nil,
	}
	bus.concurrentChan <- evt
}
