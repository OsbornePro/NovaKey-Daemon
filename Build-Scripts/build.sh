#!/bin/bash
# =============================================================================
# NovaKey - Unified cross-platform build script (Linux & macOS)
# Contact: security@novakey.app
# Author: Robert H. Osborne (OsbornePro)
# Date: December 2025
# =============================================================================

set -Eeo pipefail
shopt -s nocasematch

# ----------------------------- Colors -----------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log()    { printf "${CYAN}[-] %s ${NC}%s\n" "$(date '+%m-%d-%Y %H:%M:%S')" "$1"; }
warn()   { printf "${YELLOW}[!] %s${NC}\n" "$1"; }
success(){ printf "${GREEN}[✓] %s${NC}\n" "$1"; }
error()  { printf "${RED}[x] %s${NC}\n" "$1" >&2; exit 1; }

# ----------------------------- Host OS -----------------------------
HOST_OS="$(uname | tr '[:upper:]' '[:lower:]')"
case "$HOST_OS" in
    linux*)  HOST_OS="linux" ;;
    darwin*) HOST_OS="darwin" ;;
    *) error "Unsupported host OS: $HOST_OS" ;;
esac

# ----------------------------- Args -----------------------------
TARGET="linux"
CLEAN=false
FILENAME=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        -t|--target) TARGET="$2"; shift 2 ;;
        -c|--clean)  CLEAN=true; shift ;;
        -f|--file)   FILENAME="$2"; shift 2 ;;
        -h|--help)
            echo "Usage: ./build.sh -t windows|linux|darwin [-c]"
            exit 0
            ;;
        *) error "Unknown option: $1" ;;
    esac
done

# ----------------------------- Project Root -----------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# ----------------------------- Version -----------------------------
VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "dev")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE}"

log "Building NovaKey $VERSION for target=$TARGET (host=$HOST_OS)"

# ----------------------------- Clean -----------------------------
$CLEAN && rm -rf dist
mkdir -p dist

# ----------------------------- Build -----------------------------
case "$TARGET" in

    windows)
        CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
            go build -trimpath -ldflags="$LDFLAGS" -o "dist/${FILENAME:-NovaKey.exe}" ./cmd/novakey
        ;;

    linux)
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
            go build -trimpath -ldflags="$LDFLAGS" -o "dist/${FILENAME:-NovaKey}" ./cmd/novakey
        ;;

    darwin)
        if [[ "$HOST_OS" != "darwin" ]]; then
            warn "macOS builds must be performed on macOS."
            warn "Reason: CGO + Cocoa APIs cannot be cross-compiled from $HOST_OS."
            warn "Run this script on a Mac with Xcode installed."
            exit 0
        fi

        for ARCH in amd64 arm64; do
            log "Building darwin/$ARCH"
            CGO_ENABLED=1 GOOS=darwin GOARCH="$ARCH" \
                go build -trimpath -ldflags="$LDFLAGS" -o "dist/NovaKey-darwin-$ARCH" ./cmd/novakey
        done

        log "To create universal binary:"
        log "lipo -create -output NovaKey NovaKey-darwin-amd64 NovaKey-darwin-arm64"
        ;;
    *)
        error "Invalid target: $TARGET"
        ;;
esac

success "Build artifacts created in dist/"
