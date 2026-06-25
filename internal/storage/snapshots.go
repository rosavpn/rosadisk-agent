package storage

import (
	"fmt"
	"os/exec"
)

func CreateSnapshotBtrfs(source, dest string) error {
	// #nosec G204 - paths are from trusted sources
	cmd := exec.Command("btrfs", "subvolume", "snapshot", "-r", source, dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w, output: %s", err, string(output))
	}

	return nil
}
