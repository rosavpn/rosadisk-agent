package storage

import (
	"fmt"
	"os/exec"
	"strings"
)

func DefragmentBtrfs(path string) (string, error) {
	cmd := exec.Command("btrfs", "filesystem", "defragment", "-r", path)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		return output, fmt.Errorf("defrag failed: %w, output: %s", err, output)
	}
	return output, nil
}
