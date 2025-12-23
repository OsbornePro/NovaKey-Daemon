#!/usr/bin/env bash
set -euo pipefail

# NovaKey macOS installer (LaunchDaemon, least-privilege user)
# - Creates a dedicated system user/group: novakey
# - Installs binary + config
# - Runs LaunchDaemon as novakey (not root)
# - Restricts writable locations to data/log directories only

SERVICE_LABEL="com.osbornepro.novakey"
PLIST="/Library/LaunchDaemons/${SERVICE_LABEL}.plist"

SERVICE_USER="novakey"
SERVICE_GROUP="novakey"

# ---- Inputs (run from repo root) ----
BIN_SRC="./dist/novakey-darwin-amd64"     # adjust if your mac build name differs
CONFIG_YAML_SRC="./server_config.yaml"
DEVICES_JSON_SRC="./devices.json"        # optional; if absent, daemon will show QR on first start

# ---- Install locations ----
BIN_DST="/usr/local/bin/novakey"

APP_SUPPORT_DIR="/Library/Application Support/NovaKey"
CONFIG_DIR="${APP_SUPPORT_DIR}/config"
DATA_DIR="${APP_SUPPORT_DIR}/data"
DEFAULT_LOG_DIR="${APP_SUPPORT_DIR}/logs"

echo "[*] Installing NovaKey (macOS)"

if [[ $EUID -ne 0 ]]; then
  echo "[!] Please run with sudo"
  exit 1
fi

if [[ ! -f "$BIN_SRC" ]]; then
  echo "[!] Binary not found: $BIN_SRC"
  exit 1
fi

if [[ ! -f "$CONFIG_YAML_SRC" ]]; then
  echo "[!] Config not found: $CONFIG_YAML_SRC"
  exit 1
fi

# ---- Create service user/group (macOS) ----
# Uses dscl (built-in) and chooses a high UID/GID if not present.
ensure_group() {
  local group="$1"
  if dscl . -read "/Groups/$group" >/dev/null 2>&1; then
    return 0
  fi

  local gid
  gid="$(dscl . -list /Groups PrimaryGroupID | awk '{print $2}' | sort -n | tail -1)"
  gid=$((gid + 1))
  if [[ $gid -lt 500 ]]; then gid=501; fi
  if [[ $gid -lt 2000 ]]; then gid=2000; fi

  echo "[*] Creating group: $group (GID $gid)"
  dscl . -create "/Groups/$group"
  dscl . -create "/Groups/$group" PrimaryGroupID "$gid"
  dscl . -create "/Groups/$group" Password '*'
}

ensure_user() {
  local user="$1"
  local group="$2"

  if dscl . -read "/Users/$user" >/dev/null 2>&1; then
    return 0
  fi

  local uid
  uid="$(dscl . -list /Users UniqueID | awk '{print $2}' | sort -n | tail -1)"
  uid=$((uid + 1))
  if [[ $uid -lt 500 ]]; then uid=501; fi
  if [[ $uid -lt 2000 ]]; then uid=2000; fi

  local gid
  gid="$(dscl . -read "/Groups/$group" PrimaryGroupID 2>/dev/null | awk '{print $2}')"
  if [[ -z "${gid:-}" ]]; then
    echo "[!] Could not resolve GID for group $group"
    exit 1
  fi

  echo "[*] Creating user: $user (UID $uid, GID $gid)"
  dscl . -create "/Users/$user"
  dscl . -create "/Users/$user" UniqueID "$uid"
  dscl . -create "/Users/$user" PrimaryGroupID "$gid"
  dscl . -create "/Users/$user" UserShell "/usr/bin/false"
  dscl . -create "/Users/$user" NFSHomeDirectory "/var/empty"
  dscl . -create "/Users/$user" RealName "NovaKey Service User"
  dscl . -create "/Users/$user" Password '*'

  # Ensure group membership
  dscl . -append "/Groups/$group" GroupMembership "$user" || true
}

ensure_group "$SERVICE_GROUP"
ensure_user "$SERVICE_USER" "$SERVICE_GROUP"

# ---- Parse log_dir from server_config.yaml (best-effort) ----
# We set WorkingDirectory to DATA_DIR, so relative log_dir values resolve under DATA_DIR.
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
  LOG_DIR_ABS="$DATA_DIR/${LOG_DIR_RAW#./}"
