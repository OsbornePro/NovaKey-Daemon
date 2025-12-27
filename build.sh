#!/bin/bash
# =============================================================================
# NovaKey - Unified cross-platform build script (Linux host)
# nvpair - Pair device with the novakey daemon
# nvclient - Sends password to type to the NovaKey daemon
# Contact: security@novakey.app
# Author: Robert H. Osborne (OsbornePro)
# Date: December 2025
# Requirements: xdotool xclip wl-clipboard
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
success(){ printf "${GREEN}[âœ“] %s${NC}\n" "$1"; }
error()  { printf "${RED}[x] %s${NC}\n" "$1" >&2; exit 1; }

# ----------------------------- Host OS -----------------------------
HOST_OS="$(uname | tr '[:upper:]' '[:lower:]')"
case "$HOST_OS" in
    windows*) HOST_OS="windows" ;;
    linux*)   HOST_OS="linux" ;;
    darwin*)  HOST_OS="darwin" ;;
    *) error "Unsupported host OS: $HOST_OS" ;;
esac

# ----------------------------- Requirements (Linux helpers) -----------------
# Only try to install xdotool/xclip when we're actually on Linux.
if [[ "$HOST_OS" == "linux" ]]; then
    if command -v dnf >/dev/null 2>&1; then
        sudo dnf install -y xdotool xclip wl-clipboard
    elif command -v apt-get >/dev/null 2>&1; then
        sudo apt-get update && sudo apt-get install -y xdotool xclip
    else
        echo "Neither dnf nor apt is available on this Linux system (skipping xdotool/xclip install)."
    fi
fi

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
            echo "Usage: ./build.sh -t windows|linux|darwin [-c] [-f filename]"
            exit 0
            ;;
        *) error "Unknown option: $1" ;;
    esac
done

# ----------------------------- Project Root -----------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"   # <--- assume script is in repo root

# ----------------------------- Version -----------------------------
VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "dev")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# NOTE: this assumes you have in your Go code:
#   var version = "dev"
#   var buildDate = ""
LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE}"

log "Building NovaKey $VERSION for target=$TARGET (host=$HOST_OS)"

# ----------------------------- Clean -----------------------------
$CLEAN && rm -rf -- dist
mkdir -p dist

# ----------------------------- Build -----------------------------
case "$TARGET" in
    windows)
        log "Building novakey for windows/amd64"
        CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
          go build -trimpath -ldflags="$LDFLAGS -H=windowsgui" \
            -o "dist/${FILENAME:-novakey-windows-amd64.exe}" ./cmd/novakey
            
        log "Building nvpair for windows/amd64"
        CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
          go build -trimpath -ldflags="$LDFLAGS" \
            -o "dist/nvpair-windows-amd64.exe" ./cmd/nvpair
            
        log "Building nvclient for windows/amd64"
        CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
          go build -trimpath -ldflags="$LDFLAGS" \
            -o "dist/nvclient-windows-amd64.exe" ./cmd/nvclient
        ;;

    linux)
        for ARCH in amd64 arm64; do
            log "Building novakey for linux/$ARCH"
            CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" \
              go build -trimpath -ldflags="$LDFLAGS" \
                -o "dist/${FILENAME:-novakey-linux-$ARCH}" ./cmd/novakey
            
            log "Building nvpair for linux/$ARCH"
            CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" \
              go build -trimpath -ldflags="$LDFLAGS" \
                -o "dist/nvpair-linux-$ARCH" ./cmd/nvpair
                
            log "Building nvclient for linux/$ARCH"
            CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" \
              go build -trimpath -ldflags="$LDFLAGS" \
                -o "dist/nvclient-linux-$ARCH" ./cmd/nvclient
        done
        ;;

    darwin)
        if [[ "$HOST_OS" != "darwin" ]]; then
            warn "macOS builds should be performed on macOS."
            warn "Reason: future CGO + Cocoa APIs may not cross-compile cleanly."
            exit 0
        fi

        for ARCH in amd64 arm64; do
            log "Building novakey for darwin/$ARCH"
            CGO_ENABLED=0 GOOS=darwin GOARCH="$ARCH" \
              go build -trimpath -ldflags="$LDFLAGS" \
                -o "dist/${FILENAME:-novakey-darwin-$ARCH}" ./cmd/novakey
                
            log "Building nvpair for darwin/$ARCH"
            CGO_ENABLED=0 GOOS=darwin GOARCH="$ARCH" \
              go build -trimpath -ldflags="$LDFLAGS" \
                -o "dist/nvpair-darwin-$ARCH" ./cmd/nvpair
                
            log "Building nvclient for darwin/$ARCH"
            CGO_ENABLED=0 GOOS=darwin GOARCH="$ARCH" \
              go build -trimpath -ldflags="$LDFLAGS" \
                -o "dist/nvclient-darwin-$ARCH" ./cmd/nvclient
        done

        log "To create universal binary:"
        log "lipo -create -output NovaKey NovaKey-darwin-amd64 NovaKey-darwin-arm64"
        ;;

    *)
        error "Invalid target: $TARGET"
        ;;
esac

success "Build artifacts created in dist/"
