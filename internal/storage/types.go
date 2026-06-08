package storage

type DiskInfo struct {
	Name   string  `json:"name"`
	Size   uint64  `json:"size"`
	Type   string  `json:"type"`
	Vendor *string `json:"vendor"`
	Model  *string `json:"model"`
	FsType *string `json:"fstype"`
}
