package event

type DiskListRequest struct{}

type DiskInfo struct {
	Name   string  `json:"name"`
	Size   uint64  `json:"size"`
	Type   string  `json:"type"`
	Vendor *string `json:"vendor"`
	Model  *string `json:"model"`
}

type DiskListResponse struct {
	Disks []DiskInfo `json:"disks"`
}

type FilesystemListRequest struct{}

type FilesystemInfo struct {
	UUID        string   `json:"uuid"`
	Label       *string  `json:"label"`
	Size        uint64   `json:"size"`
	Devices     []string `json:"devices"`
	RaidProfile string   `json:"raid_profile"`
}

type FilesystemListResponse struct {
	Filesystems []FilesystemInfo `json:"filesystems"`
}

type CreateFilesystemRequest struct {
	Devices     []string `json:"devices"`
	Label       *string  `json:"label"`
	RaidProfile string   `json:"raid_profile"`
}

type CreateFilesystemResponse struct {
	Filesystem FilesystemInfo `json:"filesystem"`
}
