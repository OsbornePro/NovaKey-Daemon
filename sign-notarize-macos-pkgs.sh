#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-1.0.0}"
KEYCHAIN_PROFILE="${2:-novakey-notary}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

ARM_PKG="installers/macos/pkg/NovaKey-${VERSION}-arm64.pkg"
AMD_PKG="installers/macos/pkg/NovaKey-${VERSION}-amd64.pkg"

test -f "$ARM_PKG" || { echo "Missing $ARM_PKG (build first)"; exit 1; }
test -f "$AMD_PKG" || { echo "Missing $AMD_PKG (build first)"; exit 1; }

check_pkg_sig () {
  local pkg="$1"
  echo ""
  echo "[*] pkgutil --check-signature: $pkg"
  pkgutil --check-signature "$pkg" | sed -n '1,120p'

  if ! pkgutil --check-signature "$pkg" | grep -q "Developer ID Installer"; then
    echo "[x] Package is NOT signed with Developer ID Installer."
    echo "    Fix signing (productbuild --sign \"Developer ID Installer: ...\") and rebuild."
    exit 1
  fi
}

notarize_one () {
  local pkg="$1"

  check_pkg_sig "$pkg"

  echo ""
  echo "[*] Submitting to notary service: $pkg"
  local out status id
  out="$(xcrun notarytool submit "$pkg" --keychain-profile "$KEYCHAIN_PROFILE" --wait --output-format json)"
  echo "$out"

  status="$(printf '%s' "$out" | /usr/bin/python3 -c 'import sys,json; print(json.load(sys.stdin).get("status",""))')"
  id="$(printf '%s' "$out" | /usr/bin/python3 -c 'import sys,json; print(json.load(sys.stdin).get("id",""))')"

  echo "[*] Notary status: $status"
  echo "[*] Notary id: $id"

  if [[ "$status" != "Accepted" ]]; then
    echo ""
    echo "[x] Notarization NOT accepted. Fetching log:"
    xcrun notarytool log "$id" --keychain-profile "$KEYCHAIN_PROFILE"
    exit 1
  fi

  echo "[*] Stapling..."
  xcrun stapler staple "$pkg"
  xcrun stapler validate "$pkg"

  echo "[✓] Stapled: $pkg"
  echo "[*] Gatekeeper assessment (install):"
  spctl -a -vv --type install "$pkg" || true
}

notarize_one "$ARM_PKG"
notarize_one "$AMD_PKG"

echo ""
echo "[✓] Done:"
ls -1 "$ARM_PKG" "$AMD_PKG"

