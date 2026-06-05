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
	Name       string      `json:"name"`
	Size       json.Number `json:"size"`
	Type       string      `json:"type"`
	FSType     string      `json:"fstype"`
	MountPoint string      `json:"mountpoint"`
}

func ListDisks() ([]DiskInfo, error) {
	cmd := exec.Command("lsblk", "-J", "-b", "-o", "NAME,SIZE,TYPE,FSTYPE,MOUNTPOINT")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute lsblk: %w", err)
	}

	var lsblkOut lsblkOutput
	if err := json.Unmarshal(output, &lsblkOut); err != nil {
		return nil, fmt.Errorf("failed to parse lsblk output: %w", err)
	}

	disks := make([]DiskInfo, 0, len(lsblkOut.BlockDevices))
	for _, dev := range lsblkOut.BlockDevices {
		size, err := dev.Size.Int64()
		if err != nil {
			return nil, fmt.Errorf("failed to parse size for %s: %w", dev.Name, err)
		}

		disk := DiskInfo{
			Name: dev.Name,
			Size: uint64(size),
			Type: dev.Type,
		}

		if dev.FSType != "" {
			disk.FSType = &dev.FSType
		}

		if dev.MountPoint != "" {
			disk.MountPoint = &dev.MountPoint
		}

		disks = append(disks, disk)
	}

	return disks, nil
}
