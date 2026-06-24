package worker

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"rosadisk-agent/internal/worker/event"
)

type ConsumerPool struct {
	workers    int
	eventChan  <-chan event.Event
	dispatcher *Dispatcher
	logger     *zap.Logger
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewConsumerPool(workers int, eventChan <-chan event.Event, dispatcher *Dispatcher, logger *zap.Logger) *ConsumerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &ConsumerPool{
		workers:    workers,
		eventChan:  eventChan,
		dispatcher: dispatcher,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (p *ConsumerPool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	p.logger.Info("consumer pool started", zap.Int("workers", p.workers))
}

func (p *ConsumerPool) Stop() {
	p.cancel()
	p.wg.Wait()
	p.logger.Info("consumer pool stopped")
}

func (p *ConsumerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case evt, ok := <-p.eventChan:
			if !ok {
				return
			}
			p.processEvent(id, evt)
		}
	}
}

func (p *ConsumerPool) processEvent(workerID int, evt event.Event) {
	start := time.Now()

	p.logger.Debug("worker processing event",
		zap.Int("worker", workerID),
		zap.String("action", string(evt.Action)),
	)

	p.dispatcher.Dispatch(p.ctx, evt)

	duration := time.Since(start)

	if evt.Result != nil {
		p.logger.Debug("event dispatched with result",
			zap.Int("worker", workerID),
			zap.String("action", string(evt.Action)),
			zap.Duration("duration", duration),
		)
	} else {
		p.logger.Debug("event dispatched (fire-and-forget)",
			zap.Int("worker", workerID),
			zap.String("action", string(evt.Action)),
			zap.Duration("duration", duration),
		)
	}
}
