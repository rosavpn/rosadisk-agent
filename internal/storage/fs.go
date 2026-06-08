package storage

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

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

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "Label:") {
			if currentFS != nil {
				filesystems = append(filesystems, *currentFS)
			}

			currentFS = &FilesystemInfo{
				Devices: make([]string, 0),
			}

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
				if part == "total_bytes" && i+1 < len(parts) {
					size, err := strconv.ParseUint(parts[i+1], 10, 64)
					if err == nil {
						currentFS.Size = size
					}
				}
			}
		} else if currentFS != nil && strings.HasPrefix(strings.TrimSpace(line), "/dev/") {
			device := strings.TrimSpace(line)
			currentFS.Devices = append(currentFS.Devices, device)
		}
	}

	if currentFS != nil {
		filesystems = append(filesystems, *currentFS)
	}

	for i := range filesystems {
		if len(filesystems[i].Devices) == 1 {
			filesystems[i].RaidProfile = "single"
		} else {
			filesystems[i].RaidProfile = "unknown"
		}
	}

	return filesystems, nil
}

func CreateFilesystem(devices []string, label string, raidProfile string) (*FilesystemInfo, error) {
	if len(devices) == 0 {
		return nil, fmt.Errorf("at least one device is required")
	}

	if raidProfile == "" {
		raidProfile = "single"
	}

	args := []string{"mkfs.btrfs", "-d", raidProfile}
	if label != "" {
		args = append(args, "-L", label)
	}
	args = append(args, devices...)

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
