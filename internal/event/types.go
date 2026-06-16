package event

type DiskListRequest struct{}

type DiskInfo struct {
	Name   string  `json:"name"`
	Size   uint64  `json:"size"`
	Type   string  `json:"type"`
	Vendor *string `json:"vendor"`
	Model  *string `json:"model"`
	FsType *string `json:"fstype"`
}

type FilesystemListRequest struct{}

type FilesystemInfo struct {
	UUID        string   `json:"uuid"`
	Label       *string  `json:"label"`
	Size        uint64   `json:"size"`
	Devices     []string `json:"devices"`
	RaidProfile string   `json:"raid_profile"`
}

type CreateFilesystemRequest struct {
	Devices     []string `json:"devices"`
	Label       string   `json:"label"`
	RaidProfile string   `json:"raid_profile"`
}

type MountListRequest struct{}

type MountInfo struct {
	UUID       string   `json:"uuid"`
	Label      string   `json:"label"`
	Mountpoint string   `json:"mountpoint"`
	Devices    []string `json:"devices"`
	Used       uint64   `json:"used"`
}

type MountRequest struct {
	UUID string `json:"uuid"`
}

type SubvolumeListRequest struct{}

type SubvolumeInfo struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	FsUUID      string         `json:"fs_uuid"`
	Path        string         `json:"path"`
	Compression bool           `json:"compression"`
	Quota       QuotaConfig    `json:"quota"`
	Snapshots   SnapshotConfig `json:"snapshots"`
	Backups     BackupConfig   `json:"backups"`
	Defrag      bool           `json:"defrag"`
	NFS         bool           `json:"nfs"`
	SMB         bool           `json:"smb"`
	CreatedAt   string         `json:"created_at"`
}

type QuotaConfig struct {
	Enabled bool  `json:"enabled"`
	Limit   int64 `json:"limit,omitempty"`
}

type SnapshotConfig struct {
	Enabled   bool   `json:"enabled"`
	Frequency string `json:"frequency,omitempty"`
	Retention int    `json:"retention,omitempty"`
}

type BackupSchedule struct {
	Enabled   bool   `json:"enabled"`
	Frequency string `json:"frequency,omitempty"`
}

type BackupConfig struct {
	Incremental BackupSchedule `json:"incremental"`
	Full        BackupSchedule `json:"full"`
}

type CreateSubvolumeRequest struct {
	Name        string
	FsUUID      string
	Compression bool
	Defrag      bool
	NFS         bool
	SMB         bool
	Quota       QuotaConfig
	Snapshots   SnapshotConfig
	Backups     BackupConfig
}

type SubvolumeGetRequest struct {
	ID string
}

type SubvolumeDeleteRequest struct {
	ID string
}

type BackupRequest struct{}

type SnapshotRequest struct{}

type DefragRequest struct{}

type ScrubRequest struct{}

type BalanceRequest struct{}
