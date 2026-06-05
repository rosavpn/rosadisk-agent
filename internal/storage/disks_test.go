package storage

import (
	"testing"
)

func TestListDisks(t *testing.T) {
	disks, err := ListDisks()
	if err != nil {
		t.Fatalf("ListDisks() error = %v", err)
	}

	if len(disks) == 0 {
		t.Error("ListDisks() returned no disks")
	}

	for i, disk := range disks {
		if disk.Name == "" {
			t.Errorf("disk[%d].Name is empty", i)
		}
		if disk.Size == 0 {
			t.Errorf("disk[%d].Size is 0", i)
		}
		if disk.Type == "" {
			t.Errorf("disk[%d].Type is empty", i)
		}
	}
}
