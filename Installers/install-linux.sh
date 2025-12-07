#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME="novakey"
SERVICE_USER="novakey"

BIN_SRC="./novakey-service"
BIN_DST="/usr/local/bin/novakey-service"

CONFIG_DIR="/etc/novakey"
DATA_DIR="/var/lib/novakey"
LOG_DIR="/var/log/novakey"

SYSTEMD_UNIT="/etc/systemd/system/novakey.service"

echo "[*] Installing NovaKey (Linux)"

if [[ $EUID -ne 0 ]]; then
  echo "[!] Please run as root"
  exit 1
fi

if [[ ! -f "$BIN_SRC" ]]; then
  echo "[!] novakey-service binary not found in current directory"
  exit 1
fi

echo "[*] Creating service user (if needed)"
if ! id "$SERVICE_USER" &>/dev/null; then
  useradd \
    --system \
    --no-create-home \
    --shell /usr/sbin/nologin \
    "$SERVICE_USER"
fi

echo "[*] Installing binary"
install -m 755 "$BIN_SRC" "$BIN_DST"

echo "[*] Creating directories"
mkdir -p "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"

chown -R "$SERVICE_USER:$SERVICE_USER" \
  "$DATA_DIR" "$LOG_DIR"

chmod 700 "$DATA_DIR" "$LOG_DIR"

echo "[*] Writing systemd unit"
cat > "$SYSTEMD_UNIT" <<EOF
[Unit]
Description=NovaKey Secure Typing Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
ExecStart=$BIN_DST
Restart=on-failure

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$DATA_DIR $LOG_DIR
CapabilityBoundingSet=
AmbientCapabilities=
LockPersonality=true
MemoryDenyWriteExecute=true

[Install]
WantedBy=multi-user.target
EOF

chmod 644 "$SYSTEMD_UNIT"

echo "[*] Enabling and starting service"
systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"

echo "[✓] NovaKey installed and running"
