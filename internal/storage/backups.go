package storage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func SendBackupBtrfs(snapshotPath, destPath string, parentPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0750); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	args := []string{"send"}
	if parentPath != "" {
		args = append(args, "-p", parentPath)
	}
	args = append(args, snapshotPath)

	// #nosec G204 -- paths are from trusted sources (snapshot DB records)
	cmd := exec.Command("btrfs", args...)

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer func() { _ = outFile.Close() }()

	cmd.Stdout = outFile
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start btrfs send: %w", err)
	}

	errOutput := make([]byte, 4096)
	n, _ := stderr.Read(errOutput)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("btrfs send failed: %w, stderr: %s", err, string(errOutput[:n]))
	}

	return nil
}
