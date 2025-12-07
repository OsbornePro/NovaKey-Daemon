#!/usr/bin/env bash
# =============================================================================
# NovaKey – Unified cross-platform build script (Linux & macOS)
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

# ----------------------------- Usage -----------------------------
readonly USAGE=$(cat <<EOF
${CYAN}NovaKey Cross-Platform Build Script${NC}

Syntax: ./build.sh [-t target] [-c] [-f filename]

OPTIONS:
  -t, --target     Build target: windows | linux | darwin          [default: linux]
  -c, --clean      Delete dist/ folder before building
  -f, --file       Output filename (default: NovaKey or NovaKey.exe)
  -h, --help       Show this help

EXAMPLES:
  ./build.sh                    # Build for current OS (Linux)
  ./build.sh -t windows         # Cross-compile for Windows
  ./build.sh -t darwin -c       # Clean + build universal macOS binary
  ./build.sh -t linux -f mykey  # Build Linux binary named "mykey"

EOF
)

# ----------------------------- Helpers -----------------------------
log()    { printf "${CYAN}[-] %s ${NC}%s${NC}\n" "$(date '+%m-%d-%Y %H:%M:%S')" "$1"; }
success(){ printf "${GREEN}[✓] %s${NC}\n" "$1"; }
error()  { printf "${RED}[x] %s${NC}\n" "$1" >&2; exit 1; }

# ----------------------------- Parse Args -----------------------------
TARGET="linux"
CLEAN=false
FILENAME=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        -t|--target)
            TARGET="$2"
            shift 2
            ;;
        -c|--clean)
            CLEAN=true
            shift
            ;;
        -f|--file|--filename)
            FILENAME="$2"
            shift 2
            ;;
        -h|--help)
            printf "%b\n" "$USAGE"
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

case "$TARGET" in
    windows|linux|darwin) ;;
    *) error "Invalid target: $TARGET (use: windows, linux, darwin)" ;;
esac

# ----------------------------- Project Root -----------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."
PROJECT_ROOT="$PWD"

# ----------------------------- Version & Flags -----------------------------
VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "dev")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE}"

log "Building NovaKey ${VERSION} for ${TARGET^^}"

# ----------------------------- Clean -----------------------------
if $CLEAN; then
    log "Cleaning previous build artifacts"
    rm -rf dist
fi

mkdir -p dist

# ----------------------------- Platform Setup -----------------------------
GOOS="$TARGET"
GOARCH="amd64"
OUTPUT_NAME="NovaKey"

case "$TARGET" in
    windows)
        OUTPUT_NAME="NovaKey.exe"
        [[ -n "$FILENAME" ]] && OUTPUT_NAME="$FILENAME"
        [[ "$OUTPUT_NAME" != *.exe ]] && OUTPUT_NAME+=".exe"
        ;;
    darwin)
        # Universal binary (Intel + Apple Silicon)
        GOARCH="all"
        [[ -n "$FILENAME" ]] && OUTPUT_NAME="$FILENAME"
        ;;
    linux)
        [[ -n "$FILENAME" ]] && OUTPUT_NAME="$FILENAME"
        ;;
esac

OUTPUT_PATH="dist/$OUTPUT_NAME"

# ----------------------------- Build -----------------------------
log "Target → ${GOOS}/${GOARCH} → ${OUTPUT_PATH}"
log "Running: go build -ldflags='$LDFLAGS' -o '$OUTPUT_PATH' ./cmd/novakey"

CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
    go build -trimpath -ldflags="$LDFLAGS" -o "$OUTPUT_PATH" ./cmd/novakey || \
    error "Go build failed"

# ----------------------------- Success -----------------------------
printf "\n${GREEN}"
cat <<EOF
╔══════════════════════════════════════════════════════════╗
║                    BUILD SUCCESSFUL                      ║
║ Target   : $TARGET $( [[ "$TARGET" == "darwin" ]] && echo "(Universal)" )      ║
║ Binary   : $PROJECT_ROOT/$OUTPUT_PATH
║ Size     : $(du -h "$OUTPUT_PATH" | cut -f1)
║ SHA256   : $(sha256sum "$OUTPUT_PATH" | cut -d' ' -f1)
╚══════════════════════════════════════════════════════════╝
EOF
printf "${NC}\n"
