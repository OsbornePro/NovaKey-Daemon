#!/bin/bash
# NOTE: The public base64 value changes every time the service is restarted

set -euo pipefail

NVCLIENT="./dist/nvclient-linux-amd64"
SERVER_ADDR="127.0.0.1:60768"
ARM_ADDR="127.0.0.1:60769"
ARM_TOKEN_FILE="arm_token.txt"
ARM_MS="20000"

DEVICE_ID="phone"
KEY_HEX="7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234"
PASSWORD="SuperStrongPassword123!"
APPROVE_MAGIC="__NOVAKEY_APPROVE__"

# Set LEGACY_APPROVE=1 to send legacy approve (msgType=1 payload magic)
LEGACY_APPROVE="${LEGACY_APPROVE:-0}"

SERVER_KYBER_PUB_B64='...your base64...'

if [[ ! -x "$NVCLIENT" ]]; then
  echo "ERROR: nvclient not found/executable at $NVCLIENT" >&2
  exit 1
fi

echo "Click into the browser address bar (or focused field) now..."
sleep 3

if [[ -f "$ARM_TOKEN_FILE" ]]; then
  chmod 600 "$ARM_TOKEN_FILE" 2>/dev/null || true
fi

send_inject () {
  local payload="$1"
  "$NVCLIENT" \
    -addr "$SERVER_ADDR" \
    -device-id "$DEVICE_ID" \
    -password "$payload" \
    -key-hex "$KEY_HEX" \
    -server-kyber-pub-b64 "$SERVER_KYBER_PUB_B64"
}

send_approve () {
  if [[ "$LEGACY_APPROVE" == "1" ]]; then
    "$NVCLIENT" approve --legacy_magic --magic "$APPROVE_MAGIC" \
      -addr "$SERVER_ADDR" \
      -device-id "$DEVICE_ID" \
      -key-hex "$KEY_HEX" \
      -server-kyber-pub-b64 "$SERVER_KYBER_PUB_B64"
  else
    "$NVCLIENT" approve \
      -addr "$SERVER_ADDR" \
      -device-id "$DEVICE_ID" \
      -key-hex "$KEY_HEX" \
      -server-kyber-pub-b64 "$SERVER_KYBER_PUB_B64"
  fi
}

# Arm if API is listening
if timeout 1 bash -c "cat < /dev/null > /dev/tcp/127.0.0.1/60769" 2>/dev/null; then
  echo "[+] Arm API detected at $ARM_ADDR"

  if [[ ! -f "$ARM_TOKEN_FILE" ]]; then
    echo "ERROR: $ARM_TOKEN_FILE not found. Start novakey with arm_api_enabled:true so it auto-generates the token." >&2
    exit 1
  fi

  echo "[+] Arming for ${ARM_MS}ms..."
  "$NVCLIENT" arm --addr "$ARM_ADDR" --token_file "$ARM_TOKEN_FILE" --ms "$ARM_MS || {
    echo "ERROR: arming failed" >&2
    exit 1
  }
else
  echo "[-] Arm API not detected at $ARM_ADDR (continuing without arming)"
fi

echo "[+] Sending TWO-MAN approve..."
send_approve || {
  echo "ERROR: approve send failed" >&2
  exit 1
}

# Must still be within approve_window_ms (15s in your config)
sleep 0.2

echo "[+] Sending encrypted password frame to $SERVER_ADDR..."
send_inject "$PASSWORD"
echo "[+] Done."

