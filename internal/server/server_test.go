package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"rosadisk-agent/internal/worker/event"
)

type mockEventPublisher struct {
	publishFunc func(action event.ActionType, data interface{}) <-chan event.Result
}

func (m *mockEventPublisher) Publish(action event.ActionType, data interface{}) <-chan event.Result {
	if m.publishFunc != nil {
		return m.publishFunc(action, data)
	}
	resultChan := make(chan event.Result, 1)
	resultChan <- event.Result{Data: []event.DiskInfo{}}
	return resultChan
}

func TestListDisksHandler(t *testing.T) {
	logger, _ := zap.NewProduction()
	mockPub := &mockEventPublisher{}

	s := NewServer(logger, nil, mockPub)

	req := httptest.NewRequest(http.MethodGet, "/v1/disks", nil)
	rec := httptest.NewRecorder()

	s.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetHealth(t *testing.T) {
	logger, _ := zap.NewProduction()
	mockPub := &mockEventPublisher{}

	s := NewServer(logger, nil, mockPub)

	req := httptest.NewRequest(http.MethodGet, "/_health", nil)
	rec := httptest.NewRecorder()

	s.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetOpenAPIJSON(t *testing.T) {
	logger, _ := zap.NewProduction()
	mockPub := &mockEventPublisher{}

	s := NewServer(logger, nil, mockPub)

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()

	s.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestGetOpenAPIYAML(t *testing.T) {
	logger, _ := zap.NewProduction()
	mockPub := &mockEventPublisher{}

	s := NewServer(logger, nil, mockPub)

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rec := httptest.NewRecorder()

	s.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/yaml", rec.Header().Get("Content-Type"))
}

func TestGetDocs(t *testing.T) {
	logger, _ := zap.NewProduction()
	mockPub := &mockEventPublisher{}

	s := NewServer(logger, nil, mockPub)

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()

	s.Echo.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "html")
}
