package storage

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type lsblkOutput struct {
	BlockDevices []lsblkDevice `json:"blockdevices"`
}

type lsblkDevice struct {
	Name     string      `json:"name"`
	Size     json.Number `json:"size"`
	Type     string      `json:"type"`
	Vendor   string      `json:"vendor"`
	Model    string      `json:"model"`
	Children []lsblkDevice `json:"children,omitempty"`
}

func ListDisks() ([]DiskInfo, error) {
	cmd := exec.Command("lsblk", "-J", "-b", "-o", "NAME,SIZE,TYPE,VENDOR,MODEL")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute lsblk: %w", err)
	}

	var lsblkOut lsblkOutput
	if err := json.Unmarshal(output, &lsblkOut); err != nil {
		return nil, fmt.Errorf("failed to parse lsblk output: %w", err)
	}

	disks := make([]DiskInfo, 0)
	for _, dev := range lsblkOut.BlockDevices {
		disks = append(disks, parseDevice(dev, nil, nil)...)
	}

	return disks, nil
}

func parseDevice(dev lsblkDevice, parentVendor, parentModel *string) []DiskInfo {
	var result []DiskInfo

	// Only include devices with type "disk"
	if dev.Type != "disk" {
		// Still process children to find nested disks
		for _, child := range dev.Children {
			result = append(result, parseDevice(child, parentVendor, parentModel)...)
		}
		return result
	}

	size, err := dev.Size.Int64()
	if err != nil {
		return nil
	}

	if size < 0 {
		return nil
	}

	// Inherit vendor/model from parent if current device has null values
	vendor := dev.Vendor
	model := dev.Model

	var vendorPtr, modelPtr *string
	if vendor != "" {
		vendorPtr = &vendor
	} else if parentVendor != nil {
		vendorPtr = parentVendor
	}

	if model != "" {
		modelPtr = &model
	} else if parentModel != nil {
		modelPtr = parentModel
	}

	disk := DiskInfo{
		Name:   dev.Name,
		Size:   uint64(size),
		Type:   dev.Type,
		Vendor: vendorPtr,
		Model:  modelPtr,
	}

	result = append(result, disk)

	return result
}
