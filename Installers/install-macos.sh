#!/usr/bin/env bash
set -euo pipefail

SERVICE_LABEL="com.osbornepro.novakey"

BIN_SRC="./novakey-service"
BIN_DST="/usr/local/bin/novakey-service"

APP_SUPPORT_DIR="/Library/Application Support/NovaKey"
LOG_DIR="$APP_SUPPORT_DIR/logs"

PLIST="/Library/LaunchDaemons/$SERVICE_LABEL.plist"

echo "[*] Installing NovaKey (macOS)"

if [[ $EUID -ne 0 ]]; then
  echo "[!] Please run with sudo"
  exit 1
fi

if [[ ! -f "$BIN_SRC" ]]; then
  echo "[!] novakey-service binary not found in current directory"
  exit 1
fi

echo "[*] Installing binary"
install -m 755 "$BIN_SRC" "$BIN_DST"

echo "[*] Creating directories"
mkdir -p "$LOG_DIR"
chown -R root:wheel "$APP_SUPPORT_DIR"
chmod 755 "$APP_SUPPORT_DIR"
chmod 755 "$LOG_DIR"

echo "[*] Writing LaunchDaemon plist"
cat > "$PLIST" <<EOF
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
  </array>

  <key>RunAtLoad</key>
  <true/>

  <key>KeepAlive</key>
  <true/>

  <key>StandardOutPath</key>
  <string>$LOG_DIR/out.log</string>

  <key>StandardErrorPath</key>
  <string>$LOG_DIR/err.log</string>
</dict>
</plist>
EOF

chown root:wheel "$PLIST"
chmod 644 "$PLIST"

echo "[*] Loading service"
launchctl unload "$PLIST" >/dev/null 2>&1 || true
launchctl load "$PLIST"

echo
echo "[✓] NovaKey installed and running"
echo
echo "IMPORTANT:"
echo "You MUST grant Accessibility + Input Monitoring permissions:"
echo "System Settings → Privacy & Security → Accessibility"
echo "System Settings → Privacy & Security → Input Monitoring"
echo
