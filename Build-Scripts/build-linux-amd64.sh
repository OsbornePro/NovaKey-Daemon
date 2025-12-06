#!/bin/bash
# Build NovaKey for Linux AMD64 only

set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "dev")
LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildDate=$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

echo "Building NovaKey ${VERSION} for Linux AMD64"

rm -rf dist
mkdir -p dist

# CGO enabled for robotgo
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="${LDFLAGS}" -o dist/novakey-linux-amd64 ./cmd/novakey

echo "Build complete: ./dist/novakey-linux-amd64"
ls -lh dist/

