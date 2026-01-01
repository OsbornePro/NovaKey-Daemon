#!/usr/bin/env bash
set -euo pipefail

# NovaKey macOS uninstaller (LaunchAgent, per-user)
# - Stops/unloads the per-user LaunchAgent
# - Removes LaunchAgent plist
# - Removes installed binary
# - Removes config + runtime data dirs
# - Removes logs directory derived from the installed config (best-effort)
# - Attempts to remove additional common runtime files in ~/.local/share/novakey

SERVICE_LABEL="com.osbornepro.novakey"
PLIST_NAME="${SERVICE_LABEL}.plist"

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

BIN_DST="${USER_BIN_DIR}/novakey"
PLIST_PATH="${USER_LA_DIR}/${PLIST_NAME}"

# Best-effort: derive log_dir from the runtime config that the LaunchAgent uses
RUNTIME_CFG="${USER_DATA_DIR}/server_config.yaml"
EDIT_CFG="${USER_CONFIG_DIR}/server_config.yaml"

LOG_DIR_RAW=""
if [[ -f "$RUNTIME_CFG" ]]; then
  LOG_DIR_RAW="$(awk -F: '
    $1 ~ /^[[:space:]]*log_dir[[:space:]]*$/ {
      v=$2
      sub(/#.*/,"",v)
      gsub(/^[[:space:]]+|[[:space:]]+$/,"",v)
      gsub(/^"/,"",v); gsub(/"$/,"",v)
      print v
      exit
    }' "$RUNTIME_CFG" || true)"
elif [[ -f "$EDIT_CFG" ]]; then
  LOG_DIR_RAW="$(awk -F: '
    $1 ~ /^[[:space:]]*log_dir[[:space:]]*$/ {
      v=$2
      sub(/#.*/,"",v)
      gsub(/^[[:space:]]+|[[:space:]]+$/,"",v)
      gsub(/^"/,"",v); gsub(/"$/,"",v)
      print v
      exit
    }' "$EDIT_CFG" || true)"
fi
LOG_DIR_RAW="${LOG_DIR_RAW:-./logs}"

if [[ "$LOG_DIR_RAW" = /* ]]; then
  LOG_DIR_ABS="$LOG_DIR_RAW"
else
  LOG_DIR_ABS="${USER_DATA_DIR}/${LOG_DIR_RAW#./}"
fi

echo "[*] Uninstalling NovaKey-Daemon (macOS) per-user LaunchAgent"
echo "[*] Target user : $TARGET_USER"
echo "[*] Home        : $TARGET_HOME"
echo "[*] Agent label : $SERVICE_LABEL"
echo "[*] Plist       : $PLIST_PATH"
echo "[*] Binary      : $BIN_DST"
echo "[*] Config dir  : $USER_CONFIG_DIR"
echo "[*] Data dir    : $USER_DATA_DIR"
echo "[*] Logs (best-effort) : $LOG_DIR_RAW -> $LOG_DIR_ABS"
echo

DOMAIN="gui/$TARGET_UID"

stop_agent() {
  # Try to stop/kickstart then bootout, ignore failures (agent may not be loaded)
  launchctl kill SIGTERM "$DOMAIN/$SERVICE_LABEL" >/dev/null 2>&1 || true
  launchctl kickstart -k "$DOMAIN/$SERVICE_LABEL" >/dev/null 2>&1 || true
  launchctl bootout "$DOMAIN" "$PLIST_PATH" >/dev/null 2>&1 || true

  # If plist already gone, still try bootout by label (best-effort)
  launchctl bootout "$DOMAIN/$SERVICE_LABEL" >/dev/null 2>&1 || true
}

echo "[*] Stopping/unloading LaunchAgent"
if [[ "$TARGET_USER" == "$USER" ]]; then
  stop_agent
else
  sudo -u "$TARGET_USER" bash -c "$(declare -f stop_agent); stop_agent"
fi

echo "[*] Removing LaunchAgent plist"
rm -f "$PLIST_PATH" 2>/dev/null || true

echo "[*] Removing installed binary"
rm -f "$BIN_DST" 2>/dev/null || true

echo "[*] Removing NovaKey config + data directories"
rm -rf "$USER_CONFIG_DIR" 2>/dev/null || true
rm -rf "$USER_DATA_DIR" 2>/dev/null || true

echo "[*] Removing logs directory (best-effort)"
# Only remove if it’s inside the NovaKey data dir OR matches common expected path.
# If LOG_DIR_ABS is outside USER_DATA_DIR, we still remove it because the installer supports absolute paths.
if [[ -n "${LOG_DIR_ABS}" && -d "${LOG_DIR_ABS}" ]]; then
  rm -rf "${LOG_DIR_ABS}" 2>/dev/null || true
fi

# Clean up empty parent dirs if they’re now unused (best-effort, safe)
rmdir "$USER_LA_DIR" >/dev/null 2>&1 || true
rmdir "$USER_BIN_DIR" >/dev/null 2>&1 || true
rmdir "${TARGET_HOME}/.config" >/dev/null 2>&1 || true
rmdir "${TARGET_HOME}/.local/share" >/dev/null 2>&1 || true
rmdir "${TARGET_HOME}/.local/bin" >/dev/null 2>&1 || true
rmdir "${TARGET_HOME}/.local" >/dev/null 2>&1 || true

echo
echo "[✓] NovaKey-Daemon removed"
echo "    LaunchAgent: removed/unloaded ($SERVICE_LABEL)"
echo "    Plist      : removed ($PLIST_PATH)"
echo "    Binary     : removed ($BIN_DST)"
echo "    Config dir : removed ($USER_CONFIG_DIR)"
echo "    Data dir   : removed ($USER_DATA_DIR)"
echo "    Logs dir   : removed (best-effort) ($LOG_DIR_ABS)"
echo
echo "NOTE:"
echo "  macOS Accessibility/Input Monitoring permissions are not removed automatically."
echo "  You can remove them manually in System Settings → Privacy & Security."

