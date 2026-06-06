package event

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

type ConsumerPool struct {
	workers    int
	eventChan  <-chan Event
	dispatcher *Dispatcher
	logger     *zap.Logger
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewConsumerPool(workers int, eventChan <-chan Event, dispatcher *Dispatcher, logger *zap.Logger) *ConsumerPool {
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
		case event, ok := <-p.eventChan:
			if !ok {
				return
			}
			p.processEvent(id, event)
		}
	}
}

func (p *ConsumerPool) processEvent(workerID int, event Event) {
	start := time.Now()

	p.logger.Debug("worker processing event",
		zap.Int("worker", workerID),
		zap.String("action", string(event.Action)),
	)

	resultChan := p.dispatcher.Dispatch(p.ctx, event)
	result := <-resultChan

	duration := time.Since(start)

	if result.Error != nil {
		p.logger.Error("event handler failed",
			zap.Int("worker", workerID),
			zap.String("action", string(event.Action)),
			zap.Error(result.Error),
			zap.Duration("duration", duration),
		)
	} else {
		p.logger.Debug("event handled successfully",
			zap.Int("worker", workerID),
			zap.String("action", string(event.Action)),
			zap.Duration("duration", duration),
		)
	}

	event.Result <- result
}
