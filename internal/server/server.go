package server

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"rosadisk-agent/api"
	"rosadisk-agent/api/gen"
	"rosadisk-agent/internal/event"
	"rosadisk-agent/internal/storage"
)

//go:embed docs.html
var docsHTML []byte

type Server struct {
	Echo       *echo.Echo
	dispatcher *event.Dispatcher
	eventChan  chan event.Event
	consumer   *event.ConsumerPool
	logger     *zap.Logger
}

func NewServer(logger *zap.Logger) *Server {
	e := echo.New()

	eventChan := make(chan event.Event, 100)
	dispatcher := event.NewDispatcher(logger)
	consumer := event.NewConsumerPool(4, eventChan, dispatcher, logger)

	s := &Server{
		Echo:       e,
		dispatcher: dispatcher,
		eventChan:  eventChan,
		consumer:   consumer,
		logger:     logger,
	}

	s.registerHandlers()
	gen.RegisterHandlers(e, s)
	e.GET("/docs", s.GetDocs)

	consumer.Start()

	return s
}

func (s *Server) Start(addr string) error {
	return s.Echo.Start(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.consumer.Stop()
	close(s.eventChan)
	return s.Echo.Shutdown(ctx)
}

func (s *Server) registerHandlers() {
	s.dispatcher.Register(event.ActionDiskList, event.HandlerFunc(s.handleDiskList))
	s.dispatcher.Register(event.ActionFilesystemList, event.HandlerFunc(s.handleFilesystemList))
	s.dispatcher.Register(event.ActionFilesystemCreate, event.HandlerFunc(s.handleFilesystemCreate))
	s.logger.Info("event handlers registered")
}

func (s *Server) handleDiskList(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling disk list event")

	storageDisks, err := storage.ListDisks()
	if err != nil {
		s.logger.Error("failed to list disks", zap.Error(err))
		return nil, err
	}

	disks := make([]event.DiskInfo, len(storageDisks))
	for i, d := range storageDisks {
		disks[i] = event.DiskInfo{
			Name:   d.Name,
			Size:   d.Size,
			Type:   d.Type,
			Vendor: d.Vendor,
			Model:  d.Model,
			FsType: d.FsType,
		}
	}

	s.logger.Info("disk list completed", zap.Int("count", len(disks)))

	return event.DiskListResponse{Disks: disks}, nil
}

func (s *Server) handleFilesystemList(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling filesystem list event")

	storageFS, err := storage.ListFilesystems()
	if err != nil {
		s.logger.Error("failed to list filesystems", zap.Error(err))
		return nil, err
	}

	filesystems := make([]event.FilesystemInfo, len(storageFS))
	for i, fs := range storageFS {
		filesystems[i] = event.FilesystemInfo{
			UUID:        fs.UUID,
			Label:       fs.Label,
			Size:        fs.Size,
			Devices:     fs.Devices,
			RaidProfile: fs.RaidProfile,
		}
	}

	s.logger.Info("filesystem list completed", zap.Int("count", len(filesystems)))

	return event.FilesystemListResponse{Filesystems: filesystems}, nil
}

func (s *Server) handleFilesystemCreate(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling filesystem create event")

	req, ok := data.(event.CreateFilesystemRequest)
	if !ok {
		s.logger.Error("invalid request type for filesystem create")
		return nil, fmt.Errorf("invalid request type")
	}

	var label string
	if req.Label != nil {
		label = *req.Label
	}

	raidProfile := req.RaidProfile
	if raidProfile == "" {
		raidProfile = "single"
	}

	fs, err := storage.CreateFilesystem(req.Devices, label, raidProfile)
	if err != nil {
		s.logger.Error("failed to create filesystem", zap.Error(err))
		return nil, err
	}

	result := event.CreateFilesystemResponse{
		Filesystem: event.FilesystemInfo{
			UUID:        fs.UUID,
			Label:       fs.Label,
			Size:        fs.Size,
			Devices:     fs.Devices,
			RaidProfile: fs.RaidProfile,
		},
	}

	s.logger.Info("filesystem created", zap.String("uuid", fs.UUID))

	return result, nil
}

func (s *Server) emitEvent(action event.ActionType, data interface{}) <-chan event.Result {
	resultChan := make(chan event.Result, 1)
	evt := event.Event{
		Action:    action,
		Data:      data,
		Timestamp: time.Now(),
		Result:    resultChan,
	}
	s.eventChan <- evt
	return resultChan
}

func (s *Server) GetHealth(ctx echo.Context) error {
	resp := gen.HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
	}
	return ctx.JSON(http.StatusOK, resp)
}

func (s *Server) GetOpenAPIJSON(ctx echo.Context) error {
	specJSON, err := gen.GetSpecJSON()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Blob(http.StatusOK, "application/json", specJSON)
}

func (s *Server) GetOpenAPIYAML(ctx echo.Context) error {
	return ctx.Blob(http.StatusOK, "text/yaml", api.OpenAPIYAML)
}

func (s *Server) GetDocs(ctx echo.Context) error {
	return ctx.HTML(http.StatusOK, string(docsHTML))
}

func (s *Server) ListDisks(ctx echo.Context) error {
	s.logger.Debug("received list disks request")

	resultChan := s.emitEvent(event.ActionDiskList, event.DiskListRequest{})
	result := <-resultChan

	if result.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	diskListResp, ok := result.Data.(event.DiskListResponse)
	if !ok {
		s.logger.Error("unexpected response type from disk list handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusOK, diskListResp)
}

func (s *Server) ListFilesystems(ctx echo.Context) error {
	s.logger.Debug("received list filesystems request")

	resultChan := s.emitEvent(event.ActionFilesystemList, event.FilesystemListRequest{})
	result := <-resultChan

	if result.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	fsListResp, ok := result.Data.(event.FilesystemListResponse)
	if !ok {
		s.logger.Error("unexpected response type from filesystem list handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusOK, fsListResp)
}

func (s *Server) CreateFilesystem(ctx echo.Context) error {
	s.logger.Debug("received create filesystem request")

	var req gen.CreateFilesystemRequest
	if err := ctx.Bind(&req); err != nil {
		s.logger.Error("failed to bind request", zap.Error(err))
		return ctx.JSON(http.StatusBadRequest, gen.ErrorResponse{
			Error: "invalid request body",
		})
	}

	eventReq := event.CreateFilesystemRequest{
		Devices: req.Devices,
		Label:   req.Label,
	}
	if req.RaidProfile != nil {
		eventReq.RaidProfile = *req.RaidProfile
	}

	resultChan := s.emitEvent(event.ActionFilesystemCreate, eventReq)
	result := <-resultChan

	if result.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	createResp, ok := result.Data.(event.CreateFilesystemResponse)
	if !ok {
		s.logger.Error("unexpected response type from filesystem create handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusCreated, createResp)
}
