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

SERVER_KYBER_PUB_B64='wgkdGfeAyuqswle8f9e9Aagxc1gQr8ZZhpu2OrIlLZIadRlxMPRWEiWCf+YVfhlsVsHIb9amBboFxuOJxpwXVCiUzEmhW1GSvRAtRGk5NGRQLMdqLhTIrCcY//JKThWJauwjJmC0+6CwhUvHUNpzhQdYRsLMdcN7PxaNu1YfZ+se/5kYmuUfvTQYdUycmGGTI8KUGqFXZAasXgYO9eS/QjchYfg/PBorutxsXempesafyrQly/k4/OiosyzOLfYqMsKjH0c8ftXJCEZ/hSuZkFU7ZQSIadMXSPQkgGk5fiJEmJAoNhvDbhmNdRddpRcGX6obxCOjREpl8pxkA1XF1/SMw9dtddwRPGrPh8oUmBEd9gltDzqMa4yxz/hrj8NF6NIWXHW7GZyZIjwetUt8jBilTZeqIcMwkwtyOHyRzwxyQHx5ncgg2lpZYDKIXlEGO5wHfhq2QbDGkhxXj7LPpuKDHUdPe3TEN9cgrjeMHSZx60jEWxuqiYwGNaSr0WgmKBCFGEK5xblimnqJTXpQDZdyVyo0M7RVPdtTKxtmwAnNxFqGVNd0bWYcuji3ACOKb9wVi2VaaPeQDrYYrqpYpJqHhIZtNJKJ+zwEPDMDhfQ+q7gkd9cR4/INkhSTyhsnmhaipdAZRqBjWoVRcosHaYFbpGVFO+KA9iFDEJSA38EYyPVDsLk7WovBvAsmFoJEUcOZTEOuMoWqn+oYTMAAyXUmtXxCzPx2P4tq31NMotKnAIobfdA7x2x60YEwupssMXRzXdlCStcHHPCdelwKBicO/4agVXl1PKFyYVEl2SoA1Ss+oqwVVWanS3MrqvVSVeg4AHN7KcUDseVgm0UAGCiONoByviYT5Tdg9eeCRYS0m0xQT5uTlRWHO3kriquLniBcS2FBGdtnwhnIwDtd2ptsnpCsVJkFGTJckcIPNdBRgMG0nOSmXaU3ttsSCZtrg2RVMdApjTVOuWQaocgDeLI9iFe7fOQYM3OHu/iErkldXeVaJKiOfeu8inVG3/gJl3pJ/DcCjuuVF6NQjwuJ6BE3OSsYkeqs1vutEaOmsfAPjATPhSMWwLu+RqK/FqetooyqeKgPejRLWTJ+6HSr3eDPeax/J1ZdCMYwd3sDTFdAJXctDVY9d8KcgfIET0ehAPK2d9Fo6fqH98V+4WJ77bEJ+PMVGPAp1ApVuLg966xSzRh6Zjw4VyA+ItZpTiZPyXiPgKmDFurLhQImMNIJC9XJcgAZvlsC7fdPhnRvNduD3TueE8sEpMkt5UuZQcvJUbY7Etq3IgU3YPOpaIlF+9nME3gorxeEI0ePPjZWVlwihRkPJuEjXDQg3fxxPia64FZaOAkMw6RVpucQ51MzBSJBybe8dPum8Cdi7bpyDdKnvvDI3TqPFhYmLSo+MIpVHtrNvQeryQmax+CN6WOGlDRftKApkrwUvkGmdkte6fdeEEFQH2RlMmRmJiJIHKMoFCw2iKQAclqZfHMKEyFWE8wWceQV2LvHeQEy3RbM21QALCqTYdkrn4dcx5QX/tssHIglSWeutK1hTKrbAn2yyVlmSfVxKfqiE48cYQSXLbk='

if [[ ! -x "$NVCLIENT" ]]; then
  echo "ERROR: nvclient not found/executable at $NVCLIENT" >&2
  exit 1
fi

echo "Click into the browser address bar (or focused field) now..."
sleep 3

# Best-effort fix for hardened arm_token perms (Linux/macOS only).
if [[ -f "$ARM_TOKEN_FILE" ]]; then
  chmod 600 "$ARM_TOKEN_FILE" 2>/dev/null || true
fi

# Helper to send a payload using nvclient
send_payload () {
  local payload="$1"
  "$NVCLIENT" \
    -addr "$SERVER_ADDR" \
    -device-id "$DEVICE_ID" \
    -password "$payload" \
    -key-hex "$KEY_HEX" \
    -server-kyber-pub-b64 "$SERVER_KYBER_PUB_B64"
}

# Check if arm API is listening (fast TCP connect). If yes, arm first.
if timeout 1 bash -c "cat < /dev/null > /dev/tcp/127.0.0.1/60769" 2>/dev/null; then
  echo "[+] Arm API detected at $ARM_ADDR"

  if [[ ! -f "$ARM_TOKEN_FILE" ]]; then
    echo "ERROR: $ARM_TOKEN_FILE not found. Start novakey with arm_api_enabled:true so it auto-generates the token." >&2
    exit 1
  fi

  echo "[+] Arming for ${ARM_MS}ms..."
  "$NVCLIENT" arm --addr "$ARM_ADDR" --token_file "$ARM_TOKEN_FILE" --ms "$ARM_MS" || {
    echo "ERROR: arming failed" >&2
    exit 1
  }
else
  echo "[-] Arm API not detected at $ARM_ADDR (continuing without arming)"
fi

# TWO-MAN: send approval control message first (if enabled on server, this is required)
echo "[+] Sending TWO-MAN approval control payload..."
send_payload "$APPROVE_MAGIC" || {
  echo "ERROR: approval payload send failed" >&2
  exit 1
}

# Small delay is fine; must still be within approve_window_ms (15s in your config)
sleep 0.2

echo "[+] Sending encrypted password frame to $SERVER_ADDR..."
send_payload "$PASSWORD"
echo "[+] Done."

