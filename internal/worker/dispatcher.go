package worker

import (
	"context"

	"go.uber.org/zap"
	"rosadisk-agent/internal/worker/event"
	"rosadisk-agent/internal/worker/handler"
)

type Dispatcher struct {
	handlers map[event.ActionType]handler.Handler
	logger   *zap.Logger
}

func NewDispatcher(logger *zap.Logger, handlers map[event.ActionType]handler.Handler) *Dispatcher {
	return &Dispatcher{
		handlers: handlers,
		logger:   logger,
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, evt event.Event) {
	h, ok := d.handlers[evt.Action]
	if !ok {
		d.logger.Warn("no handler registered for event",
			zap.String("action", string(evt.Action)),
		)
		if evt.Result != nil {
			evt.Result <- event.Result{Error: ErrNoHandler}
		}
		return
	}

	d.logger.Debug("dispatching event",
		zap.String("action", string(evt.Action)),
		zap.Time("timestamp", evt.Timestamp),
	)

	go func() {
		data, err := h.Handle(ctx, evt.Data)
		if evt.Result != nil {
			evt.Result <- event.Result{Data: data, Error: err}
		}
	}()
}

var ErrNoHandler = &HandlerError{message: "no handler registered for event action"}

type HandlerError struct {
	message string
}

func (e *HandlerError) Error() string {
	return e.message
}
