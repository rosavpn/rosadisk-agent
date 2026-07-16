package storage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func CreateBackupSnapshot(source, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0750); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// #nosec G204 -- paths are from trusted sources
	cmd := exec.Command("btrfs", "subvolume", "snapshot", "-r", source, dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create backup snapshot: %w, output: %s", err, string(output))
	}

	return nil
}

func SendBackup(snapshotPath, destPath, keyFile, parentSnapshotPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0750); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// #nosec G304 -- destPath is from trusted subvolume/mountpoint path
	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer func() { _ = outFile.Close() }()

	// #nosec G204 -- paths are from trusted sources
	sendArgs := []string{"send"}
	if parentSnapshotPath != "" {
		sendArgs = append(sendArgs, "-p", parentSnapshotPath)
	}
	sendArgs = append(sendArgs, snapshotPath)
	sendCmd := exec.Command("btrfs", sendArgs...)
	gzipCmd := exec.Command("gzip")
	// #nosec G204 -- keyFile path is fixed /var/lib/rosadisk-agent/e2ee_key
	opensslCmd := exec.Command("openssl", "enc", "-aes-256-cbc", "-salt", "-pbkdf2",
		"-pass", fmt.Sprintf("file:%s", keyFile))

	gzipCmd.Stdin, _ = sendCmd.StdoutPipe()
	opensslCmd.Stdin, _ = gzipCmd.StdoutPipe()
	opensslCmd.Stdout = outFile

	stderr, err := opensslCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := opensslCmd.Start(); err != nil {
		return fmt.Errorf("failed to start openssl: %w", err)
	}
	if err := gzipCmd.Start(); err != nil {
		return fmt.Errorf("failed to start gzip: %w", err)
	}
	if err := sendCmd.Start(); err != nil {
		return fmt.Errorf("failed to start btrfs send: %w", err)
	}

	if err := sendCmd.Wait(); err != nil {
		return fmt.Errorf("btrfs send failed: %w", err)
	}
	if err := gzipCmd.Wait(); err != nil {
		return fmt.Errorf("gzip failed: %w", err)
	}

	errOutput := make([]byte, 4096)
	n, _ := stderr.Read(errOutput)

	if err := opensslCmd.Wait(); err != nil {
		return fmt.Errorf("openssl failed: %w, stderr: %s", err, string(errOutput[:n]))
	}

	return nil
}
