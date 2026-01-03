#!/bin/bash
# =============================================================================
# NovaKey - Unified cross-platform build script
# (Default behavior: build binaries only)
# Optional: --package to build platform installer/package artifacts
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
  msys*|mingw*|cygwin*|windows*) HOST_OS="windows" ;;
  *) error "Unsupported host OS: $HOST_OS" ;;
esac

# ----------------------------- Requirements (Linux helpers) -----------------
# Keep your original behavior: only on Linux host.
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
PACKAGE=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    -t|--target)  TARGET="$2"; shift 2 ;;
    -c|--clean)   CLEAN=true; shift ;;
    -p|--package) PACKAGE=true; shift ;;
    -h|--help)
      cat <<EOF
Usage:
  ./build.sh -t windows|linux|darwin [-c] [--package]

Default:
  Builds binaries only into dist/<os>/...

--package:
  linux  -> builds deb/rpm via nfpm (also creates dist/linux/novakey from amd64 elf)
  darwin -> builds pkgs via installers/macos/pkg/build-pkg.sh (arm64 + amd64)
  windows-> prints instruction (installer built on Windows)
EOF
      exit 0
      ;;
    *) error "Unknown option: $1" ;;
  esac
done

# ----------------------------- Project Root -----------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# ----------------------------- Version -----------------------------
VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "dev")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE}"

log "Building NovaKey $VERSION for target=$TARGET (host=$HOST_OS)"

# ----------------------------- Clean -----------------------------
if $CLEAN; then
  rm -rf -- dist
fi

mkdir -p dist/windows dist/linux dist/macos

# ----------------------------- Build (BINARIES ONLY) -----------------------------
case "$TARGET" in
  windows)
    log "Building novakey for windows/amd64"
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
      go build -trimpath -ldflags="$LDFLAGS -H=windowsgui" \
        -o "dist/windows/novakey.exe" ./cmd/novakey

    log "Building nvpair for windows/amd64"
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
      go build -trimpath -ldflags="$LDFLAGS" \
        -o "dist/windows/nvpair-windows-amd64.exe" ./cmd/nvpair

    log "Building nvclient for windows/amd64"
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
      go build -trimpath -ldflags="$LDFLAGS" \
        -o "dist/windows/nvclient-windows-amd64.exe" ./cmd/nvclient
    ;;

  linux)
    for ARCH in amd64 arm64; do
      log "Building novakey for linux/$ARCH"
      CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" \
        go build -trimpath -ldflags="$LDFLAGS" \
          -o "dist/linux/novakey-linux-$ARCH.elf" ./cmd/novakey

      log "Building nvpair for linux/$ARCH"
      CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" \
        go build -trimpath -ldflags="$LDFLAGS" \
          -o "dist/linux/nvpair-linux-$ARCH.elf" ./cmd/nvpair

      log "Building nvclient for linux/$ARCH"
      CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" \
        go build -trimpath -ldflags="$LDFLAGS" \
          -o "dist/linux/nvclient-linux-$ARCH.elf" ./cmd/nvclient
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
          -o "dist/macos/novakey-$ARCH" ./cmd/novakey

      log "Building nvpair for darwin/$ARCH"
      CGO_ENABLED=0 GOOS=darwin GOARCH="$ARCH" \
        go build -trimpath -ldflags="$LDFLAGS" \
          -o "dist/macos/nvpair-darwin-$ARCH" ./cmd/nvpair

      log "Building nvclient for darwin/$ARCH"
      CGO_ENABLED=0 GOOS=darwin GOARCH="$ARCH" \
        go build -trimpath -ldflags="$LDFLAGS" \
          -o "dist/macos/nvclient-darwin-$ARCH" ./cmd/nvclient
    done
    ;;

  *)
    error "Invalid target: $TARGET"
    ;;
esac

success "Binary build complete (dist/)"

# ----------------------------- Package (ONLY when requested) -----------------------------
if $PACKAGE; then
  log "Packaging enabled (--package)"

  case "$TARGET" in
    linux)
      # nfpm expects dist/linux/novakey; we only create it here (NOT during normal build).
      if [[ -f "dist/linux/novakey-linux-amd64.elf" ]]; then
        cp -f "dist/linux/novakey-linux-amd64.elf" "dist/linux/novakey"
        chmod +x "dist/linux/novakey" || true
      else
        error "Missing dist/linux/novakey-linux-amd64.elf (build linux first)"
      fi

      if ! command -v nfpm >/dev/null 2>&1; then
        warn "nfpm not found. Install it:"
        warn "  go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest"
        exit 1
      fi

      ./installers/linux/nfpm/build-packages.sh "$VERSION"
      success "Linux packages built under dist/linux/"
      ;;

    darwin)
      if [[ "$HOST_OS" != "darwin" ]]; then
        error "macOS packaging must run on macOS"
      fi
      ./installers/macos/pkg/build-pkg.sh "$VERSION" arm64
      ./installers/macos/pkg/build-pkg.sh "$VERSION" amd64
      success "macOS pkgs built under installers/macos/pkg/"
      ;;

    windows)
      warn "Windows installer must be built on Windows:"
      warn "  powershell -ExecutionPolicy Bypass -File installers/windows/build-installer.ps1"
      ;;
  esac
fi

