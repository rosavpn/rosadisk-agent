#!/bin/bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$SCRIPT_DIR/../test_disks"
cd "$TEST_DIR"

sudo losetup -D
rm -f virtual_disk*.img
