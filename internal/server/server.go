package server

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.uber.org/zap"
	"rosadisk-agent/api"
	"rosadisk-agent/api/gen"
	"rosadisk-agent/internal/database"
	"rosadisk-agent/internal/event"
	"rosadisk-agent/internal/storage"
)

//go:embed docs.html
var docsHTML []byte

type Server struct {
	Echo       *echo.Echo
	DB         *sql.DB
	dispatcher *event.Dispatcher
	eventChan  chan event.Event
	consumer   *event.ConsumerPool
	logger     *zap.Logger
}

func NewServer(logger *zap.Logger, db *sql.DB) *Server {
	e := echo.New()

	eventChan := make(chan event.Event, 100)
	dispatcher := event.NewDispatcher(logger)
	consumer := event.NewConsumerPool(4, eventChan, dispatcher, logger)

	s := &Server{
		Echo:       e,
		DB:         db,
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
	s.dispatcher.Register(event.ActionSubvolumeList, event.HandlerFunc(s.handleSubvolumeList))
	s.dispatcher.Register(event.ActionSubvolumeCreate, event.HandlerFunc(s.handleSubvolumeCreate))
	s.dispatcher.Register(event.ActionSubvolumeGet, event.HandlerFunc(s.handleSubvolumeGet))
	s.dispatcher.Register(event.ActionSubvolumeDelete, event.HandlerFunc(s.handleSubvolumeDelete))
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
			UUID:       m.UUID,
			Label:      m.Label,
			Mountpoint: m.Mountpoint,
			Devices:    m.Devices,
			Used:       m.Used,
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
			Used:       mount.Used,
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

func (s *Server) ListSubvolumes(ctx echo.Context) error {
	s.logger.Debug("received list subvolumes request")

	resultChan := s.emitEvent(event.ActionSubvolumeList, event.SubvolumeListRequest{})
	result := <-resultChan

	if result.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	subvolListResp, ok := result.Data.(event.SubvolumeListResponse)
	if !ok {
		s.logger.Error("unexpected response type from subvolume list handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusOK, subvolListResp)
}

func (s *Server) CreateSubvolume(ctx echo.Context) error {
	s.logger.Debug("received create subvolume request")

	var req gen.CreateSubvolumeRequest
	if err := ctx.Bind(&req); err != nil {
		s.logger.Error("failed to bind request", zap.Error(err))
		return ctx.JSON(http.StatusBadRequest, gen.ErrorResponse{
			Error: "invalid request body",
		})
	}

	var limit int64
	if req.Quota.Limit != nil {
		limit = int64(*req.Quota.Limit)
	}

	var snapshotFreq string
	if req.Snapshots.Frequency != nil {
		snapshotFreq = string(*req.Snapshots.Frequency)
	}

	var snapshotRetention int
	if req.Snapshots.Retention != nil {
		snapshotRetention = *req.Snapshots.Retention
	}

	var incFreq string
	if req.Backups.Incremental.Frequency != nil {
		incFreq = string(*req.Backups.Incremental.Frequency)
	}

	var fullFreq string
	if req.Backups.Full.Frequency != nil {
		fullFreq = string(*req.Backups.Full.Frequency)
	}

	eventReq := event.CreateSubvolumeRequest{
		Name:        req.Name,
		FsUUID:      req.FsUuid.String(),
		Compression: req.Compression,
		Defrag:      req.Defrag,
		NFS:         req.Nfs,
		SMB:         req.Smb,
		Quota: event.QuotaConfig{
			Enabled: req.Quota.Enabled,
			Limit:   limit,
		},
		Snapshots: event.SnapshotConfig{
			Enabled:   req.Snapshots.Enabled,
			Frequency: snapshotFreq,
			Retention: snapshotRetention,
		},
		Backups: event.BackupConfig{
			Incremental: event.BackupSchedule{
				Enabled:   req.Backups.Incremental.Enabled,
				Frequency: incFreq,
			},
			Full: event.BackupSchedule{
				Enabled:   req.Backups.Full.Enabled,
				Frequency: fullFreq,
			},
		},
	}

	resultChan := s.emitEvent(event.ActionSubvolumeCreate, eventReq)
	result := <-resultChan

	if result.Error != nil {
		return ctx.JSON(http.StatusBadRequest, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	createResp, ok := result.Data.(event.CreateSubvolumeResponse)
	if !ok {
		s.logger.Error("unexpected response type from subvolume create handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusCreated, createResp)
}

func (s *Server) GetSubvolume(ctx echo.Context, id openapi_types.UUID) error {
	s.logger.Debug("received get subvolume request", zap.String("id", id.String()))

	eventReq := event.SubvolumeGetRequest{
		ID: id.String(),
	}

	resultChan := s.emitEvent(event.ActionSubvolumeGet, eventReq)
	result := <-resultChan

	if result.Error != nil {
		return ctx.JSON(http.StatusNotFound, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	getResp, ok := result.Data.(event.SubvolumeGetResponse)
	if !ok {
		s.logger.Error("unexpected response type from subvolume get handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusOK, getResp)
}

func (s *Server) DeleteSubvolume(ctx echo.Context, id openapi_types.UUID) error {
	s.logger.Debug("received delete subvolume request", zap.String("id", id.String()))

	eventReq := event.SubvolumeDeleteRequest{
		ID: id.String(),
	}

	resultChan := s.emitEvent(event.ActionSubvolumeDelete, eventReq)
	result := <-resultChan

	if result.Error != nil {
		return ctx.JSON(http.StatusNotFound, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) handleSubvolumeList(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling subvolume list event")

	dbRecords, err := database.ListSubvolumes(s.DB)
	if err != nil {
		s.logger.Error("failed to list subvolumes", zap.Error(err))
		return nil, err
	}

	subvolumes := make([]event.SubvolumeInfo, len(dbRecords))
	for i, r := range dbRecords {
		subvolumes[i] = event.SubvolumeInfo{
			ID:          r.ID,
			Name:        r.Name,
			FsUUID:      r.FsUUID,
			Path:        r.Path,
			Compression: r.Compression,
			Quota: event.QuotaConfig{
				Enabled: r.QuotaEnabled,
				Limit:   r.QuotaLimit,
			},
			Snapshots: event.SnapshotConfig{
				Enabled:   r.SnapshotEnabled,
				Frequency: r.SnapshotFrequency,
				Retention: r.SnapshotRetention,
			},
			Backups: event.BackupConfig{
				Incremental: toEventBackupSchedule(r.BackupIncrementalEnabled, r.BackupIncrementalFrequency),
				Full:        toEventBackupSchedule(r.BackupFullEnabled, r.BackupFullFrequency),
			},
			Defrag:    r.Defrag,
			NFS:       r.NFS,
			SMB:       r.SMB,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
		}
	}

	s.logger.Info("subvolume list completed", zap.Int("count", len(subvolumes)))

	return event.SubvolumeListResponse{Subvolumes: subvolumes}, nil
}

func (s *Server) handleSubvolumeCreate(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling subvolume create event")

	req, ok := data.(event.CreateSubvolumeRequest)
	if !ok {
		s.logger.Error("invalid request type for subvolume create")
		return nil, fmt.Errorf("invalid request type")
	}

	mountpoint, err := storage.FindMountpointByUUID(req.FsUUID)
	if err != nil {
		s.logger.Error("filesystem not mounted", zap.Error(err))
		return nil, err
	}

	subvolPath, err := storage.CreateSubvolumeBtrfs(storage.CreateSubvolumeBtrfsRequest{
		Mountpoint:  mountpoint,
		Name:        req.Name,
		Compression: req.Compression,
		QuotaLimit:  req.Quota.Limit,
	})
	if err != nil {
		s.logger.Error("failed to create btrfs subvolume", zap.Error(err))
		return nil, err
	}

	id := uuid.New().String()

	dbRecord := database.CreateSubvolumeRecord{
		ID:                         id,
		Name:                       req.Name,
		FsUUID:                     req.FsUUID,
		Path:                       subvolPath,
		Compression:                req.Compression,
		QuotaEnabled:               req.Quota.Enabled,
		QuotaLimit:                 req.Quota.Limit,
		SnapshotEnabled:            req.Snapshots.Enabled,
		SnapshotFrequency:          req.Snapshots.Frequency,
		SnapshotRetention:          req.Snapshots.Retention,
		BackupIncrementalEnabled:   req.Backups.Incremental.Enabled,
		BackupIncrementalFrequency: req.Backups.Incremental.Frequency,
		BackupFullEnabled:          req.Backups.Full.Enabled,
		BackupFullFrequency:        req.Backups.Full.Frequency,
		Defrag:                     req.Defrag,
		NFS:                        req.NFS,
		SMB:                        req.SMB,
	}

	if err := database.InsertSubvolumeRecord(s.DB, dbRecord); err != nil {
		_ = storage.DeleteSubvolumeBtrfs(subvolPath)
		s.logger.Error("failed to persist subvolume", zap.Error(err))
		return nil, err
	}

	result := event.CreateSubvolumeResponse{
		Subvolume: event.SubvolumeInfo{
			ID:          id,
			Name:        req.Name,
			FsUUID:      req.FsUUID,
			Path:        subvolPath,
			Compression: req.Compression,
			Quota:       req.Quota,
			Snapshots:   req.Snapshots,
			Backups:     req.Backups,
			Defrag:      req.Defrag,
			NFS:         req.NFS,
			SMB:         req.SMB,
			CreatedAt:   time.Now().Format(time.RFC3339),
		},
	}

	s.logger.Info("subvolume created", zap.String("id", id), zap.String("path", subvolPath))

	return result, nil
}

func (s *Server) handleSubvolumeGet(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling subvolume get event")

	req, ok := data.(event.SubvolumeGetRequest)
	if !ok {
		s.logger.Error("invalid request type for subvolume get")
		return nil, fmt.Errorf("invalid request type")
	}

	r, err := database.GetSubvolume(s.DB, req.ID)
	if err != nil {
		s.logger.Error("failed to get subvolume", zap.Error(err))
		return nil, err
	}

	result := event.SubvolumeGetResponse{
		Subvolume: event.SubvolumeInfo{
			ID:          r.ID,
			Name:        r.Name,
			FsUUID:      r.FsUUID,
			Path:        r.Path,
			Compression: r.Compression,
			Quota: event.QuotaConfig{
				Enabled: r.QuotaEnabled,
				Limit:   r.QuotaLimit,
			},
			Snapshots: event.SnapshotConfig{
				Enabled:   r.SnapshotEnabled,
				Frequency: r.SnapshotFrequency,
				Retention: r.SnapshotRetention,
			},
			Backups: event.BackupConfig{
				Incremental: toEventBackupSchedule(r.BackupIncrementalEnabled, r.BackupIncrementalFrequency),
				Full:        toEventBackupSchedule(r.BackupFullEnabled, r.BackupFullFrequency),
			},
			Defrag:    r.Defrag,
			NFS:       r.NFS,
			SMB:       r.SMB,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
		},
	}

	s.logger.Info("subvolume retrieved", zap.String("id", r.ID))

	return result, nil
}

func (s *Server) handleSubvolumeDelete(ctx context.Context, data interface{}) (interface{}, error) {
	s.logger.Info("handling subvolume delete event")

	req, ok := data.(event.SubvolumeDeleteRequest)
	if !ok {
		s.logger.Error("invalid request type for subvolume delete")
		return nil, fmt.Errorf("invalid request type")
	}

	r, err := database.GetSubvolume(s.DB, req.ID)
	if err != nil {
		return nil, err
	}

	if err := storage.DeleteSubvolumeBtrfs(r.Path); err != nil {
		s.logger.Error("failed to delete btrfs subvolume", zap.Error(err))
		return nil, err
	}

	if err := database.DeleteSubvolumeRecord(s.DB, req.ID); err != nil {
		s.logger.Error("failed to remove subvolume from database", zap.Error(err))
		return nil, err
	}

	s.logger.Info("subvolume deleted", zap.String("id", req.ID))

	return event.SubvolumeDeleteResponse{}, nil
}

func toEventBackupSchedule(enabled bool, freq string) event.BackupSchedule {
	return event.BackupSchedule{
		Enabled:   enabled,
		Frequency: freq,
	}
}
