#!/bin/bash
set -euo pipefail

SERVICE_NAME="novakey"
SERVICE_USER="novakey"

BIN_SRC="./dist/novakey-linux-amd64"
BIN_DST="/usr/local/bin/novakey-linux-amd64"

CONFIG_DIR="/etc/novakey"
DATA_DIR="/var/lib/novakey"
LOG_DIR="/var/log/novakey"

SYSTEMD_UNIT="/etc/systemd/system/novakey.service"
CONFIG_YAML_SRC="./server_config.yaml"
DEVICES_JSON_SRC="./devices.json"

echo "[*] Installing NovaKey (Linux)"

if [[ $EUID -ne 0 ]]; then
    echo "[!] Please run as root"
    exit 1
fi

if [[ ! -f "$BIN_SRC" ]]; then
    echo "[!] $BIN_SRC binary not found in current directory"
    exit 1
fi

echo "[*] Creating service user (if needed)"
if ! id "$SERVICE_USER" &>/dev/null; then
    useradd --system --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER"
fi

echo "[*] Installing binary"
install -m 755 "$BIN_SRC" "$BIN_DST"

echo "[*] Creating directories"
mkdir -p "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"

chown root:"$SERVICE_USER" "$CONFIG_DIR"
chmod 750 "$CONFIG_DIR"

chown -R "$SERVICE_USER:$SERVICE_USER" "$DATA_DIR" "$LOG_DIR"
chmod 700 "$DATA_DIR" "$LOG_DIR"

echo "[*] Installing config file"
if [[ -f "$CONFIG_YAML_SRC" ]]; then
    install -m 640 -o root -g "$SERVICE_USER" "$CONFIG_YAML_SRC" "$CONFIG_DIR/server_config.yaml"
    install -m 600 -o "$SERVICE_USER" -g "$SERVICE_USER" "$CONFIG_YAML_SRC" "$DATA_DIR/server_config.yaml"
    cp -f server_config.yaml $CONFIG_DIR/server_config.yaml
    cp -f server_config.yaml $DATA_DIR/server_config.yaml
    chown root:novakey /etc/novakey/server_config.yaml
    chmod 0640 /etc/novakey/server_config.yaml
else
    echo "[!] server_config.yaml not found in current directory"
    exit 1
fi

if [[ -f "$DEVICES_JSON_SRC" ]]; then
    install -m 640 -o root -g "$SERVICE_USER" "$DEVICES_JSON_SRC" "$CONFIG_DIR/devices.json"
else
    rm -rf -- "$CONFIG_DIR/devices.json" 2>/dev/null || true
fi

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

WorkingDirectory=$DATA_DIR
ExecStart=$BIN_DST --config $CONFIG_DIR/server_config.yaml

Restart=on-failure
RestartSec=1

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$DATA_DIR $LOG_DIR $CONFIG_DIR
CapabilityBoundingSet=
AmbientCapabilities=
LockPersonality=true
MemoryDenyWriteExecute=true

[Install]
WantedBy=multi-user.target
EOF

chmod 644 "$SYSTEMD_UNIT"

if command -v firewall-cmd >/dev/null 2>&1; then
    echo "[*] Setting the firewall rule"
    FW_FILE="/etc/firewalld/services/novakey.xml"
    install -d -m 0755 /etc/firewalld/services
    cat > "$FW_FILE" <<'EOF'
<?xml version="1.0" encoding="utf-8"?>
<service>
    <short>novakey</short>
    <description>NovaKey Service</description>
    <port protocol="tcp" port="60768"/>
</service>
EOF
    chmod 0644 "$FW_FILE"

    firewall-cmd --reload || true

    if ! firewall-cmd --permanent --query-service=novakey >/dev/null 2>&1; then
        firewall-cmd --permanent --add-service=novakey
    fi

    firewall-cmd --reload || true
fi

echo "[*] Enabling and starting service"
systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"

echo "[✓] NovaKey installed and running"
systemctl status "$SERVICE_NAME" --no-pager -l

