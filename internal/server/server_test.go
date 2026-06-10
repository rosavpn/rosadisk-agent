package server

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"rosadisk-agent/internal/event"
)

func TestHandleDiskList(t *testing.T) {
	logger, _ := zap.NewProduction()
	dispatcher := event.NewDispatcher(logger)

	s := &Server{
		DB:         nil,
		dispatcher: dispatcher,
		eventChan:  make(chan event.Event, 10),
		logger:     logger,
	}

	s.registerHandlers()

	go func() {
		for evt := range s.eventChan {
			go func(e event.Event) {
				resultChan := s.dispatcher.Dispatch(context.Background(), e)
				result := <-resultChan
				e.Result <- result
			}(evt)
		}
	}()

	resultChan := s.emitEvent(event.ActionDiskList, event.DiskListRequest{})
	result := <-resultChan

	require.NoError(t, result.Error)
	resp, ok := result.Data.(event.DiskListResponse)
	require.True(t, ok)
	assert.Greater(t, len(resp.Disks), 0, "should have at least one disk")
}

func TestListDisksHandler(t *testing.T) {
	logger, _ := zap.NewProduction()
	s := NewServer(logger, nil)
	defer s.Shutdown(context.Background())

	req := httptest.NewRequest(http.MethodGet, "/v1/disks", nil)
	rec := httptest.NewRecorder()

	s.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestEventEmission(t *testing.T) {
	logger, _ := zap.NewProduction()
	eventChan := make(chan event.Event, 10)
	dispatcher := event.NewDispatcher(logger)
	dispatcher.Register(event.ActionDiskList, event.HandlerFunc(func(ctx context.Context, data interface{}) (interface{}, error) {
		return event.DiskListResponse{Disks: []event.DiskInfo{}}, nil
	}))

	s := &Server{
		DB:         nil,
		eventChan:  eventChan,
		dispatcher: dispatcher,
		logger:     logger,
	}

	resultChan := s.emitEvent(event.ActionDiskList, event.DiskListRequest{})

	select {
	case evt := <-eventChan:
		assert.Equal(t, event.ActionDiskList, evt.Action)
		assert.IsType(t, event.DiskListRequest{}, evt.Data)
		require.NotNil(t, evt.Result)

		go func(e event.Event) {
			resultChan := s.dispatcher.Dispatch(context.Background(), e)
			result := <-resultChan
			e.Result <- result
		}(evt)
	case <-time.After(time.Second):
		t.Fatal("event not emitted")
	}

	select {
	case <-resultChan:
	case <-time.After(time.Second):
		t.Fatal("result channel not receiving")
	}
}
