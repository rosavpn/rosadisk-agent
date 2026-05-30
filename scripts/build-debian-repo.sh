#!/bin/bash
set -euo pipefail

# Build Debian Repository Structure
# Usage: ./scripts/build-debian-repo.sh <deb-packages-dir> <output-dir> [gpg-key-id]

DEB_PACKAGES_DIR="${1:-.}"
OUTPUT_DIR="${2:-debian-repo}"
GPG_KEY_ID="${3:-}"

echo "Building Debian repository structure..."
echo "  Input:  ${DEB_PACKAGES_DIR}"
echo "  Output: ${OUTPUT_DIR}"

# Clean output directory
rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}/dists/trixie/main/binary-amd64"
mkdir -p "${OUTPUT_DIR}/dists/trixie/main/binary-arm64"
mkdir -p "${OUTPUT_DIR}/pool/main"

# Copy .deb packages to pool
for deb in "${DEB_PACKAGES_DIR}"/*.deb; do
    if [ -f "$deb" ]; then
        cp "$deb" "${OUTPUT_DIR}/pool/main/"
        echo "Copied: $(basename "$deb")"
    fi
done

# Change to output directory for dpkg-scanpackages
cd "${OUTPUT_DIR}"

# Process each architecture
for arch in amd64 arm64; do
    echo "Processing ${arch}..."

    ARCH_DIR="dists/trixie/main/binary-${arch}"

    # Create Packages file
    > "${ARCH_DIR}/Packages"

    for deb in pool/main/*_${arch}.deb; do
        if [ -f "$deb" ]; then
            echo "  Adding: $(basename "$deb")"
            dpkg-scanpackages --arch "${arch}" pool/main > "${ARCH_DIR}/Packages"
            break
        fi
    done

    # Compress Packages
    gzip -c "${ARCH_DIR}/Packages" > "${ARCH_DIR}/Packages.gz"

    # Create Release file for this architecture
    cat > "${ARCH_DIR}/Release" <<EOF
Archive: trixie
Origin: rosadisk-agent
Label: rosadisk-agent
Codename: trixie
Architectures: ${arch}
Components: main
Description: Rosadisk Agent Debian Repository
EOF
done

# Create main Release file
cat > "dists/trixie/Release" <<EOF
Origin: rosadisk-agent
Label: rosadisk-agent
Suite: trixie
Codename: trixie
Architectures: amd64 arm64
Components: main
Description: Rosadisk Agent Debian Repository
Date: $(date -Ru)
EOF

# Collect checksums
MD5_LINES=""
SHA1_LINES=""
SHA256_LINES=""

for arch in amd64 arm64; do
    PKG_FILE="dists/trixie/main/binary-${arch}/Packages"
    if [ -f "$PKG_FILE" ]; then
        MD5=$(md5sum "$PKG_FILE" | awk '{print $1}')
        SHA1=$(sha1sum "$PKG_FILE" | awk '{print $1}')
        SHA256=$(sha256sum "$PKG_FILE" | awk '{print $1}')
        SIZE=$(stat -c%s "$PKG_FILE")
        
        MD5_LINES="${MD5_LINES} ${MD5} ${SIZE} main/binary-${arch}/Packages\n"
        SHA1_LINES="${SHA1_LINES} ${SHA1} ${SIZE} main/binary-${arch}/Packages\n"
        SHA256_LINES="${SHA256_LINES} ${SHA256} ${SIZE} main/binary-${arch}/Packages\n"
    fi
done

# Append checksums in proper format
printf "MD5Sum:\n%s" "${MD5_LINES}" >> "dists/trixie/Release"
printf "SHA1:\n%s" "${SHA1_LINES}" >> "dists/trixie/Release"
printf "SHA256:\n%s" "${SHA256_LINES}" >> "dists/trixie/Release"

# Sign the Release file if GPG key is provided
if [ -n "$GPG_KEY_ID" ]; then
    echo "Signing repository with GPG key: ${GPG_KEY_ID}"
    gpg --default-key "$GPG_KEY_ID" --armor --detach-sign --output "dists/trixie/Release.gpg" "dists/trixie/Release"
    gpg --default-key "$GPG_KEY_ID" --armor --clearsign --output "dists/trixie/InRelease" "dists/trixie/Release"
    echo "Repository signed successfully"
else
    echo "No GPG key provided, skipping signing"
fi

cd - > /dev/null

echo "Debian repository built successfully at: ${OUTPUT_DIR}"
echo ""
echo "Directory structure:"
find "${OUTPUT_DIR}" -type f | sort
