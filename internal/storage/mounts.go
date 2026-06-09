package storage

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type MountInfo struct {
	UUID        string
	Label       *string
	Mountpoint  string
	Devices     []string
	RaidProfile string
	Size        uint64
}

func ListMounts() ([]MountInfo, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/mounts: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	mounts := make([]MountInfo, 0)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		device := fields[0]
		mountpoint := fields[1]
		fstype := fields[2]

		if fstype != "btrfs" {
			continue
		}

		if !strings.HasPrefix(mountpoint, "/mnt/rosadisk/") {
			continue
		}

		uuid := strings.TrimPrefix(mountpoint, "/mnt/rosadisk/")

		mount := MountInfo{
			UUID:       uuid,
			Mountpoint: mountpoint,
			Devices:    []string{device},
		}

		mounts = append(mounts, mount)
	}

	return mounts, nil
}

func MountByUUID(uuid string) (*MountInfo, error) {
	if !uuidRegex.MatchString(uuid) {
		return nil, fmt.Errorf("invalid UUID format")
	}

	device, err := findDeviceByUUID(uuid)
	if err != nil {
		return nil, err
	}

	mountpoint := filepath.Join("/mnt/rosadisk", uuid)

	if err := checkAlreadyMounted(mountpoint); err != nil {
		return nil, err
	}

	if err := ensureMountpointDir(mountpoint); err != nil {
		return nil, err
	}

	if err := executeMount(device, mountpoint); err != nil {
		return nil, err
	}

	mount := &MountInfo{
		UUID:       uuid,
		Mountpoint: mountpoint,
		Devices:    []string{device},
	}

	return mount, nil
}

func findDeviceByUUID(uuid string) (string, error) {
	cmd := exec.Command("blkid", "-U", uuid) // nolint:gosec
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("filesystem not found: %w", err)
	}

	device := strings.TrimSpace(string(output))
	if device == "" {
		return "", fmt.Errorf("filesystem not found")
	}

	return device, nil
}

func checkAlreadyMounted(mountpoint string) error {
	mounts, err := ListMounts()
	if err != nil {
		return err
	}

	for _, mount := range mounts {
		if mount.Mountpoint == mountpoint {
			return fmt.Errorf("already mounted at %s", mountpoint)
		}
	}

	return nil
}

func ensureMountpointDir(mountpoint string) error {
	if err := os.MkdirAll(mountpoint, 0750); err != nil {
		return fmt.Errorf("failed to create mountpoint directory: %w", err)
	}
	return nil
}

func executeMount(device, mountpoint string) error {
	cmd := exec.Command("mount", "-o", "defaults", device, mountpoint) // nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to mount: %w, output: %s", err, string(output))
	}
	return nil
}
