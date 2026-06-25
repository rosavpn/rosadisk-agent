package server

import (
	_ "embed"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.uber.org/zap"
	"rosadisk-agent/api"
	"rosadisk-agent/api/gen"
	"rosadisk-agent/internal/config"
	"rosadisk-agent/internal/worker/event"
)

//go:embed docs.html
var docsHTML []byte

func (s *Server) emitEvent(action event.ActionType, data interface{}) event.Result {
	return s.eventPub.PublishSync(action, data)
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

func (s *Server) ListJobLogs(ctx echo.Context, params gen.ListJobLogsParams) error {
	limit := 50
	if params.Limit != nil && *params.Limit > 0 {
		limit = *params.Limit
	}

	jobType := ""
	if params.JobType != nil {
		jobType = *params.JobType
	}
	status := ""
	if params.Status != nil {
		status = *params.Status
	}

	records, err := s.DB.ListJobLogs(jobType, status, limit)
	if err != nil {
		s.logger.Error("failed to list job logs", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "failed to list job logs",
		})
	}

	summaries := make([]gen.JobLogSummary, len(records))
	for i, r := range records {
		summaries[i] = toJobLogSummary(r)
	}

	return ctx.JSON(http.StatusOK, summaries)
}

func (s *Server) GetJobLog(ctx echo.Context, id int) error {
	record, err := s.DB.GetJobLog(int64(id))
	if err != nil {
		return ctx.JSON(http.StatusNotFound, gen.ErrorResponse{
			Error: "job log not found",
		})
	}

	detail := gen.JobLogDetail{
		CompletedAt: nilTime(record.CompletedAt),
		Error:       nullStringPtr(record.Error),
		Id:          int(record.ID),
		JobType:     record.JobType,
		Mountpoint:  nullStringPtr(record.Mountpoint),
		Output:      nullStringPtr(record.Output),
		StartedAt:   record.StartedAt,
		Status:      record.Status,
		SubvolumeId: nullStringPtr(record.SubvolumeID),
		TargetName:  nullStringPtr(record.TargetName),
	}

	return ctx.JSON(http.StatusOK, detail)
}

func (s *Server) GetConfig(ctx echo.Context) error {
	cfg, err := config.GetConfig(s.DB)
	if err != nil {
		s.logger.Error("failed to get config", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "failed to read configuration",
		})
	}
	return ctx.JSON(http.StatusOK, cfg)
}

func (s *Server) UpdateConfig(ctx echo.Context) error {
	var req gen.GlobalConfig
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.ErrorResponse{
			Error: "invalid request body",
		})
	}

	options := make(map[string]string)
	if req.BackupStorage.Options != nil {
		options = *req.BackupStorage.Options
	}

	if accessKey := options["access_key"]; accessKey != "" {
		if err := config.WriteS3AccessKey(accessKey); err != nil {
			s.logger.Error("failed to write s3 access key", zap.Error(err))
			return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
				Error: "failed to store s3 access key",
			})
		}
		delete(options, "access_key")
	}
	if secretKey := options["secret_key"]; secretKey != "" {
		if err := config.WriteS3SecretKey(secretKey); err != nil {
			s.logger.Error("failed to write s3 secret key", zap.Error(err))
			return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
				Error: "failed to store s3 secret key",
			})
		}
		delete(options, "secret_key")
	}

	if req.Encryption.Passphrase != nil && *req.Encryption.Passphrase != "" {
		if err := config.WriteE2EEKey(*req.Encryption.Passphrase); err != nil {
			s.logger.Error("failed to write e2ee key", zap.Error(err))
			return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
				Error: "failed to store e2ee key",
			})
		}
	}

	internal := config.GlobalConfig{
		Backup:   toVolumeJobSchedule(req.Backup),
		Snapshot: toVolumeJobSchedule(req.Snapshot),
		Defrag:   toVolumeJobSchedule(req.Defrag),
		Scrub:    toDiskJobSchedule(req.Scrub),
		Balance:  toDiskJobSchedule(req.Balance),
		BackupStorage: config.BackupStorageConfig{
			Type:    req.BackupStorage.Type,
			Options: options,
		},
		Encryption: config.EncryptionConfig{
			Active: config.HasE2EEKey(),
		},
	}

	if err := config.SaveConfig(s.DB, internal); err != nil {
		s.logger.Error("failed to save config", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "failed to save configuration",
		})
	}

	s.logger.Info("configuration updated")
	return ctx.JSON(http.StatusOK, internal)
}

