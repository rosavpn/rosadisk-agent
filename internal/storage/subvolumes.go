package storage

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type CreateSubvolumeBtrfsRequest struct {
	Mountpoint  string
	Name        string
	Compression bool
	QuotaLimit  *int64
}

func validateSubvolumeName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("subvolume name is required")
	}
	if len(name) > 255 {
		return fmt.Errorf("subvolume name must be at most 255 characters")
	}
	if strings.Contains(name, "/") {
		return fmt.Errorf("subvolume name cannot contain slashes")
	}
	return nil
}

func CreateSubvolumeBtrfs(req CreateSubvolumeBtrfsRequest) (string, error) {
	subvolPath := filepath.Join(req.Mountpoint, req.Name)

	if err := validateSubvolumeName(req.Name); err != nil {
		return "", err
	}

	// #nosec G204 - subvolPath is validated by validateSubvolumeName()
	cmd := exec.Command("btrfs", "subvolume", "create", subvolPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create subvolume: %w, output: %s", err, string(output))
	}

	if req.Compression {
		// #nosec G204 - subvolPath is validated by validateSubvolumeName()
		propCmd := exec.Command("btrfs", "property", "set", subvolPath, "compression", "zstd")
		if _, err := propCmd.CombinedOutput(); err != nil {
			// #nosec G204 - subvolPath is validated by validateSubvolumeName()
			_ = exec.Command("btrfs", "subvolume", "delete", subvolPath).Run()
			return "", fmt.Errorf("failed to set compression property: %w", err)
		}
	}

	if req.QuotaLimit != nil {
		// #nosec G204 - mountpoint is from filesystem mount list
		_ = exec.Command("btrfs", "quota", "enable", req.Mountpoint).Run()

		// TODO: qgroup creation for testing
		// qgroupID := fmt.Sprintf("1/0")
		// #nosec G204 - mountpoint is from filesystem mount list
		// qgroupCmd := exec.Command("btrfs", "qgroup", "create", qgroupID, req.Mountpoint)
		// if _, err := qgroupCmd.CombinedOutput(); err != nil {
		// }

		// #nosec G204 - subvolPath is validated by validateSubvolumeName()
		limitCmd := exec.Command("btrfs", "qgroup", "limit", fmt.Sprintf("%d", *req.QuotaLimit), subvolPath)
		if output, err := limitCmd.CombinedOutput(); err != nil {
			// #nosec G204 - subvolPath is validated by validateSubvolumeName()
			_ = exec.Command("btrfs", "subvolume", "delete", subvolPath).Run()
			return "", fmt.Errorf("failed to set quota limit: %w, output: %s", err, string(output))
		}
	}

	return subvolPath, nil
}

func DeleteSubvolumeBtrfs(path string) error {
	// #nosec G204 - path is from database, validated at creation time
	cmd := exec.Command("btrfs", "subvolume", "delete", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete subvolume: %w, output: %s", err, string(output))
	}

	return nil
}

func FindMountpointByUUID(fsUUID string) (string, error) {
	mounts, err := ListMounts()
	if err != nil {
		return "", fmt.Errorf("failed to list mounts: %w", err)
	}

	for _, mount := range mounts {
		if mount.UUID == fsUUID {
			return mount.Mountpoint, nil
		}
	}

	return "", fmt.Errorf("filesystem %s is not mounted", fsUUID)
}
