#!/bin/bash
# Build NovaKey for Linux ARM64 (requires aarch64 cross-compiler)

set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "dev")
LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildDate=$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

echo "Building NovaKey ${VERSION} for Linux ARM64"

rm -rf dist
mkdir -p dist

# Set your cross-compiler (install gcc-aarch64-linux-gnu first)
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -trimpath -ldflags="${LDFLAGS}" -o dist/novakey-linux-arm64 ./cmd/novakey

echo "Build complete: ./dist/novakey-linux-arm64"
ls -lh dist/

