#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-1.0.0}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

# Ensure scripts are executable
chmod +x installers/macos/pkg/build-pkg.sh installers/macos/pkg/postinstall

# Sanity check inputs
test -f "dist/macos/novakey-arm64" || { echo "Missing dist/macos/novakey-arm64"; exit 1; }
test -f "dist/macos/novakey-amd64" || { echo "Missing dist/macos/novakey-amd64"; exit 1; }

echo "[*] Building macOS pkgs for version ${VERSION}"
./installers/macos/pkg/build-pkg.sh "$VERSION" arm64
./installers/macos/pkg/build-pkg.sh "$VERSION" amd64

echo ""
echo "[✓] Built:"
ls -1 "installers/macos/pkg/NovaKey-${VERSION}-arm64.pkg" "installers/macos/pkg/NovaKey-${VERSION}-amd64.pkg"

if security find-identity -v -p basic | grep -q "Developer ID Installer"; then
  echo "Developer ID Installer identity found."
else
  echo "You do NOT have a 'Developer ID Installer' identity in Keychain."
  echo "Create one in Xcode -> Settings -> Accounts -> Manage Certificates -> '+' -> Developer ID Installer"
fi

