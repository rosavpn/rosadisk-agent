package storage

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	devicePathRegex = regexp.MustCompile(`^/dev/(sd[a-z]+|nvme[0-9]+n[0-9]+(p[0-9]+)?|vd[a-z]+(p[0-9]+)?|loop[0-9]+)$`)
	labelRegex      = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*$`)
)

func validateDevicePath(device string) error {
	if !devicePathRegex.MatchString(device) {
		return fmt.Errorf("invalid device path: %s", device)
	}
	return nil
}

func validateLabel(label string) error {
	if len(label) == 0 {
		return fmt.Errorf("label is required")
	}
	if len(label) > 255 {
		return fmt.Errorf("label must be at most 255 characters")
	}
	if !labelRegex.MatchString(label) {
		return fmt.Errorf("label must start with alphanumeric and contain only alphanumeric and dash characters")
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

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "Label:") {
			if currentFS != nil {
				filesystems = append(filesystems, *currentFS)
			}

			currentFS = &FilesystemInfo{
				Devices: make([]string, 0),
			}
			deviceSizes = make([]uint64, 0)

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
		filesystems = append(filesystems, *currentFS)
	}

	for i := range filesystems {
		if len(filesystems[i].Devices) > 0 {
			raidProfile, err := detectRaidProfile(filesystems[i].Devices[0])
			if err == nil {
				filesystems[i].RaidProfile = raidProfile
				filesystems[i].Size = calculateSize(deviceSizes, raidProfile)
			} else {
				filesystems[i].RaidProfile = "unknown"
				filesystems[i].Size = calculateSize(deviceSizes, "unknown")
			}
		}
	}

	return filesystems, nil
}

func detectRaidProfile(device string) (string, error) {
	// #nosec G204 - device path is validated by validateDevicePath()
	cmd := exec.Command("btrfs", "inspect-internal", "dump-tree", "-t", "chunk", device)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "type") && strings.Contains(line, "|") {
			if strings.Contains(line, "RAID0") {
				return "raid0", nil
			}
			if strings.Contains(line, "RAID1") {
				return "raid1", nil
			}
		}
	}

	return "single", nil
}

func calculateSize(deviceSizes []uint64, raidProfile string) uint64 {
	if len(deviceSizes) == 0 {
		return 0
	}

	switch raidProfile {
	case "raid1":
		minSize := deviceSizes[0]
		for _, size := range deviceSizes[1:] {
			if size < minSize {
				minSize = size
			}
		}
		return minSize
	case "raid0":
		var total uint64
		for _, size := range deviceSizes {
			total += size
		}
		return total
	default:
		return deviceSizes[0]
	}
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

	if err := validateLabel(label); err != nil {
		return nil, err
	}

	if raidProfile != "single" && raidProfile != "raid0" && raidProfile != "raid1" {
		return nil, fmt.Errorf("invalid raid profile: %s", raidProfile)
	}

	if raidProfile == "single" && len(devices) > 1 {
		return nil, fmt.Errorf("single profile requires exactly one device")
	}
	if (raidProfile == "raid0" || raidProfile == "raid1") && len(devices) < 2 {
		return nil, fmt.Errorf("%s profile requires at least two devices", raidProfile)
	}

	args := []string{"mkfs.btrfs"}
	if raidProfile == "raid0" || raidProfile == "raid1" {
		args = append(args, "-d", raidProfile, "-m", raidProfile)
	}
	args = append(args, "-L", label)
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
		Label:       &label,
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
