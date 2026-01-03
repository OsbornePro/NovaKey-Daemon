#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME="novakey.service"
USER_SYSTEMD_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
SERVICE_FILE="$USER_SYSTEMD_DIR/$SERVICE_NAME"
WANTS_LINK="$USER_SYSTEMD_DIR/default.target.wants/$SERVICE_NAME"

CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/novakey"
DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/novakey"

BIN_PATH="/usr/local/bin/novakey-linux-amd64.elf"

log() { printf '[novakey-uninstall] %s\n' "$*"; }

have() { command -v "$1" >/dev/null 2>&1; }

remove_path() {
  local p="$1"
  if [[ -e "$p" || -L "$p" ]]; then
    log "Removing: $p"
    rm -rf -- "$p"
  else
    log "Already gone: $p"
  fi
}

log "Starting uninstall (user: $USER, home: $HOME)"

if have systemctl; then
  # Stop/disable are fine even if the unit doesn't exist; don't fail uninstall for that.
  log "Stopping user service (if running)..."
  systemctl --user stop "$SERVICE_NAME" 2>/dev/null || true

  log "Disabling user service (if enabled)..."
  systemctl --user disable "$SERVICE_NAME" 2>/dev/null || true
else
  log "systemctl not found; skipping service stop/disable"
fi

# Remove unit file + wants symlink(s)
remove_path "$SERVICE_FILE"
remove_path "$WANTS_LINK"

# Reload user systemd
if have systemctl; then
  log "Reloading user systemd daemon..."
  systemctl --user daemon-reload 2>/dev/null || true
  systemctl --user reset-failed 2>/dev/null || true
fi

# Remove app config/data
remove_path "$CONFIG_DIR"
remove_path "$DATA_DIR"

# Remove binary (needs sudo)
if [[ -e "$BIN_PATH" ]]; then
  log "Removing binary: $BIN_PATH (requires sudo)"
  sudo rm -f -- "$BIN_PATH"
else
  log "Binary not found: $BIN_PATH"
fi

log "Uninstall complete."