func (s *Server) ListDisks(ctx echo.Context) error {
	s.logger.Debug("received list disks request")

	result := s.emitEvent(event.ActionDiskList, event.DiskListRequest{})

	if result.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	disks, ok := result.Data.([]event.DiskInfo)
	if !ok {
		s.logger.Error("unexpected response type from disk list handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusOK, disks)
}

func (s *Server) ListFilesystems(ctx echo.Context) error {
	s.logger.Debug("received list filesystems request")

	result := s.emitEvent(event.ActionFilesystemList, event.FilesystemListRequest{})

	if result.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	filesystems, ok := result.Data.([]event.FilesystemInfo)
	if !ok {
		s.logger.Error("unexpected response type from filesystem list handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusOK, filesystems)
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

	result := s.emitEvent(event.ActionFilesystemCreate, eventReq)

	if result.Error != nil {
		return ctx.JSON(http.StatusBadRequest, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	fs, ok := result.Data.(event.FilesystemInfo)
	if !ok {
		s.logger.Error("unexpected response type from filesystem create handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusCreated, fs)
}

func (s *Server) ListMounts(ctx echo.Context) error {
	s.logger.Debug("received list mounts request")

	result := s.emitEvent(event.ActionMountList, event.MountListRequest{})

	if result.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	mounts, ok := result.Data.([]event.MountInfo)
	if !ok {
		s.logger.Error("unexpected response type from mount list handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusOK, mounts)
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

	result := s.emitEvent(event.ActionMountCreate, eventReq)

	if result.Error != nil {
		return ctx.JSON(http.StatusBadRequest, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	mount, ok := result.Data.(event.MountInfo)
	if !ok {
		s.logger.Error("unexpected response type from mount create handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusCreated, mount)
}

func (s *Server) ListSubvolumes(ctx echo.Context) error {
	s.logger.Debug("received list subvolumes request")

	result := s.emitEvent(event.ActionSubvolumeList, event.SubvolumeListRequest{})

	if result.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	subvolumes, ok := result.Data.([]event.SubvolumeInfo)
	if !ok {
		s.logger.Error("unexpected response type from subvolume list handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusOK, subvolumes)
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

	var defragFreq string
	if req.Defrag.Frequency != nil {
		defragFreq = string(*req.Defrag.Frequency)
	}

	eventReq := event.CreateSubvolumeRequest{
		Name:        req.Name,
		FsUUID:      req.FsUuid.String(),
		Compression: req.Compression,
		Defrag: event.DefragConfig{
			Enabled:   req.Defrag.Enabled,
			Frequency: defragFreq,
		},
		NFS: req.Nfs,
		SMB: req.Smb,
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

	result := s.emitEvent(event.ActionSubvolumeCreate, eventReq)

	if result.Error != nil {
		return ctx.JSON(http.StatusBadRequest, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	subvol, ok := result.Data.(event.SubvolumeInfo)
	if !ok {
		s.logger.Error("unexpected response type from subvolume create handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusCreated, subvol)
}

func (s *Server) GetSubvolume(ctx echo.Context, id openapi_types.UUID) error {
	s.logger.Debug("received get subvolume request", zap.String("id", id.String()))

	eventReq := event.SubvolumeGetRequest{
		ID: id.String(),
	}

	result := s.emitEvent(event.ActionSubvolumeGet, eventReq)

	if result.Error != nil {
		return ctx.JSON(http.StatusNotFound, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	subvol, ok := result.Data.(event.SubvolumeInfo)
	if !ok {
		s.logger.Error("unexpected response type from subvolume get handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusOK, subvol)
}

func (s *Server) DeleteSubvolume(ctx echo.Context, id openapi_types.UUID) error {
	s.logger.Debug("received delete subvolume request", zap.String("id", id.String()))

	eventReq := event.SubvolumeDeleteRequest{
		ID: id.String(),
	}

	result := s.emitEvent(event.ActionSubvolumeDelete, eventReq)

	if result.Error != nil {
		return ctx.JSON(http.StatusNotFound, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (s *Server) ListSubvolumeSnapshots(ctx echo.Context, id openapi_types.UUID) error {
	s.logger.Debug("received list subvolume snapshots request", zap.String("id", id.String()))

	eventReq := event.SnapshotListRequest{
		SubvolumeID: id.String(),
	}

	result := s.emitEvent(event.ActionSnapshotList, eventReq)

	if result.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: result.Error.Error(),
		})
	}

	snapshots, ok := result.Data.([]event.SnapshotInfo)
	if !ok {
		s.logger.Error("unexpected response type from snapshot list handler")
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: "internal error",
		})
	}

	return ctx.JSON(http.StatusOK, snapshots)
}
