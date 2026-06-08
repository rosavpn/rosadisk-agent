package storage

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var devicePathRegex = regexp.MustCompile(`^/dev/(sd[a-z]+|nvme[0-9]+n[0-9]+(p[0-9]+)?|vd[a-z]+(p[0-9]+)?|loop[0-9]+)$`)

func validateDevicePath(device string) error {
	if !devicePathRegex.MatchString(device) {
		return fmt.Errorf("invalid device path: %s", device)
	}
	return nil
}

type FilesystemInfo struct {
	UUID        string
	Label       *string
	Size        uint64
	Devices     []string
	RaidProfile string
}

func ListFilesystems() ([]FilesystemInfo, error) {
	cmd := exec.Command("btrfs", "filesystem", "show", "--raw")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []FilesystemInfo{}, nil
		}
		return nil, fmt.Errorf("failed to execute btrfs filesystem show: %w", err)
	}

	filesystems := make([]FilesystemInfo, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	var currentFS *FilesystemInfo
	var deviceSizes []uint64
	var deviceCount int

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "Label:") {
			if currentFS != nil {
				currentFS.Size = calculateSize(deviceSizes, deviceCount)
				currentFS.RaidProfile = determineRaidProfile(deviceCount, currentFS.Label)
				filesystems = append(filesystems, *currentFS)
			}

			currentFS = &FilesystemInfo{
				Devices: make([]string, 0),
			}
			deviceSizes = make([]uint64, 0)
			deviceCount = 0

			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "Label:" && i+1 < len(parts) {
					label := parts[i+1]
					if label != "none" {
						currentFS.Label = &label
					}
				}
				if part == "uuid:" && i+1 < len(parts) {
					currentFS.UUID = strings.TrimSuffix(parts[i+1], "")
				}
			}
		} else if strings.Contains(line, "Total devices") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "devices" && i+1 < len(parts) {
					count, err := strconv.Atoi(parts[i+1])
					if err == nil {
						deviceCount = count
					}
				}
			}
		} else if strings.Contains(line, "devid") && strings.Contains(line, "path") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "size" && i+1 < len(parts) {
					size, err := strconv.ParseUint(parts[i+1], 10, 64)
					if err == nil {
						deviceSizes = append(deviceSizes, size)
					}
				}
				if part == "path" && i+1 < len(parts) {
					currentFS.Devices = append(currentFS.Devices, parts[i+1])
				}
			}
		}
	}

	if currentFS != nil {
		currentFS.Size = calculateSize(deviceSizes, deviceCount)
		currentFS.RaidProfile = determineRaidProfile(deviceCount, currentFS.Label)
		filesystems = append(filesystems, *currentFS)
	}

	return filesystems, nil
}

func calculateSize(deviceSizes []uint64, deviceCount int) uint64 {
	if len(deviceSizes) == 0 {
		return 0
	}

	if deviceCount == 1 {
		return deviceSizes[0]
	}

	minSize := deviceSizes[0]
	for _, size := range deviceSizes[1:] {
		if size < minSize {
			minSize = size
		}
	}

	return minSize
}

func determineRaidProfile(deviceCount int, label *string) string {
	if deviceCount == 1 {
		return "single"
	} else if deviceCount == 2 {
		return "raid1"
	}
	return "unknown"
}

func CreateFilesystem(devices []string, label string, raidProfile string) (*FilesystemInfo, error) {
	if len(devices) == 0 {
		return nil, fmt.Errorf("at least one device is required")
	}

	for _, device := range devices {
		if err := validateDevicePath(device); err != nil {
			return nil, err
		}
	}

	if raidProfile == "" {
		raidProfile = "single"
	}

	args := []string{"mkfs.btrfs", "-d", raidProfile}
	if label != "" {
		args = append(args, "-L", label)
	}
	args = append(args, devices...)

	// #nosec G204 - device paths are validated by validateDevicePath()
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create btrfs filesystem: %w, output: %s", err, string(output))
	}

	fs := &FilesystemInfo{
		Devices:     devices,
		RaidProfile: raidProfile,
		Label:       nil,
	}

	if label != "" {
		fs.Label = &label
	}

	// #nosec G204 - device path is validated by validateDevicePath()
	cmd = exec.Command("btrfs", "filesystem", "show", "--raw", devices[0])
	output, err = cmd.Output()
	if err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "uuid:") {
				parts := strings.Fields(line)
				for i, part := range parts {
					if part == "uuid:" && i+1 < len(parts) {
						fs.UUID = part
						break
					}
				}
				break
			}
		}
	}

	return fs, nil
}
