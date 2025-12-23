#!/usr/bin/env bash
set -euo pipefail

# NovaKey macOS installer (LaunchAgent, per-user)
# - Installs binary + config under the target user's home
# - Creates a LaunchAgent: ~/Library/LaunchAgents/com.osbornepro.novakey.plist
# - Runs as the logged-in user (so UI automation / focused typing can work)
# - Does NOT create devices.json if missing (pairing bootstrap should handle it)

SERVICE_LABEL="com.osbornepro.novakey"
PLIST_NAME="${SERVICE_LABEL}.plist"

BIN_SRC="./dist/novakey-darwin-amd64"     # adjust if your mac build name differs
CONFIG_YAML_SRC="./server_config.yaml"
DEVICES_JSON_SRC="./devices.json"        # optional; if absent, daemon will bootstrap pairing

if [[ ! -f "$BIN_SRC" ]]; then
  echo "[!] Binary not found: $BIN_SRC"
  exit 1
fi

if [[ ! -f "$CONFIG_YAML_SRC" ]]; then
  echo "[!] Config not found: $CONFIG_YAML_SRC"
  exit 1
fi

TARGET_USER="${SUDO_USER:-$USER}"
TARGET_HOME="$(dscl . -read /Users/"$TARGET_USER" NFSHomeDirectory 2>/dev/null | awk '{print $2}')"

if [[ -z "${TARGET_HOME}" || ! -d "${TARGET_HOME}" ]]; then
  echo "[!] Could not resolve home directory for user: $TARGET_USER"
  exit 1
fi

USER_LA_DIR="${TARGET_HOME}/Library/LaunchAgents"
USER_CONFIG_DIR="${TARGET_HOME}/.config/novakey"
USER_DATA_DIR="${TARGET_HOME}/.local/share/novakey"
USER_BIN_DIR="${TARGET_HOME}/.local/bin"

# Parse log_dir from YAML (best-effort). Relative paths resolve under WorkingDirectory (= USER_DATA_DIR)
LOG_DIR_RAW="$(awk -F: '
  $1 ~ /^[[:space:]]*log_dir[[:space:]]*$/ {
    v=$2
    sub(/#.*/,"",v)
    gsub(/^[[:space:]]+|[[:space:]]+$/,"",v)
    gsub(/^"/,"",v); gsub(/"$/,"",v)
    print v
    exit
  }' "$CONFIG_YAML_SRC" || true)"
LOG_DIR_RAW="${LOG_DIR_RAW:-./logs}"

if [[ "$LOG_DIR_RAW" = /* ]]; then
  LOG_DIR_ABS="$LOG_DIR_RAW"
else
  LOG_DIR_ABS="${USER_DATA_DIR}/${LOG_DIR_RAW#./}"
fi

BIN_DST="${USER_BIN_DIR}/novakey"

echo "[*] Installing NovaKey (macOS) as a per-user LaunchAgent"
echo "[*] Target user : $TARGET_USER"
echo "[*] Home        : $TARGET_HOME"
echo "[*] Binary      : $BIN_DST"
echo "[*] Config dir  : $USER_CONFIG_DIR"
echo "[*] Data dir    : $USER_DATA_DIR"
echo "[*] Logs        : $LOG_DIR_RAW -> $LOG_DIR_ABS"

# Create per-user dirs
mkdir -p "$USER_LA_DIR" "$USER_CONFIG_DIR" "$USER_DATA_DIR" "$USER_BIN_DIR" "$LOG_DIR_ABS"
chown -R "$TARGET_USER:staff" "$USER_LA_DIR" "$USER_CONFIG_DIR" "$USER_DATA_DIR" "$USER_BIN_DIR" || true

chmod 700 "$USER_CONFIG_DIR" "$USER_DATA_DIR" "$USER_BIN_DIR" || true
chmod 755 "$USER_LA_DIR" || true
chmod 700 "$LOG_DIR_ABS" || true

# Install binary
install -m 755 "$BIN_SRC" "$BIN_DST"
chown "$TARGET_USER:staff" "$BIN_DST" || true

# Install config
install -m 600 "$CONFIG_YAML_SRC" "$USER_CONFIG_DIR/server_config.yaml"
chown "$TARGET_USER:staff" "$USER_CONFIG_DIR/server_config.yaml" || true

# devices.json: install if present; otherwise do NOT create it
if [[ -f "$DEVICES_JSON_SRC" ]]; then
  install -m 600 "$DEVICES_JSON_SRC" "$USER_CONFIG_DIR/devices.json"
  chown "$TARGET_USER:staff" "$USER_CONFIG_DIR/devices.json" || true
else
  rm -f "$USER_CONFIG_DIR/devices.json" 2>/dev/null || true
fi

PLIST_PATH="${USER_LA_DIR}/${PLIST_NAME}"

# Write LaunchAgent plist
cat > "$PLIST_PATH" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
 "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>$SERVICE_LABEL</string>

  <key>ProgramArguments</key>
  <array>
    <string>$BIN_DST</string>
    <string>--config</string>
    <string>$USER_CONFIG_DIR/server_config.yaml</string>
  </array>

  <key>WorkingDirectory</key>
  <string>$USER_DATA_DIR</string>

  <key>RunAtLoad</key>
  <true/>

  <key>KeepAlive</key>
  <true/>

  <key>StandardOutPath</key>
  <string>$LOG_DIR_ABS/out.log</string>

  <key>StandardErrorPath</key>
  <string>$LOG_DIR_ABS/err.log</string>

  <key>ThrottleInterval</key>
  <integer>2</integer>
</dict>
</plist>
EOF

chown "$TARGET_USER:staff" "$PLIST_PATH" || true
chmod 644 "$PLIST_PATH"

echo "[*] Loading LaunchAgent"
if [[ "$TARGET_USER" == "$USER" ]]; then
  launchctl unload "$PLIST_PATH" >/dev/null 2>&1 || true
  launchctl load "$PLIST_PATH"
else
  sudo -u "$TARGET_USER" launchctl unload "$PLIST_PATH" >/dev/null 2>&1 || true
  sudo -u "$TARGET_USER" launchctl load "$PLIST_PATH"
fi

echo
echo "[✓] NovaKey installed (per-user)"
echo "    User   : $TARGET_USER"
echo "    Bin    : $BIN_DST"
echo "    Config : $USER_CONFIG_DIR/server_config.yaml"
echo "    Data   : $USER_DATA_DIR"
echo "    Logs   : $LOG_DIR_ABS"
echo "    Agent  : $PLIST_PATH"
echo
echo "IMPORTANT (macOS typing permissions):"
echo "  System Settings → Privacy & Security → Accessibility"
echo "  System Settings → Privacy & Security → Input Monitoring"
echo
echo "To tail logs:"
echo "  tail -f \"$LOG_DIR_ABS/out.log\""
echo "  tail -f \"$LOG_DIR_ABS/err.log\""

