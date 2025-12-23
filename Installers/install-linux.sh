#!/bin/bash
set -euo pipefail

# NovaKey Linux installer (recommended: systemd *user* service so GUI injection works)
# - Installs binary to /usr/local/bin
# - Installs config + runtime files under the target user's home
# - Creates a systemd user unit: ~/.config/systemd/user/novakey.service
# - Enables firewalld service rule (optional)

SERVICE_NAME="novakey"

BIN_SRC="./dist/novakey-linux-amd64"
BIN_DST="/usr/local/bin/novakey-linux-amd64"

CONFIG_YAML_SRC="./server_config.yaml"
DEVICES_JSON_SRC="./devices.json" # optional; if absent, daemon will show QR on first start

LISTEN_PORT="60768"
FIREWALL_SERVICE_NAME="novakey"
FW_FILE="/etc/firewalld/services/${FIREWALL_SERVICE_NAME}.xml"

echo "[*] Installing NovaKey (Linux) as a systemd *user* service"

if [[ $EUID -ne 0 ]]; then
  echo "[!] Please run as root (sudo)"
  exit 1
fi

TARGET_USER="${SUDO_USER:-}"
if [[ -z "${TARGET_USER}" || "${TARGET_USER}" == "root" ]]; then
  echo "[!] This installer must be run via sudo from the user account that will run NovaKey."
  echo "    Example: sudo ./Installers/install-linux.sh"
  exit 1
fi

TARGET_UID="$(id -u "$TARGET_USER")"
TARGET_HOME="$(getent passwd "$TARGET_USER" | cut -d: -f6)"

if [[ -z "${TARGET_HOME}" || ! -d "${TARGET_HOME}" ]]; then
  echo "[!] Could not resolve home directory for user: $TARGET_USER"
  exit 1
fi

if [[ ! -f "$BIN_SRC" ]]; then
  echo "[!] $BIN_SRC binary not found"
  exit 1
fi

if [[ ! -f "$CONFIG_YAML_SRC" ]]; then
  echo "[!] server_config.yaml not found in current directory"
  exit 1
fi

# ---- Per-user install locations ----
USER_CONFIG_DIR="${TARGET_HOME}/.config/novakey"
USER_SYSTEMD_DIR="${TARGET_HOME}/.config/systemd/user"
USER_DATA_DIR="${TARGET_HOME}/.local/share/novakey"

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

echo "[*] Target user : $TARGET_USER"
echo "[*] Home        : $TARGET_HOME"
echo "[*] Data dir    : $USER_DATA_DIR"
echo "[*] Config dir  : $USER_CONFIG_DIR"
echo "[*] log_dir     : $LOG_DIR_RAW -> $LOG_DIR_ABS"

# ---- Install binary ----
echo "[*] Installing binary"
install -m 755 "$BIN_SRC" "$BIN_DST"

# ---- Create per-user dirs ----
echo "[*] Creating per-user directories"
install -d -m 0700 -o "$TARGET_USER" -g "$TARGET_USER" "$USER_DATA_DIR"
install -d -m 0700 -o "$TARGET_USER" -g "$TARGET_USER" "$USER_CONFIG_DIR"
install -d -m 0700 -o "$TARGET_USER" -g "$TARGET_USER" "$USER_SYSTEMD_DIR"
install -d -m 0700 -o "$TARGET_USER" -g "$TARGET_USER" "$LOG_DIR_ABS"

# ---- Install config ----
echo "[*] Installing config to user profile"

# Keep a "config copy" (editable location)...
install -m 0600 -o "$TARGET_USER" -g "$TARGET_USER" "$CONFIG_YAML_SRC" "$USER_CONFIG_DIR/server_config.yaml"

# ...and a "runtime copy" in WorkingDirectory so relative paths resolve and fallback lookup succeeds.
install -m 0600 -o "$TARGET_USER" -g "$TARGET_USER" "$CONFIG_YAML_SRC" "$USER_DATA_DIR/server_config.yaml"

# devices.json (optional): do NOT create if absent (daemon will show QR on first start)
# Install into DATA_DIR (runtime) so the daemon finds it when devices_file is relative (e.g. "devices.json").
if [[ -f "$DEVICES_JSON_SRC" ]]; then
  install -m 0600 -o "$TARGET_USER" -g "$TARGET_USER" "$DEVICES_JSON_SRC" "$USER_DATA_DIR/devices.json"
  install -m 0600 -o "$TARGET_USER" -g "$TARGET_USER" "$DEVICES_JSON_SRC" "$USER_CONFIG_DIR/devices.json"
else
  rm -f "$USER_DATA_DIR/devices.json" 2>/dev/null || true
  rm -f "$USER_CONFIG_DIR/devices.json" 2>/dev/null || true
