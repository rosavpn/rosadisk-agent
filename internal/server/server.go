package server

import (
	"context"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"rosadisk-agent/api/gen"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/worker"
)

type Server struct {
	Echo     *echo.Echo
	DB       *database.Database
	eventPub worker.SyncEventPublisher
	logger   *zap.Logger
}

func NewServer(logger *zap.Logger, db *database.Database, eventPub worker.SyncEventPublisher) *Server {
	e := echo.New()

	s := &Server{
		Echo:     e,
		DB:       db,
		eventPub: eventPub,
		logger:   logger,
	}

	gen.RegisterHandlers(e, s)
	e.GET("/docs", s.GetDocs)

	return s
}

func (s *Server) Start(addr string) error {
	return s.Echo.Start(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.Echo.Shutdown(ctx)
}
