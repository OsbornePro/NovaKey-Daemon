#!/bin/bash
set -euo pipefail

VERSION="${1:-1.0.0}"
OUT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${OUT_DIR}/../../.." && pwd)"

ARCH="${2:-arm64}"
if [[ "$ARCH" != "arm64" && "$ARCH" != "amd64" && "$ARCH" != "x86_64" ]]; then
  echo "Usage: $0 <version> <arm64|amd64>"
  exit 1
fi
if [[ "$ARCH" == "x86_64" ]]; then ARCH="amd64"; fi

# Ensure keychain search list includes roots + system intermediates
security list-keychains -d user -s \
  "${HOME}/Library/Keychains/login.keychain-db" \
  "/Library/Keychains/System.keychain" \
  "/System/Library/Keychains/SystemRootCertificates.keychain" >/dev/null

# Identity hashes (stable)
APP_ID_HASH="E37E607B0C730C95445DFE53A3B13AB8B413E834"   # Developer ID Application
PKG_ID_HASH="F97BE43C6F7D4DDB2ABB4DD4C73FEDB0235F3CB5"   # Developer ID Installer

STAGE="$(mktemp -d)"
PKGROOT="${STAGE}/root"
SCRIPTS="${STAGE}/scripts"
mkdir -p "${PKGROOT}/usr/local/novakey" "${SCRIPTS}"

if [[ "$ARCH" == "arm64" ]]; then
  BIN_SRC="${REPO_ROOT}/dist/macos/novakey-arm64"
else
  BIN_SRC="${REPO_ROOT}/dist/macos/novakey-amd64"
fi

test -f "$BIN_SRC" || { echo "Missing macOS binary: $BIN_SRC"; exit 1; }

# Copy payload files
install -m 755 "$BIN_SRC" "${PKGROOT}/usr/local/novakey/novakey"
install -m 644 "${REPO_ROOT}/server_config.yaml" "${PKGROOT}/usr/local/novakey/server_config.yaml"
install -m 644 "${OUT_DIR}/com.osbornepro.novakey.plist.template" \
  "${PKGROOT}/usr/local/novakey/com.osbornepro.novakey.plist.template"

if [[ -f "${REPO_ROOT}/devices.json" ]]; then
  install -m 644 "${REPO_ROOT}/devices.json" "${PKGROOT}/usr/local/novakey/devices.json"
fi

install -m 755 "${OUT_DIR}/postinstall" "${SCRIPTS}/postinstall"
if [[ -f "${OUT_DIR}/preinstall" ]]; then
  install -m 755 "${OUT_DIR}/preinstall" "${SCRIPTS}/preinstall"
fi

echo "[*] Signing payload executable (Developer ID Application + hardened runtime + timestamp)"
codesign --force --options runtime --timestamp --sign "${APP_ID_HASH}" \
  "${PKGROOT}/usr/local/novakey/novakey"

codesign --verify --strict --verbose=4 "${PKGROOT}/usr/local/novakey/novakey"

PKG_NAME="NovaKey-${VERSION}-${ARCH}.pkg"
COMPONENT_PKG="${STAGE}/NovaKey-${VERSION}-${ARCH}-component.pkg"
FINAL_PKG="${OUT_DIR}/${PKG_NAME}"

# IMPORTANT: sign the component pkg too
pkgbuild \
  --sign "${PKG_ID_HASH}" \
  --root "${PKGROOT}" \
  --scripts "${SCRIPTS}" \
  --identifier "com.osbornepro.novakey" \
  --version "${VERSION}" \
  "${COMPONENT_PKG}"

# Sign the final product pkg
productbuild \
  --sign "${PKG_ID_HASH}" \
  --package "${COMPONENT_PKG}" \
  "${FINAL_PKG}"

echo "[*] Checking final pkg signature:"
pkgutil --check-signature "${FINAL_PKG}"

echo "[✓] Built signed product pkg: ${FINAL_PKG}"
echo "NEXT: notarize + staple"

