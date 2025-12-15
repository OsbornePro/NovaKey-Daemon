#!/bin/bash
set -euo pipefail

NVCLIENT="./dist/nvclient-linux-amd64"
SERVER_ADDR="127.0.0.1:60768"
ARM_ADDR="127.0.0.1:60769"
ARM_TOKEN_FILE="arm_token.txt"
ARM_MS="20000"

DEVICE_ID="phone"
KEY_HEX="7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234"
PASSWORD="SuperStrongPassword123!"
SERVER_KEYS_FILE="${SERVER_KEYS_FILE:-server_keys.json}"

[[ -x "$NVCLIENT" ]] || { echo "ERROR: nvclient not found at $NVCLIENT" >&2; exit 1; }
[[ -f "$SERVER_KEYS_FILE" ]] || { echo "ERROR: $SERVER_KEYS_FILE not found" >&2; exit 1; }

SERVER_KYBER_PUB_B64="$(
python3 - <<'PY'
import json
with open("server_keys.json","r",encoding="utf-8") as f:
    obj=json.load(f)
b64=(obj.get("kyber768_public") or "").strip()
if not b64: raise SystemExit("missing kyber768_public")
print(b64)
PY
)"

echo "Click into the focused field now..."
sleep 3

if [[ -f "$ARM_TOKEN_FILE" ]]; then chmod 600 "$ARM_TOKEN_FILE" 2>/dev/null || true; fi

# Arm (best effort)
if timeout 1 bash -c "cat < /dev/null > /dev/tcp/127.0.0.1/60769" 2>/dev/null; then
  echo "[+] Arm API detected at $ARM_ADDR"
  [[ -f "$ARM_TOKEN_FILE" ]] || { echo "ERROR: $ARM_TOKEN_FILE missing" >&2; exit 1; }
  echo "[+] Arming for ${ARM_MS}ms..."
  "$NVCLIENT" arm --addr "$ARM_ADDR" --token_file "$ARM_TOKEN_FILE" --ms "$ARM_MS"
else
  echo "[-] Arm API not detected at $ARM_ADDR (continuing)"
fi

echo "[+] Sending TWO-MAN approve (msgType=2)..."
"$NVCLIENT" approve \
  -addr "$SERVER_ADDR" \
  -device-id "$DEVICE_ID" \
  -key-hex "$KEY_HEX" \
  -server-kyber-pub-b64 "$SERVER_KYBER_PUB_B64"

sleep 0.2

echo "[+] Sending password frame (msgType=1)..."
"$NVCLIENT" \
  -addr "$SERVER_ADDR" \
  -device-id "$DEVICE_ID" \
  -password "$PASSWORD" \
  -key-hex "$KEY_HEX" \
  -server-kyber-pub-b64 "$SERVER_KYBER_PUB_B64"

echo "[+] Done."

