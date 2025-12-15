#!/usr/bin/env bash
# test_send.sh (portable: macOS bash 3.2 + Linux bash 4/5)
# - Reads server Kyber public key from server_keys.json
# - Optionally arms via local Arm API if reachable
# - Optionally sends TWO-MAN approve control first
# - Sends the real password frame

set -euo pipefail

# -------------------------
# Config (override via env)
# -------------------------
NVCLIENT="${NVCLIENT:-./dist/nvclient-linux-amd64}"     # mac: set NVCLIENT=./dist/nvclient-darwin-arm64 (or your name)
SERVER_ADDR="${SERVER_ADDR:-127.0.0.1:60768}"

SERVER_KEYS_FILE="${SERVER_KEYS_FILE:-./server_keys.json}"

DEVICE_ID="${DEVICE_ID:-phone}"
KEY_HEX="${KEY_HEX:-7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234}"
PASSWORD="${PASSWORD:-SuperStrongPassword123!}"

# Arm (optional)
ARM_ADDR="${ARM_ADDR:-127.0.0.1:60769}"
ARM_TOKEN_FILE="${ARM_TOKEN_FILE:-./arm_token.txt}"
ARM_MS="${ARM_MS:-20000}"

# Two-man (optional)
TWO_MAN_ENABLED="${TWO_MAN_ENABLED:-true}"
APPROVE_MAGIC="${APPROVE_MAGIC:-__NOVAKEY_APPROVE__}"

# -------------------------
# Helpers
# -------------------------
die() { echo "ERROR: $*" >&2; exit 1; }

is_true() {
  # Accept: true/1/yes/on/y (case-insensitive)
  case "$(printf '%s' "${1:-}" | tr '[:upper:]' '[:lower:]')" in
    1|true|yes|y|on) return 0 ;;
    *)               return 1 ;;
  esac
}

read_kyber_pub_b64() {
  local f="$1"
  [[ -f "$f" ]] || die "server keys file not found: $f"

  # Prefer jq if available
  if command -v jq >/dev/null 2>&1; then
    jq -r '.kyber768_public // empty' "$f" | tr -d '[:space:]'
    return 0
  fi

  # Fallback: python3 one-liner (NO heredoc)
  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import json,sys; obj=json.load(open(sys.argv[1],"r",encoding="utf-8")); v=obj.get("kyber768_public",""); print("".join(str(v).split()))' "$f"
    return 0
  fi

  # Last resort: sed (brittle but OK for simple JSON)
  sed -n 's/.*"kyber768_public"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$f" | tr -d '[:space:]'
}

validate_b64() {
  local b64="$1"
  [[ -n "$b64" ]] || return 1

  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import base64,sys; base64.b64decode(sys.argv[1], validate=True)' "$b64" >/dev/null 2>&1 || return 1
  fi
  return 0
}

tcp_port_open() {
  # Usage: tcp_port_open host port timeout_seconds
  local host="$1" port="$2" t="${3:-1}"

  # Prefer nc if present
  if command -v nc >/dev/null 2>&1; then
    nc -z -w "$t" "$host" "$port" >/dev/null 2>&1
    return $?
  fi

  # Bash /dev/tcp (works in bash on both mac/linux)
  if command -v timeout >/dev/null 2>&1; then
    timeout "$t" bash -c "cat < /dev/null > /dev/tcp/$host/$port" >/dev/null 2>&1
    return $?
  fi

  # No timeout on stock macOS: best-effort quick connect
  bash -c "cat < /dev/null > /dev/tcp/$host/$port" >/dev/null 2>&1
  return $?
}

send_payload() {
  local payload="$1"
  "$NVCLIENT" \
    -addr "$SERVER_ADDR" \
    -device-id "$DEVICE_ID" \
    -password "$payload" \
    -key-hex "$KEY_HEX" \
    -server-kyber-pub-b64 "$SERVER_KYBER_PUB_B64"
}

# -------------------------
# Main
# -------------------------
[[ -x "$NVCLIENT" ]] || die "nvclient not found/executable at: $NVCLIENT"

SERVER_KYBER_PUB_B64="$(read_kyber_pub_b64 "$SERVER_KEYS_FILE")"
[[ -n "$SERVER_KYBER_PUB_B64" ]] || die "kyber768_public missing/empty in $SERVER_KEYS_FILE"
validate_b64 "$SERVER_KYBER_PUB_B64" || die "kyber768_public in $SERVER_KEYS_FILE is not valid base64"

echo "Click into the browser address bar (or focused field) now..."
sleep 3

# Best-effort: lock down token file perms (mac/linux)
if [[ -f "$ARM_TOKEN_FILE" ]]; then
  chmod 600 "$ARM_TOKEN_FILE" 2>/dev/null || true
fi

# --- ARM (optional) ---
ARM_HOST="${ARM_ADDR%:*}"
ARM_PORT="${ARM_ADDR##*:}"
if [[ -n "$ARM_HOST" && -n "$ARM_PORT" ]] && tcp_port_open "$ARM_HOST" "$ARM_PORT" 1; then
  echo "[+] Arm API detected at $ARM_ADDR"
  [[ -f "$ARM_TOKEN_FILE" ]] || die "Arm API is up but token file not found: $ARM_TOKEN_FILE"

  echo "[+] Arming for ${ARM_MS}ms..."
  "$NVCLIENT" arm --addr "$ARM_ADDR" --token_file "$ARM_TOKEN_FILE" --ms "$ARM_MS"
else
  echo "[-] Arm API not detected at $ARM_ADDR (continuing without arming)"
fi

# --- TWO-MAN approve (optional) ---
if is_true "$TWO_MAN_ENABLED"; then
  echo "[+] Sending TWO-MAN approval control payload..."
  send_payload "$APPROVE_MAGIC"
  sleep 0.2
fi

# --- Send real password ---
echo "[+] Sending encrypted password frame to $SERVER_ADDR..."
send_payload "$PASSWORD"
echo "[+] Done."

