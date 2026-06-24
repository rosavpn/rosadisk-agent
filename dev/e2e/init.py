#!/usr/bin/env python3
import json
import urllib.request
import urllib.error
import sys

BASE = "http://localhost:8080"


def req(method, path, body=None):
    url = f"{BASE}{path}"
    data = json.dumps(body).encode() if body else None
    r = urllib.request.Request(url, data=data, method=method)
    r.add_header("Content-Type", "application/json") if body else None
    try:
        with urllib.request.urlopen(r) as resp:
            text = resp.read().decode()
            return resp.status, (json.loads(text) if text else None)
    except urllib.error.HTTPError as e:
        text = e.read().decode()
        return e.code, json.loads(text) if text else None


def main():
    print("=== Step 1: List disks ===")
    status, data = req("GET", "/v1/disks")
    if status != 200:
        print(f"FAILED: {data}", file=sys.stderr)
        sys.exit(1)
    print(json.dumps(data, indent=2))

    loop_devs = [f"/dev/{d['name']}" for d in data if d["type"] == "loop"]
    if len(loop_devs) < 2:
        print("Need at least 2 loop devices", file=sys.stderr)
        sys.exit(1)

    print("\n=== Step 2: Create RAID1 filesystem ===")
    status, data = req("POST", "/v1/fs", {
        "devices": loop_devs[:2],
        "label": "test-pool",
        "raid_profile": "raid1",
    })
    if status != 201:
        print(f"FAILED: {data}", file=sys.stderr)
        sys.exit(1)
    fs_uuid = data["uuid"]
    print(json.dumps(data, indent=2))

    print("\n=== Step 3: Mount filesystem ===")
    status, data = req("POST", "/v1/mounts", {"uuid": fs_uuid})
    if status != 201:
        print(f"FAILED: {data}", file=sys.stderr)
        sys.exit(1)
    print(json.dumps(data, indent=2))

    print(f"\nDone. Mounted at {data['mountpoint']}")


if __name__ == "__main__":
    main()
