package storage

type DiskInfo struct {
	Name       string  `json:"name"`
	Size       uint64  `json:"size"`
	Type       string  `json:"type"`
	FSType     *string `json:"fstype"`
	MountPoint *string `json:"mountpoint"`
}
