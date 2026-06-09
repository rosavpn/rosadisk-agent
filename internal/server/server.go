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
	s.dispatcher.Register(event.ActionMountList, event.HandlerFunc(s.handleMountList))
	s.dispatcher.Register(event.ActionMountCreate, event.HandlerFunc(s.handleMountCreate))
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

	fs, err := storage.CreateFilesystem(req.Devices, req.Label, req.RaidProfile)
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

func (s *Server) handleMountList(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling mount list event")

	storageMounts, err := storage.ListMounts()
	if err != nil {
		s.logger.Error("failed to list mounts", zap.Error(err))
		return nil, err
	}

	mounts := make([]event.MountInfo, len(storageMounts))
	for i, m := range storageMounts {
		mounts[i] = event.MountInfo{
			UUID:        m.UUID,
			Label:       m.Label,
			Mountpoint:  m.Mountpoint,
			Devices:     m.Devices,
			RaidProfile: m.RaidProfile,
			Size:        m.Size,
		}
	}

	s.logger.Info("mount list completed", zap.Int("count", len(mounts)))

	return event.MountListResponse{Mounts: mounts}, nil
}

func (s *Server) handleMountCreate(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling mount create event")

	req, ok := data.(event.MountRequest)
	if !ok {
		s.logger.Error("invalid request type for mount create")
		return nil, fmt.Errorf("invalid request type")
	}

	mount, err := storage.MountByUUID(req.UUID)
	if err != nil {
		s.logger.Error("failed to mount filesystem", zap.Error(err))
		return nil, err
	}

	result := event.MountResponse{
		Mount: event.MountInfo{
			UUID:       mount.UUID,
			Label:      mount.Label,
			Mountpoint: mount.Mountpoint,
			Devices:    mount.Devices,
		},
	}

	s.logger.Info("filesystem mounted", zap.String("uuid", mount.UUID), zap.String("mountpoint", mount.Mountpoint))

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
		Devices:     req.Devices,
		Label:       req.Label,
		RaidProfile: string(req.RaidProfile),
	}

	resultChan := s.emitEvent(event.ActionFilesystemCreate, eventReq)
	result := <-resultChan

	if result.Error != nil {
		return ctx.JSON(http.StatusBadRequest, gen.ErrorResponse{
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

func (s *Server) ListMounts(ctx echo.Context) error {
	s.logger.Debug("received list mounts request")

	resultChan := s.emitEvent(event.ActionMountList, event.MountListRequest{})
	result := <-resultChan

	if result.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	mountListResp, ok := result.Data.(event.MountListResponse)
	if !ok {
		s.logger.Error("unexpected response type from mount list handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusOK, mountListResp)
}

func (s *Server) MountFilesystem(ctx echo.Context) error {
	s.logger.Debug("received mount filesystem request")

	var req gen.MountRequest
	if err := ctx.Bind(&req); err != nil {
		s.logger.Error("failed to bind request", zap.Error(err))
		return ctx.JSON(http.StatusBadRequest, gen.ErrorResponse{
			Error: "invalid request body",
		})
	}

	eventReq := event.MountRequest{
		UUID: req.Uuid,
	}

	resultChan := s.emitEvent(event.ActionMountCreate, eventReq)
	result := <-resultChan

	if result.Error != nil {
		return ctx.JSON(http.StatusBadRequest, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	mountResp, ok := result.Data.(event.MountResponse)
	if !ok {
		s.logger.Error("unexpected response type from mount create handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusCreated, mountResp)
}
