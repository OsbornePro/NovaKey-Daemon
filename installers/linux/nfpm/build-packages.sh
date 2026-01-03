#!/bin/bash
set -euo pipefail

VERSION="${1:-1.0.0}"
HERE="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${HERE}/../../.." && pwd)"

need() { command -v "$1" >/dev/null 2>&1 || { echo "Missing required tool: $1" >&2; exit 1; }; }
need nfpm
need sed
need mktemp

build_from_yaml() {
  local yaml="$1"
  local deb_out="$2"
  local rpm_out="$3"

  local tmp_yaml
  tmp_yaml="$(mktemp)"
  sed "s/^version: \".*\"/version: \"${VERSION}\"/" "${yaml}" > "${tmp_yaml}"

  nfpm pkg -f "${tmp_yaml}" -p deb -t "${deb_out}"
  nfpm pkg -f "${tmp_yaml}" -p rpm -t "${rpm_out}"

  rm -f "${tmp_yaml}"
}

build_from_yaml \
  "${HERE}/nfpm.yaml" \
  "${REPO_ROOT}/dist/linux/novakey_${VERSION}_amd64.deb" \
  "${REPO_ROOT}/dist/linux/novakey-${VERSION}-1.amd64.rpm"

build_from_yaml \
  "${HERE}/nfpm-arm64.yaml" \
  "${REPO_ROOT}/dist/linux/novakey_${VERSION}_arm64.deb" \
  "${REPO_ROOT}/dist/linux/novakey-${VERSION}-1.aarch64.rpm"

echo "Built Linux packages under dist/linux/"

