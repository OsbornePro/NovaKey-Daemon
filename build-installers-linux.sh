#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-1.0.0}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

command -v nfpm >/dev/null 2>&1 || {
  echo "[x] nfpm not found in PATH."
  echo "Install: go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest"
  exit 1
}

chmod +x installers/linux/nfpm/build-packages.sh \
         installers/linux/nfpm/postinstall.sh \
         installers/linux/nfpm/postremove.sh

# Sanity check inputs
test -f "dist/linux/novakey-linux-amd64.elf" || { echo "Missing dist/linux/novakey-linux-amd64.elf"; exit 1; }
test -f "dist/linux/novakey-linux-arm64.elf" || { echo "Missing dist/linux/novakey-linux-arm64.elf"; exit 1; }

echo "[*] Building Linux packages for version ${VERSION}"
./installers/linux/nfpm/build-packages.sh "$VERSION"

echo ""
echo "[✓] Linux packages are in dist/linux/"
ls -1 dist/linux | sed 's/^/  - /'