fi

echo "[*] log_dir from YAML: $LOG_DIR_RAW -> $LOG_DIR_ABS"

# ---- Install binary ----
echo "[*] Installing binary to $BIN_DST"
install -m 755 "$BIN_SRC" "$BIN_DST"
chown root:wheel "$BIN_DST"
chmod 755 "$BIN_DST"

# ---- Create directories ----
echo "[*] Creating directories"
mkdir -p "$CONFIG_DIR" "$DATA_DIR" "$DEFAULT_LOG_DIR" "$LOG_DIR_ABS"

# Ownership/permissions model:
# - /Library/Application Support/NovaKey (root:wheel 755) — readable
# - config dir root:wheel 755 — readable
# - config files root:novakey 640/600 — readable by service group
# - data/log dirs novakey:novakey 700 — writable only by service user
chown root:wheel "$APP_SUPPORT_DIR"
chmod 755 "$APP_SUPPORT_DIR"

chown root:wheel "$CONFIG_DIR"
chmod 755 "$CONFIG_DIR"

chown -R "$SERVICE_USER:$SERVICE_GROUP" "$DATA_DIR"
chmod 700 "$DATA_DIR"

# Use the parsed log dir (could be inside DATA_DIR or absolute elsewhere)
# Ensure it exists and is writable by service user.
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$LOG_DIR_ABS"
chmod 700 "$LOG_DIR_ABS"

# ---- Install config ----
echo "[*] Installing config"
# server_config.yaml: readable by service (group), writable only by root
install -m 640 -o root -g "$SERVICE_GROUP" "$CONFIG_YAML_SRC" "$CONFIG_DIR/server_config.yaml"

# devices.json:
# - If present in repo, install it (root writable; service-readable if group is novakey)
# - If absent, do NOT create it (daemon will show QR on first start)
if [[ -f "$DEVICES_JSON_SRC" ]]; then
  install -m 640 -o root -g "$SERVICE_GROUP" "$DEVICES_JSON_SRC" "$CONFIG_DIR/devices.json"
else
  rm -f "$CONFIG_DIR/devices.json" 2>/dev/null || true
fi

# Optional: copy YAML into DATA_DIR for convenience/debug (not required)
install -m 600 -o "$SERVICE_USER" -g "$SERVICE_GROUP" "$CONFIG_YAML_SRC" "$DATA_DIR/server_config.yaml"

# ---- Write LaunchDaemon plist ----
echo "[*] Writing LaunchDaemon plist: $PLIST"
cat > "$PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
 "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>$SERVICE_LABEL</string>

  <key>UserName</key>
  <string>$SERVICE_USER</string>

  <key>GroupName</key>
  <string>$SERVICE_GROUP</string>

  <key>ProgramArguments</key>
  <array>
    <string>$BIN_DST</string>
    <string>--config</string>
    <string>$CONFIG_DIR/server_config.yaml</string>
  </array>

  <key>WorkingDirectory</key>
  <string>$DATA_DIR</string>

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

chown root:wheel "$PLIST"
chmod 644 "$PLIST"

# ---- Load/Reload service ----
echo "[*] Loading service"
launchctl bootout system "$PLIST" >/dev/null 2>&1 || true
launchctl bootstrap system "$PLIST"
launchctl enable "system/$SERVICE_LABEL" >/dev/null 2>&1 || true
launchctl kickstart -k "system/$SERVICE_LABEL" >/dev/null 2>&1 || true

echo
echo "[✓] NovaKey installed and running"
echo
echo "Service label : $SERVICE_LABEL"
echo "Run as        : $SERVICE_USER:$SERVICE_GROUP"
echo "Binary        : $BIN_DST"
echo "Config        : $CONFIG_DIR/server_config.yaml"
echo "Working dir   : $DATA_DIR"
echo "Logs          : $LOG_DIR_ABS/out.log"
echo "               $LOG_DIR_ABS/err.log"
echo
echo "IMPORTANT (macOS typing permissions):"
echo "  System Settings → Privacy & Security → Accessibility"
echo "  System Settings → Privacy & Security → Input Monitoring"
echo
echo "To view logs:"
echo "  sudo tail -f \"$LOG_DIR_ABS/out.log\""
echo "  sudo tail -f \"$LOG_DIR_ABS/err.log\""

