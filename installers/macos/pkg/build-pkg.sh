#!/bin/bash
set -euo pipefail

VERSION="${1:-1.0.0}"
OUT_DIR="$(cd "$(dirname "$0")" && pwd)"
# pkg dir -> macos -> installers -> repo root
REPO_ROOT="$(cd "${OUT_DIR}/../../.." && pwd)"

STAGE="$(mktemp -d)"
PKGROOT="${STAGE}/root"
SCRIPTS="${STAGE}/scripts"

mkdir -p "${PKGROOT}/usr/local/novakey" "${SCRIPTS}"

ARCH="${2:-arm64}"
BIN_SRC=""
if [[ "$ARCH" == "arm64" ]]; then
  BIN_SRC="${REPO_ROOT}/dist/macos/novakey-arm64"
else
  BIN_SRC="${REPO_ROOT}/dist/macos/novakey-amd64"
fi

if [[ ! -f "$BIN_SRC" ]]; then
  echo "Missing macOS binary: $BIN_SRC"
  exit 1
fi

cp -f "$BIN_SRC" "${PKGROOT}/usr/local/novakey/novakey"
cp -f "${REPO_ROOT}/server_config.yaml" "${PKGROOT}/usr/local/novakey/server_config.yaml"
cp -f "${OUT_DIR}/com.osbornepro.novakey.plist.template" \
  "${PKGROOT}/usr/local/novakey/com.osbornepro.novakey.plist.template"

if [[ -f "${REPO_ROOT}/devices.json" ]]; then
  cp -f "${REPO_ROOT}/devices.json" "${PKGROOT}/usr/local/novakey/devices.json"
fi

cp -f "${OUT_DIR}/postinstall" "${SCRIPTS}/postinstall"
chmod +x "${SCRIPTS}/postinstall"

PKG_NAME="NovaKey-${VERSION}-${ARCH}.pkg"

pkgbuild \
  --root "${PKGROOT}" \
  --scripts "${SCRIPTS}" \
  --identifier "com.osbornepro.novakey" \
  --version "${VERSION}" \
  "${OUT_DIR}/${PKG_NAME}"

echo "Built ${OUT_DIR}/${PKG_NAME}"
echo "NEXT: sign + notarize with Developer ID Installer"

