package storage

import (
	"fmt"
	"os/exec"
	"strings"
)

func validateMountpoint(mountpoint string) error {
	if !strings.HasPrefix(mountpoint, "/mnt/rosadisk/") {
		return fmt.Errorf("invalid mountpoint: %s", mountpoint)
	}
	return nil
}

func StartScrub(mountpoint string) (string, error) {
	if err := validateMountpoint(mountpoint); err != nil {
		return "", err
	}

	// #nosec G204 - mountpoint is validated by validateMountpoint()
	cmd := exec.Command("btrfs", "scrub", "start", "-B", "-d", mountpoint)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("scrub failed on %s: %w, output: %s", mountpoint, err, string(output))
	}

	return string(output), nil
}

func StartBalance(mountpoint string) (string, error) {
	if err := validateMountpoint(mountpoint); err != nil {
		return "", err
	}

	// #nosec G204 - mountpoint is validated by validateMountpoint()
	cmd := exec.Command("btrfs", "balance", "start", "-dusage=50", "-musage=50", mountpoint)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("balance failed on %s: %w, output: %s", mountpoint, err, string(output))
	}

	return string(output), nil
}
