package event

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

type Handler interface {
	Handle(ctx context.Context, data interface{}) (interface{}, error)
}

type HandlerFunc func(ctx context.Context, data interface{}) (interface{}, error)

func (f HandlerFunc) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	return f(ctx, data)
}

type Dispatcher struct {
	handlers map[ActionType]Handler
	logger   *zap.Logger
	mu       sync.RWMutex
}

func NewDispatcher(logger *zap.Logger) *Dispatcher {
	return &Dispatcher{
		handlers: make(map[ActionType]Handler),
		logger:   logger,
	}
}

func (d *Dispatcher) Register(action ActionType, handler Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[action] = handler
	d.logger.Debug("registered event handler", zap.String("action", string(action)))
}

func (d *Dispatcher) Dispatch(ctx context.Context, event Event) <-chan Result {
	resultChan := make(chan Result, 1)

	d.mu.RLock()
	handler, ok := d.handlers[event.Action]
	d.mu.RUnlock()

	if !ok {
		d.logger.Warn("no handler registered for event",
			zap.String("action", string(event.Action)),
		)
		resultChan <- Result{Error: ErrNoHandler}
		return resultChan
	}

	d.logger.Debug("dispatching event",
		zap.String("action", string(event.Action)),
		zap.Time("timestamp", event.Timestamp),
	)

	go func() {
		data, err := handler.Handle(ctx, event.Data)
		resultChan <- Result{Data: data, Error: err}
	}()

	return resultChan
}

var ErrNoHandler = &HandlerError{message: "no handler registered for event action"}

type HandlerError struct {
	message string
}

func (e *HandlerError) Error() string {
	return e.message
}
