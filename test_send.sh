#!/bin/bash
# send_test.sh
# Reads server kyber public key from server_keys.json, then uses nvclient to:
#  1) (optional) arm via local Arm API if reachable
#  2) (optional) send TWO-MAN approve (msgType=2) if enabled
#  3) send the real password (msgType=1)
#
# Requirements (Linux):
#  - bash
#  - ./dist/nvclient-linux-amd64 present + executable
#  - server_keys.json present (default in repo root)
#  - Optional: arm_token.txt if Arm API enabled
#
# Optional tools:
#  - jq (preferred) OR python3 fallback for JSON parsing

set -euo pipefail

NVCLIENT="${NVCLIENT:-./dist/nvclient-linux-amd64}"
SERVER_ADDR="${SERVER_ADDR:-127.0.0.1:60768}"

# Arm API (optional)
ARM_ADDR="${ARM_ADDR:-127.0.0.1:60769}"
ARM_TOKEN_FILE="${ARM_TOKEN_FILE:-arm_token.txt}"
ARM_MS="${ARM_MS:-20000}"

# Two-man (optional)
TWO_MAN_ENABLED="${TWO_MAN_ENABLED:-true}"
APPROVE_MAGIC="${APPROVE_MAGIC:-__NOVAKEY_APPROVE__}"

# Device + secret
DEVICE_ID="${DEVICE_ID:-phone}"
KEY_HEX="${KEY_HEX:-7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234}"
PASSWORD="${PASSWORD:-SuperStrongPassword123!}"

# Server keys file
SERVER_KEYS_FILE="${SERVER_KEYS_FILE:-server_keys.json}"

die() { echo "ERROR: $*" >&2; exit 1; }

[[ -x "$NVCLIENT" ]] || die "nvclient not found/executable at $NVCLIENT"
[[ -f "$SERVER_KEYS_FILE" ]] || die "server keys file not found: $SERVER_KEYS_FILE"

get_kyber_pub_b64() {
  local path="$1"
  if command -v jq >/dev/null 2>&1; then
    # -r strips quotes; also strip whitespace just in case
    jq -r '.kyber768_public // empty' "$path" | tr -d ' \t\r\n'
    return
  fi

  if command -v python3 >/dev/null 2>&1; then
    python3 - <<'PY' "$path"
import json,sys
p=sys.argv[1]
with open(p,'r',encoding='utf-8') as f:
    o=json.load(f)
v=(o.get('kyber768_public') or '').strip()
v=''.join(v.split())
print(v)
PY
    return
  fi

  die "need jq or python3 to parse $path"
}

SERVER_KYBER_PUB_B64="$(get_kyber_pub_b64 "$SERVER_KEYS_FILE")"
[[ -n "$SERVER_KYBER_PUB_B64" ]] || die "kyber768_public missing/empty in $SERVER_KEYS_FILE"

# Quick TCP check (bash /dev/tcp)
tcp_up() {
  local hostport="$1"
  local host="${hostport%:*}"
  local port="${hostport#*:}"
  # shellcheck disable=SC2086
  timeout 1 bash -c "cat < /dev/null > /dev/tcp/${host}/${port}" >/dev/null 2>&1
}

echo "Click into the browser address bar (or focused field) now..."
sleep 3

# Best-effort fix for hardened arm_token perms
if [[ -f "$ARM_TOKEN_FILE" ]]; then
  chmod 600 "$ARM_TOKEN_FILE" 2>/dev/null || true
fi

send_payload () {
  local payload="$1"
  "$NVCLIENT" \
    -addr "$SERVER_ADDR" \
    -device-id "$DEVICE_ID" \
    -password "$payload" \
    -key-hex "$KEY_HEX" \
    -server-kyber-pub-b64 "$SERVER_KYBER_PUB_B64"
}

# --- ARM (optional) ---
if tcp_up "$ARM_ADDR"; then
  echo "[+] Arm API detected at $ARM_ADDR"
  [[ -f "$ARM_TOKEN_FILE" ]] || die "Arm API is up but token file not found: $ARM_TOKEN_FILE"
  echo "[+] Arming for ${ARM_MS}ms..."
  "$NVCLIENT" arm --addr "$ARM_ADDR" --token_file "$ARM_TOKEN_FILE" --ms "$ARM_MS" || die "arming failed"
else
  echo "[-] Arm API not detected at $ARM_ADDR (continuing without arming)"
fi

# --- TWO-MAN approve (optional) ---
if [[ "${TWO_MAN_ENABLED,,}" == "true" || "${TWO_MAN_ENABLED,,}" == "1" || "${TWO_MAN_ENABLED,,}" == "yes" ]]; then
  echo "[+] Sending TWO-MAN approval control payload..."
  send_payload "$APPROVE_MAGIC" || die "approval payload send failed"
  # keep this short; approval window is often 15s
  sleep 0.2
else
  echo "[-] TWO-MAN disabled for this test (TWO_MAN_ENABLED=$TWO_MAN_ENABLED)"
fi

echo "[+] Sending encrypted password frame to $SERVER_ADDR..."
send_payload "$PASSWORD"
echo "[+] Done."

