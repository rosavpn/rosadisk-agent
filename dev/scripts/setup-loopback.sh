#!/bin/bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$SCRIPT_DIR/../test_disks"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Create first disk
dd if=/dev/zero of=virtual_disk1.img bs=1M count=5000

# Copy for second disk (faster than dd)
cp virtual_disk1.img virtual_disk2.img

sudo losetup -fP virtual_disk1.img
sudo losetup -fP virtual_disk2.img
sudo losetup -a
