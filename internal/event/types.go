package event

type DiskListRequest struct{}

type DiskInfo struct {
	Name       string  `json:"name"`
	Size       uint64  `json:"size"`
	Type       string  `json:"type"`
	FSType     *string `json:"fstype"`
	MountPoint *string `json:"mountpoint"`
	Vendor     *string `json:"vendor"`
	Model      *string `json:"model"`
}

type DiskListResponse struct {
	Disks []DiskInfo `json:"disks"`
}
