#!/usr/bin/env bash
set -euo pipefail

# ---------------------------
# Config (override via env)
# ---------------------------
SERVER_ADDR="${SERVER_ADDR:-127.0.0.1:60768}"
ARM_ADDR="${ARM_ADDR:-127.0.0.1:60769}"
ARM_TOKEN_FILE="${ARM_TOKEN_FILE:-arm_token.txt}"
ARM_MS="${ARM_MS:-20000}"

DEVICE_ID="${DEVICE_ID:-phone}"
KEY_HEX="${KEY_HEX:-7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234}"
PASSWORD="${PASSWORD:-SuperStrongPassword123!}"

# If your server has two_man_enabled:true, keep this true.
TWO_MAN_ENABLED="${TWO_MAN_ENABLED:-true}"

SERVER_KEYS_FILE="${SERVER_KEYS_FILE:-server_keys.json}"

# ---------------------------
# Pick nvclient binary
# ---------------------------
OS="$(uname -s)"
ARCH="$(uname -m)"

NVCLIENT="${NVCLIENT:-}"
if [[ -z "$NVCLIENT" ]]; then
  if [[ "$OS" == "Darwin" ]]; then
    # Try a few common names
    if [[ -x "./dist/nvclient-darwin-amd64" ]]; then
      NVCLIENT="./dist/nvclient-darwin-amd64"
    elif [[ -x "./dist/nvclient-darwin-arm64" ]]; then
      NVCLIENT="./dist/nvclient-darwin-arm64"
    elif [[ -x "./dist/nvclient" ]]; then
      NVCLIENT="./dist/nvclient"
    else
      echo "ERROR: nvclient not found under ./dist (expected nvclient-darwin-amd64 or nvclient-darwin-arm64)" >&2
      exit 1
    fi
  elif [[ "$OS" == "Linux" ]]; then
    if [[ -x "./dist/nvclient-linux-amd64" ]]; then
      NVCLIENT="./dist/nvclient-linux-amd64"
    elif [[ -x "./dist/nvclient-linux-arm64" ]]; then
      NVCLIENT="./dist/nvclient-linux-arm64"
    elif [[ -x "./dist/nvclient" ]]; then
      NVCLIENT="./dist/nvclient"
    else
      echo "ERROR: nvclient not found under ./dist (expected nvclient-linux-amd64)" >&2
      exit 1
    fi
  else
    echo "ERROR: unsupported OS: $OS" >&2
    exit 1
  fi
fi

if [[ ! -x "$NVCLIENT" ]]; then
  echo "ERROR: NVCLIENT is not executable: $NVCLIENT" >&2
  exit 1
fi

# ---------------------------
# Read kyber pubkey from server_keys.json
# ---------------------------
if [[ ! -f "$SERVER_KEYS_FILE" ]]; then
  echo "ERROR: server keys file not found: $SERVER_KEYS_FILE" >&2
  exit 1
fi

PYTHON_BIN=""
if command -v python3 >/dev/null 2>&1; then
  PYTHON_BIN="python3"
elif command -v python >/dev/null 2>&1; then
  PYTHON_BIN="python"
else
  echo "ERROR: python3/python not found (needed to parse $SERVER_KEYS_FILE)" >&2
  exit 1
fi

SERVER_KYBER_PUB_B64="$("$PYTHON_BIN" -c 'import json,sys,re; o=json.load(open(sys.argv[1],"r")); s=o.get("kyber768_public",""); s=re.sub(r"\s+","",s.strip()); 
import base64; base64.b64decode(s); print(s)' "$SERVER_KEYS_FILE")"

# ---------------------------
# Helper: normalize booleans (bash 3.2 safe)
# ---------------------------
is_true() {
  case "$1" in
    1|true|TRUE|True|yes|YES|Yes|y|Y|on|ON|On) return 0 ;;
    *) return 1 ;;
  esac
}

echo "Click into the browser address bar (or focused field) now..."
sleep 3

# Tighten token perms best-effort
if [[ -f "$ARM_TOKEN_FILE" ]]; then
  chmod 600 "$ARM_TOKEN_FILE" 2>/dev/null || true
fi

# ---------------------------
# Arm (required if two-man enabled in your daemon flow)
# ---------------------------
echo "[+] Arming for ${ARM_MS}ms..."
"$NVCLIENT" arm --addr "$ARM_ADDR" --token_file "$ARM_TOKEN_FILE" --ms "$ARM_MS"

# ---------------------------
# Two-man approve (typed msgType=2)
# ---------------------------
if is_true "$TWO_MAN_ENABLED"; then
  echo "[+] Sending TWO-MAN approval control payload..."
  # This requires nvclient to have an "approve" subcommand
  if "$NVCLIENT" approve -h >/dev/null 2>&1; then
    "$NVCLIENT" approve \
      -addr "$SERVER_ADDR" \
      -device-id "$DEVICE_ID" \
      -key-hex "$KEY_HEX" \
      -server-kyber-pub-b64 "$SERVER_KYBER_PUB_B64"
  else
    echo "ERROR: nvclient does not support 'approve' yet." >&2
    echo "Rebuild nvclient after adding the approve subcommand (msgType=2 typed message frames)." >&2
    exit 1
  fi
fi

# Must be within approve_window_ms; keep small
sleep 0.2

# ---------------------------
# Send actual password (typed inject msgType=1)
# ---------------------------
echo "[+] Sending encrypted password frame to $SERVER_ADDR..."
"$NVCLIENT" \
  -addr "$SERVER_ADDR" \
  -device-id "$DEVICE_ID" \
  -password "$PASSWORD" \
  -key-hex "$KEY_HEX" \
  -server-kyber-pub-b64 "$SERVER_KYBER_PUB_B64"

echo "[+] Done."

