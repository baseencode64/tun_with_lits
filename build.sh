#!/bin/bash
# Build script for GoXRay VPN Client
# Compiles for Debian 13 (amd64) with semantic versioning

set -e

VERSION=${1:-"1.3.0"}
OUTPUT="goxray_v${VERSION}_linux_amd64"

echo "Building GoXRay v${VERSION} for Linux amd64..."
echo "Output: ${OUTPUT}"

# Set environment variables for cross-compilation
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

# Build with optimizations
go build -ldflags="-s -w" -o "${OUTPUT}" .

echo "Build completed successfully!"
echo "Binary size: $(du -h "${OUTPUT}" | cut -f1)"
echo ""
echo "To deploy on Debian 13:"
echo "  1. Copy ${OUTPUT} to target system"
echo "  2. chmod +x ${OUTPUT}"
echo "  3. sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip ${OUTPUT}"
echo "  4. Run: sudo ./${OUTPUT} --config goxray.yaml"
