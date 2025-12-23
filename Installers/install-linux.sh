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

if [[ ! -f "$CONFIG_YAML_SRC" ]]; then
    echo "[!] server_config.yaml not found in current directory"
    exit 1
fi

# ---- Parse log_dir from server_config.yaml (best-effort) ----
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

# Resolve to absolute path. Relative values are relative to WorkingDirectory (= DATA_DIR)
if [[ "$LOG_DIR_RAW" = /* ]]; then
    LOG_DIR_ABS="$LOG_DIR_RAW"
else
    LOG_DIR_ABS="$DATA_DIR/${LOG_DIR_RAW#./}"
fi

echo "[*] log_dir from YAML: $LOG_DIR_RAW -> $LOG_DIR_ABS"

echo "[*] Creating service user (if needed)"
if ! id "$SERVICE_USER" &>/dev/null; then
    useradd --system --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER"
fi

echo "[*] Installing binary"
install -m 755 "$BIN_SRC" "$BIN_DST"

echo "[*] Creating directories"
mkdir -p "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR" "$LOG_DIR_ABS"

chown root:"$SERVICE_USER" "$CONFIG_DIR"
chmod 750 "$CONFIG_DIR"

chown -R "$SERVICE_USER:$SERVICE_USER" "$DATA_DIR" "$LOG_DIR"
chmod 700 "$DATA_DIR" "$LOG_DIR"

# Ensure YAML-defined log dir is writable by service user
chown -R "$SERVICE_USER:$SERVICE_USER" "$LOG_DIR_ABS" || true
chmod 700 "$LOG_DIR_ABS" || true

echo "[*] Installing config file"
install -m 640 -o root -g "$SERVICE_USER" "$CONFIG_YAML_SRC" "$CONFIG_DIR/server_config.yaml"
install -m 600 -o "$SERVICE_USER" -g "$SERVICE_USER" "$CONFIG_YAML_SRC" "$DATA_DIR/server_config.yaml"

# (remove redundant copies; install already did it)
# cp -f server_config.yaml $CONFIG_DIR/server_config.yaml
# cp -f server_config.yaml $DATA_DIR/server_config.yaml

if [[ -f "$DEVICES_JSON_SRC" ]]; then
    install -m 640 -o root -g "$SERVICE_USER" "$DEVICES_JSON_SRC" "$CONFIG_DIR/devices.json"
else
    rm -f -- "$CONFIG_DIR/devices.json" 2>/dev/null || true
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

# Ensure log_dir exists and is writable before starting
ExecStartPre=/usr/bin/mkdir -p $LOG_DIR_ABS
ExecStartPre=/usr/bin/chown -R $SERVICE_USER:$SERVICE_USER $LOG_DIR_ABS

ExecStart=$BIN_DST --config $CONFIG_DIR/server_config.yaml

Restart=on-failure
RestartSec=1

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true

# Allow writes to runtime dirs and configured log dir
ReadWritePaths=$DATA_DIR $CONFIG_DIR $LOG_DIR_ABS

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
