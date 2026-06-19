package storage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func CreateSnapshotBtrfs(mountpoint, subvolPath, subvolName, subvolID, freq string) (string, error) {
	if !strings.HasPrefix(mountpoint, "/mnt/rosadisk/") {
		return "", fmt.Errorf("invalid mountpoint: %s", mountpoint)
	}

	snapshotDir := filepath.Join(mountpoint, ".rosa-disk", "snapshots", subvolName+"-"+subvolID)
	if err := os.MkdirAll(snapshotDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	now := time.Now()
	snapshotName := fmt.Sprintf("snapshot-%s-%s-%s", freq, now.Format("02012006"), now.Format("150405"))
	snapshotPath := filepath.Join(snapshotDir, snapshotName)

	// #nosec G204 - subvolPath is from database (validated at creation), snapshotPath is constructed
	cmd := exec.Command("btrfs", "subvolume", "snapshot", "-r", subvolPath, snapshotPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create snapshot: %w, output: %s", err, string(output))
	}

	return snapshotPath, nil
}

func DeleteSnapshotBtrfs(path string) error {
	if !strings.HasPrefix(path, "/mnt/rosadisk/") {
		return fmt.Errorf("invalid snapshot path: %s", path)
	}

	// #nosec G204 - path is from database
	cmd := exec.Command("btrfs", "subvolume", "delete", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %w, output: %s", err, string(output))
	}

	return nil
}
