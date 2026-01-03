#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-1.0.0}"
KEYCHAIN_PROFILE="${2:-novakey-notary}"   # optional: name you used in notarytool store-credentials

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

ARM_PKG="installers/macos/pkg/NovaKey-${VERSION}-arm64.pkg"
AMD_PKG="installers/macos/pkg/NovaKey-${VERSION}-amd64.pkg"

test -f "$ARM_PKG" || { echo "Missing $ARM_PKG (build first)"; exit 1; }
test -f "$AMD_PKG" || { echo "Missing $AMD_PKG (build first)"; exit 1; }

CERT_LINE="$(security find-identity -v -p basic | grep 'Developer ID Installer' | head -n 1 || true)"
if [[ -z "$CERT_LINE" ]]; then
  echo "[x] No 'Developer ID Installer' identity found in Keychain."
  echo "Create one in Xcode -> Settings -> Accounts -> Manage Certificates -> '+' -> Developer ID Installer"
  exit 1
fi

CERT_NAME="$(echo "$CERT_LINE" | sed -n 's/.*"\(Developer ID Installer:.*\)".*/\1/p')"
if [[ -z "$CERT_NAME" ]]; then
  echo "[x] Could not parse certificate name."
  echo "$CERT_LINE"
  exit 1
fi

echo "[*] Using certificate:"
echo "    $CERT_NAME"

ARM_SIGNED="installers/macos/pkg/NovaKey-${VERSION}-arm64-signed.pkg"
AMD_SIGNED="installers/macos/pkg/NovaKey-${VERSION}-amd64-signed.pkg"

echo "[*] Signing pkgs..."
productsign --sign "$CERT_NAME" "$ARM_PKG" "$ARM_SIGNED"
productsign --sign "$CERT_NAME" "$AMD_PKG" "$AMD_SIGNED"

echo "[*] Verifying signatures..."
pkgutil --check-signature "$ARM_SIGNED"
pkgutil --check-signature "$AMD_SIGNED"

echo "[*] Notarizing (requires notarytool credentials profile: ${KEYCHAIN_PROFILE})..."
xcrun notarytool submit "$ARM_SIGNED" --keychain-profile "$KEYCHAIN_PROFILE" --wait
xcrun notarytool submit "$AMD_SIGNED" --keychain-profile "$KEYCHAIN_PROFILE" --wait

echo "[*] Stapling..."
xcrun stapler staple "$ARM_SIGNED"
xcrun stapler staple "$AMD_SIGNED"

echo "[✓] Done:"
ls -1 "$ARM_SIGNED" "$AMD_SIGNED"

