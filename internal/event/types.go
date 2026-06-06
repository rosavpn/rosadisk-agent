package event

type DiskListRequest struct{}

type DiskInfo struct {
	Name   string `json:"name"`
	Size   uint64 `json:"size"`
	Type   string `json:"type"`
	Vendor *string `json:"vendor"`
	Model  *string `json:"model"`
}

type DiskListResponse struct {
	Disks []DiskInfo `json:"disks"`
}