fi

# ---- Write systemd user unit ----
USER_UNIT="${USER_SYSTEMD_DIR}/${SERVICE_NAME}.service"
echo "[*] Writing systemd user unit: $USER_UNIT"

cat > "$USER_UNIT" <<EOF
[Unit]
Description=NovaKey Secure Typing Service (user session)
After=graphical-session.target network-online.target
Wants=graphical-session.target network-online.target

[Service]
Type=simple
WorkingDirectory=$USER_DATA_DIR

# Ensure log_dir exists
ExecStartPre=/usr/bin/mkdir -p $LOG_DIR_ABS

# Use the runtime config in WorkingDirectory so relative file paths resolve
ExecStart=$BIN_DST --config $USER_DATA_DIR/server_config.yaml

Restart=on-failure
RestartSec=1

# Hardening (safe for user services)
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=false
LockPersonality=true
MemoryDenyWriteExecute=true

# Allow writes only where needed
ReadWritePaths=$USER_DATA_DIR $LOG_DIR_ABS $USER_CONFIG_DIR

[Install]
WantedBy=default.target
EOF

chown "$TARGET_USER:$TARGET_USER" "$USER_UNIT"
chmod 0644 "$USER_UNIT"

# ---- Firewall (optional) ----
if command -v firewall-cmd >/dev/null 2>&1; then
  echo "[*] Setting firewalld service rule (if available)"
  install -d -m 0755 /etc/firewalld/services

  PAIR_PORT=$((LISTEN_PORT + 2))

  cat > "$FW_FILE" <<EOF
<?xml version="1.0" encoding="utf-8"?>
<service>
    <short>novakey</short>
    <description>NovaKey Service</description>
    <port protocol="tcp" port="$LISTEN_PORT"/>
    <port protocol="tcp" port="$PAIR_PORT"/>
</service>
EOF
  chmod 0644 "$FW_FILE"

  firewall-cmd --reload || true

  # Only add if not already enabled (idempotent)
  if ! firewall-cmd --permanent --query-service="$FIREWALL_SERVICE_NAME" >/dev/null 2>&1; then
    firewall-cmd --permanent --add-service="$FIREWALL_SERVICE_NAME"
  fi

  firewall-cmd --reload || true
fi

# ---- Enable/start user service ----
echo "[*] Enabling user service (requires an active user session)"

USER_RUNTIME_DIR="/run/user/$TARGET_UID"
USER_BUS_ADDR="unix:path=${USER_RUNTIME_DIR}/bus"

# Try to enable/start immediately if user systemd is reachable; otherwise leave installed for next login.
set +e
sudo -u "$TARGET_USER" \
  XDG_RUNTIME_DIR="$USER_RUNTIME_DIR" \
  DBUS_SESSION_BUS_ADDRESS="$USER_BUS_ADDR" \
  systemctl --user daemon-reload

sudo -u "$TARGET_USER" \
  XDG_RUNTIME_DIR="$USER_RUNTIME_DIR" \
  DBUS_SESSION_BUS_ADDRESS="$USER_BUS_ADDR" \
  systemctl --user reset-failed "$SERVICE_NAME" >/dev/null 2>&1

sudo -u "$TARGET_USER" \
  XDG_RUNTIME_DIR="$USER_RUNTIME_DIR" \
  DBUS_SESSION_BUS_ADDRESS="$USER_BUS_ADDR" \
  systemctl --user enable --now "$SERVICE_NAME" 2>/dev/null

START_RC=$?
set -e

echo
echo "[✓] NovaKey installed"
echo "    Binary  : $BIN_DST"
echo "    Config  : $USER_CONFIG_DIR/server_config.yaml"
echo "    Runtime : $USER_DATA_DIR/server_config.yaml"
echo "    Data    : $USER_DATA_DIR"
echo "    Logs    : $LOG_DIR_ABS"
echo "    Unit    : $USER_UNIT"
echo

if [[ $START_RC -ne 0 ]]; then
  echo "[!] Could not start the user service right now (user session/bus not reachable)."
  echo "    Log in to the desktop as $TARGET_USER, then run:"
  echo "      systemctl --user daemon-reload"
  echo "      systemctl --user enable --now $SERVICE_NAME"
  echo
  echo "    To start at boot (even before login), enable linger (optional):"
  echo "      sudo loginctl enable-linger $TARGET_USER"
  echo
else
  echo "[✓] User service enabled and started"
  sudo -u "$TARGET_USER" \
    XDG_RUNTIME_DIR="$USER_RUNTIME_DIR" \
    DBUS_SESSION_BUS_ADDRESS="$USER_BUS_ADDR" \
    systemctl --user status "$SERVICE_NAME" --no-pager -l || true
fi

