package storage

import "time"

type DiskInfo struct {
	Name   string  `json:"name"`
	Size   uint64  `json:"size"`
	Type   string  `json:"type"`
	Vendor *string `json:"vendor"`
	Model  *string `json:"model"`
	FsType *string `json:"fstype"`
}

type QuotaConfig struct {
	Enabled bool
	Limit   *int64
}

type SnapshotConfig struct {
	Enabled   bool
	Frequency string
	Retention int
}

type BackupSchedule struct {
	Enabled   bool
	Frequency string
}

type BackupConfig struct {
	Incremental *BackupSchedule
	Full        *BackupSchedule
}

type SubvolumeInfo struct {
	ID        string
	Name      string
	FsUUID    string
	Path      string
	Compression bool
	Quota     QuotaConfig
	Snapshots SnapshotConfig
	Backups   BackupConfig
	Defrag    bool
	NFS       bool
	SMB       bool
	CreatedAt time.Time
}
