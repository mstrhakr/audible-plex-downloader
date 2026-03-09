#!/bin/bash
# Local Docker build script
# This script prepares the build context for Docker with the go-audible dependency

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_DIR="$(mktemp -d)"

echo "Preparing Docker build context in $BUILD_DIR..."

# Copy audible-plex-downloader
echo "Copying audible-plex-downloader..."
cp -r "$SCRIPT_DIR" "$BUILD_DIR/audible-plex-downloader"

# Copy or clone go-audible
if [ -d "$SCRIPT_DIR/../go-audible" ]; then
    echo "Copying local go-audible from ../go-audible..."
    cp -r "$SCRIPT_DIR/../go-audible" "$BUILD_DIR/go-audible"
else
    echo "Cloning go-audible from GitHub..."
    git clone https://github.com/mstrhakr/go-audible.git "$BUILD_DIR/go-audible"
fi

# Build the Docker image
echo "Building Docker image..."
cd "$BUILD_DIR"
docker build -f audible-plex-downloader/Dockerfile -t audible-plex-downloader:local .

echo "Cleaning up build context..."
rm -rf "$BUILD_DIR"

echo ""
echo "✅ Docker image built successfully as 'audible-plex-downloader:local'"
echo ""
echo "To run:"
echo "  docker run -d -p 8080:8080 -v ./config:/config -v ./audiobooks:/audiobooks audible-plex-downloader:local"
