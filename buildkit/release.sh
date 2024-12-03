#!/bin/bash

# Usage: ./release.sh <platform>
# Example: ./release.sh linux/amd64

set -e

# Default platform
platform="${1:-linux/amd64}"

# Extract the full buildkit version (e.g., "harness/buildkit:1.0.6") without jq or grep -P
full_buildkit_version=$(grep '"buildkit_version"' buildkit/version.json | awk -F'"' '{print $4}')

# Extract only the version number (e.g., "1.0.6")
buildkit_version=${full_buildkit_version##*:}

# Parse platform components
os=${platform%%/*}
arch=${platform##*/}

# Construct GCS tarball URL
tarball_url="https://storage.cloud.google.com/harness-ti/buildkit/${buildkit_version}/harness-buildkit-${buildkit_version}-${os}-${arch}.tar"

# Download the tarball
echo "Downloading Buildkit tarball from ${tarball_url}..."
mkdir -p buildkit
wget -O buildkit/buildkit.tar "${tarball_url}"

echo "Buildkit tarball downloaded to buildkit/buildkit.tar"
