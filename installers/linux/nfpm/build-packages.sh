#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-1.0.0}"

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${HERE}/../../.." && pwd)"

need() { command -v "$1" >/dev/null 2>&1 || { echo "Missing required tool: $1" >&2; exit 1; }; }
need nfpm
need sed

NFPM_AMD="${HERE}/nfpm.yaml"
NFPM_ARM="${HERE}/nfpm-arm64.yaml"

[[ -f "$NFPM_AMD" ]] || { echo "Missing $NFPM_AMD" >&2; exit 1; }
[[ -f "$NFPM_ARM" ]] || { echo "Missing $NFPM_ARM" >&2; exit 1; }

# Absolute input paths (what you actually built)
BIN_AMD="${REPO_ROOT}/dist/linux/novakey-linux-amd64.elf"
BIN_ARM="${REPO_ROOT}/dist/linux/novakey-linux-arm64.elf"

[[ -f "$BIN_AMD" ]] || { echo "Missing binary: $BIN_AMD" >&2; exit 1; }
[[ -f "$BIN_ARM" ]] || { echo "Missing binary: $BIN_ARM" >&2; exit 1; }

OUT_DIR="${REPO_ROOT}/dist/linux"
mkdir -p "$OUT_DIR"

echo "[*] Repo root: $REPO_ROOT"
echo "[*] nfpm dir : $HERE"
echo "[*] Version  : $VERSION"

TMP_AMD="${HERE}/.nfpm.tmp.amd64.yaml"
TMP_ARM="${HERE}/.nfpm.tmp.arm64.yaml"

cleanup() { rm -f "$TMP_AMD" "$TMP_ARM"; }
trap cleanup EXIT

# Patch YAML so nfpm uses ABSOLUTE src paths (no CWD ambiguity),
# but keep scripts relative by running nfpm from $HERE.
patch_yaml() {
  local in_yaml="$1"
  local out_yaml="$2"
  local bin_abs="$3"

  sed -E \
    -e "s/^version: \"[^\"]*\"/version: \"${VERSION}\"/" \
    -e "s|^([[:space:]]*- src: )[[:space:]]*../../dist/linux/novakey-linux-[a-z0-9]+\\.elf$|\\1${bin_abs}|" \
    -e "s|^([[:space:]]*- src: )[[:space:]]*../systemd/novakey\\.service$|\\1${REPO_ROOT}/installers/linux/systemd/novakey.service|" \
    -e "s|^([[:space:]]*- src: )[[:space:]]*../../server_config\\.yaml$|\\1${REPO_ROOT}/server_config.yaml|" \
    "$in_yaml" > "$out_yaml"
}

patch_yaml "$NFPM_AMD" "$TMP_AMD" "$BIN_AMD"
patch_yaml "$NFPM_ARM" "$TMP_ARM" "$BIN_ARM"

# Output filenames
DEB_AMD="${OUT_DIR}/novakey_${VERSION}_amd64.deb"
RPM_AMD="${OUT_DIR}/novakey-${VERSION}-1.amd64.rpm"
DEB_ARM="${OUT_DIR}/novakey_${VERSION}_arm64.deb"
RPM_ARM="${OUT_DIR}/novakey-${VERSION}-1.aarch64.rpm"

# Run nfpm from HERE so scripts: ./postinstall.sh resolve correctly
cd "$HERE"

nfpm pkg -f "$TMP_AMD" -p deb -t "$DEB_AMD"
nfpm pkg -f "$TMP_AMD" -p rpm -t "$RPM_AMD"

nfpm pkg -f "$TMP_ARM" -p deb -t "$DEB_ARM"
nfpm pkg -f "$TMP_ARM" -p rpm -t "$RPM_ARM"

echo "[✓] Built packages:"
echo "  $DEB_AMD"
echo "  $RPM_AMD"
echo "  $DEB_ARM"
echo "  $RPM_ARM"

