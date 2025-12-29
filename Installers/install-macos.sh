#!/usr/bin/env bash
set -euo pipefail

# NovaKey macOS installer (LaunchAgent, per-user)
# - Installs binary + config under the target user's home
# - Creates a LaunchAgent: ~/Library/LaunchAgents/com.osbornepro.novakey.plist
# - Runs as the logged-in user (so UI automation / focused typing can work)
# - Keeps an editable config copy in ~/.config/novakey
# - Uses a runtime WorkingDirectory in ~/.local/share/novakey so relative paths resolve (devices.json, server_keys.json, ./logs, etc.)
# - Does NOT create devices.json if missing (pairing bootstrap should handle it)

SERVICE_LABEL="com.osbornepro.novakey"
PLIST_NAME="${SERVICE_LABEL}.plist"

CONFIG_YAML_SRC="./server_config.yaml"
DEVICES_JSON_SRC="./devices.json" # optional

# ---- Choose binary based on arch (Apple Silicon vs Intel) ----
ARCH="$(uname -m)"
if [[ "$ARCH" == "arm64" ]]; then
  BIN_SRC="./dist/novakey-darwin-arm64"
else
  BIN_SRC="./dist/novakey-darwin-amd64"
fi

if [[ ! -f "$BIN_SRC" ]]; then
  echo "[!] Binary not found: $BIN_SRC"
  echo "    (If your build output name differs, update BIN_SRC mapping.)"
  exit 1
fi

if [[ ! -f "$CONFIG_YAML_SRC" ]]; then
  echo "[!] Config not found: $CONFIG_YAML_SRC"
  exit 1
fi

TARGET_USER="${SUDO_USER:-$USER}"
TARGET_UID="$(id -u "$TARGET_USER")"
TARGET_GROUP="$(id -gn "$TARGET_USER")"

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
PLIST_PATH="${USER_LA_DIR}/${PLIST_NAME}"

echo "[*] Installing NovaKey (macOS) as a per-user LaunchAgent"
echo "[*] Target user : $TARGET_USER"
echo "[*] Home        : $TARGET_HOME"
echo "[*] Arch        : $ARCH"
echo "[*] Binary src  : $BIN_SRC"
echo "[*] Binary dst  : $BIN_DST"
echo "[*] Config dir  : $USER_CONFIG_DIR"
echo "[*] Data dir    : $USER_DATA_DIR"
echo "[*] Logs        : $LOG_DIR_RAW -> $LOG_DIR_ABS"
echo "[*] Agent       : $PLIST_PATH"

# ---- Create per-user dirs ----
mkdir -p "$USER_LA_DIR" "$USER_CONFIG_DIR" "$USER_DATA_DIR" "$USER_BIN_DIR" "$LOG_DIR_ABS"

chown -R "$TARGET_USER:$TARGET_GROUP" \
  "$USER_LA_DIR" "$USER_CONFIG_DIR" "$USER_DATA_DIR" "$USER_BIN_DIR" "$LOG_DIR_ABS" || true

chmod 755 "$USER_LA_DIR" || true
chmod 700 "$USER_CONFIG_DIR" "$USER_DATA_DIR" "$USER_BIN_DIR" "$LOG_DIR_ABS" || true

# ---- Install binary ----
install -m 755 "$BIN_SRC" "$BIN_DST"
chown "$TARGET_USER:$TARGET_GROUP" "$BIN_DST" || true

# ---- Install config ----
# Editable copy:
install -m 600 "$CONFIG_YAML_SRC" "$USER_CONFIG_DIR/server_config.yaml"
chown "$TARGET_USER:$TARGET_GROUP" "$USER_CONFIG_DIR/server_config.yaml" || true

# Runtime copy in WorkingDirectory so relative paths resolve (matches Linux installer behavior):
install -m 600 "$CONFIG_YAML_SRC" "$USER_DATA_DIR/server_config.yaml"
chown "$TARGET_USER:$TARGET_GROUP" "$USER_DATA_DIR/server_config.yaml" || true

# ---- devices.json (optional): install to BOTH config + runtime dirs if present; otherwise ensure absent ----
if [[ -f "$DEVICES_JSON_SRC" ]]; then
  install -m 600 "$DEVICES_JSON_SRC" "$USER_CONFIG_DIR/devices.json"
  install -m 600 "$DEVICES_JSON_SRC" "$USER_DATA_DIR/devices.json"
  chown "$TARGET_USER:$TARGET_GROUP" "$USER_CONFIG_DIR/devices.json" "$USER_DATA_DIR/devices.json" || true
else
  rm -f "$USER_CONFIG_DIR/devices.json" "$USER_DATA_DIR/devices.json" 2>/dev/null || true
fi

# ---- Write LaunchAgent plist ----
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
    <string>$USER_DATA_DIR/server_config.yaml</string>
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

chown "$TARGET_USER:$TARGET_GROUP" "$PLIST_PATH" || true
chmod 644 "$PLIST_PATH"

# ---- Load LaunchAgent (modern launchctl) ----
echo "[*] Loading LaunchAgent"
# Note: For LaunchAgents, this should be run in the context of the GUI login session for that user.
DOMAIN="gui/$TARGET_UID"

if [[ "$TARGET_USER" == "$USER" ]]; then
  launchctl bootout "$DOMAIN" "$PLIST_PATH" >/dev/null 2>&1 || true
  launchctl bootstrap "$DOMAIN" "$PLIST_PATH"
  launchctl enable "$DOMAIN/$SERVICE_LABEL" >/dev/null 2>&1 || true
  launchctl kickstart -k "$DOMAIN/$SERVICE_LABEL" >/dev/null 2>&1 || true
else
  sudo -u "$TARGET_USER" launchctl bootout "$DOMAIN" "$PLIST_PATH" >/dev/null 2>&1 || true
  sudo -u "$TARGET_USER" launchctl bootstrap "$DOMAIN" "$PLIST_PATH"
  sudo -u "$TARGET_USER" launchctl enable "$DOMAIN/$SERVICE_LABEL" >/dev/null 2>&1 || true
  sudo -u "$TARGET_USER" launchctl kickstart -k "$DOMAIN/$SERVICE_LABEL" >/dev/null 2>&1 || true
fi

echo
echo "[✓] NovaKey installed (per-user)"
echo "    User    : $TARGET_USER"
echo "    Bin     : $BIN_DST"
echo "    Config  : $USER_CONFIG_DIR/server_config.yaml  (editable)"
echo "    Runtime : $USER_DATA_DIR/server_config.yaml    (used by LaunchAgent)"
echo "    Data    : $USER_DATA_DIR"
echo "    Logs    : $LOG_DIR_ABS"
echo "    Agent   : $PLIST_PATH"
echo
echo "IMPORTANT (macOS typing permissions):"
echo "  System Settings → Privacy & Security → Accessibility"
echo "  System Settings → Privacy & Security → Input Monitoring"
echo
echo "To tail logs:"
echo "  tail -f \"$LOG_DIR_ABS/out.log\""
echo "  tail -f \"$LOG_DIR_ABS/err.log\""

