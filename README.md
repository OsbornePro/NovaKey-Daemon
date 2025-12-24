# 🔐 NovaKey-Daemon

The **NovaKey** iOS application (*not yet publicly available*) pairs securely with this daemon. When `devices.json` does not exist on first start, the daemon automatically enters pairing mode and generates a QR code (`novakey-pair.png`) for easy device onboarding.

The recommended installation method uses the provided installer scripts. These set up **per-user** services (*systemd user unit on Linux, Scheduled Task on Windows*), which is essential for reliable GUI interaction and keystroke injection in logged-in desktop sessions.

> **macOS Note (as of December 2025):** Full support is still in progress. The daemon runs on macOS, but actual injection from a real iOS device may be blocked by system restrictions during development/testing. The Xcode iOS simulator works correctly. A dedicated macOS installer is not yet provided.

| File / Script                  | Status   | Notes                                      |
|--------------------------------|----------|--------------------------------------------|
| build.ps1                      | Working  |                                            |
| build.sh                       | Working  |                                            |
| Installers/install-linux.sh    | Working  | Recommended for Linux                      |
| Installers/install-windows.ps1 | Working  | Recommended for Windows                    |
| Installers/install-macos.sh    | Not available | macOS installer in progress             |

**NovaKey-Daemon** is a lightweight, cross-platform Go agent that turns your computer into a secure, authenticated password-injection endpoint.

It’s designed for a world where you don’t want to type high-value secrets (*master passwords, recovery keys, etc.*) on your desktop keyboard:
* The secret lives on a trusted device (*e.g. your phone*).
* Delivery is encrypted and authenticated.
* The daemon injects into the currently focused text field.

> **Key point:** Secrets do not traverse the network in plaintext.
> **Status note:** NovaKey targets normal logged-in desktop sessions (*browser fields, terminals, editors, etc*). Login screens / lock screens are future/experimental and not guaranteed.

---
## Security Review Invited
NovaKey-Daemon v3 uses **ML-KEM-768 + HKDF-SHA-256 + XChaCha20-Poly1305** with per-device secrets, freshness checks, replay protection, and per-device rate limiting.

In addition, NovaKey supports safety controls:
* arming (“push-to-type”)
* two-man approval gating (per-device approve window)
* injection safety rules (`allow_newlines`, `max_inject_len`)
* target policy allow/deny lists

* Protocol format: `PROTOCOL.md`
* Security model: `SECURITY.md`

We want feedback on:
* protocol framing & crypto schedule
* replay / freshness / rate-limit logic
* injection paths on Windows/macOS/Linux
* safety controls (arming, two-man, newline blocking, target policy)

---
## Table of Contents
* [Overview](#overview)
* [Current Capabilities](#current-capabilities)
* [Installers](#installers)
  * [Linux Installer](#linux-installer)
  * [Windows Installer](#windows-installer)
  * [macOS Installer](#macos-installer)
* [Command-line Tools](#command-line-tools)
* [Configuration Files](#configuration-files)
* [Arming & Two-Man Safety](#arming--two-man-safety)
* [Logging](#logging)
* [Auto-Type Support Notes](#auto-type-support-notes)
* [Build from Source](#build-from-source)
* [Running NovaKey-Daemon](#running-novakey-daemon)
* [Known Issues](#known-issues)
* [License](#license)
* [Contact & Support](#contact--support)

---
## Overview
NovaKey-Daemon (`novakey`) runs on a workstation (*Windows, macOS, Linux*). Clients connect to the daemon’s TCP listener (default `:60768`), send an encrypted payload, and NovaKey:
1. routes by device ID
2. decapsulates ML-KEM ciphertext (per-message shared secret)
3. derives a per-message AEAD key via HKDF
4. decrypts and validates (freshness / replay / rate limit)
5. parses the **inner typed message frame**
6. applies safety policies (arming, two-man, newline blocking, target policy)
7. injects into the currently focused control (or clipboard fallback)

Protocol version is **v3**.

> **Important (v3 framing):**
> The v3 *outer* header `msgType` is **fixed to `1`** so the daemon accepts the frame.
> “Approve vs Inject” is represented by a **typed inner message frame** inside the AEAD plaintext.

## Pairing Process in NovaKey

The **NovaKey** iOS application pairs with the **NovaKey-Daemon** by scanning a QR code that the daemon generates when no devices are yet paired (*i.e., when* `devices.json` *does not exist or is empty*).

### Why a Two-Step Pairing Process?

The daemon’s static ML-KEM-768 public key is approximately 1188 bytes long. A standard QR code cannot reliably store this much data while remaining scannable at typical camera distances, especially with the required error correction.

To solve this while keeping the process simple and secure, NovaKey uses a **hybrid QR + direct TCP fetch** approach:

1. **QR Code (Initial Bootstrap)**
   - The QR code contains a compact JSON blob with:
     - The daemon’s listening address (*IP:port, default* `60768`)
     - A short-lived pairing token (*random nonce*)
     - The device ID suggested for the phone (*optional, user-editable*)
     - A custom URL scheme trigger: `novakey://pair?host=...&token=...`
   - Scanning the QR code opens the NovaKey iOS app and provides it with the exact server address and authentication token needed for the next step.
   - This keeps the QR code small, dense, and highly reliable to scan.

2. **Direct TCP Fetch (Port 60770)**
   - The iOS app immediately opens a plaintext TCP connection to the daemon on **port 60770** (*or 2 ports above the main listener you set as the default port*).
   - It sends the pairing token received from the QR code.
   - The daemon validates the token (*rate-limited and single-use*) and responds with the full ML-KEM-768 public key (*base64-encoded*).
   - The app now has everything required to generate encrypted v3 frames: device ID, 32-byte device secret, server address, and server Kyber public key.

### Benefits of This Design

- **User-friendly**: One quick QR scan starts the process; the rest happens automatically in the background.
- **Reliable scanning**: QR code stays small and robust.
- **Secure**:
  - The public key is sent only to clients that present a valid short-lived token from the QR.
  - Connection is local-network only (no encryption needed for a public key).
  - Rate limiting and token expiration prevent abuse.
- **No manual entry**: Users never have to copy-paste long base64 strings.

### Firewall / Network Notes

The installers automatically open **TCP port 60770** (*in addition to 60768*) in firewalld (*Linux*) and Windows Defender Firewall where possible. 
This port is used **only during pairing** and only responds to valid token requests.

If you are behind a strict firewall, ensure both ports (`60768` and `60770`) are accessible on the local network.

---

## Current Capabilities
| ✅ | Capability |
| - | ------------------------------------------------------------- |
| ✅ | Cross-platform daemon for Linux / macOS / Windows |
| ✅ | ML-KEM-768 + HKDF-SHA-256 + XChaCha20-Poly1305 transport |
| ✅ | Per-device keys in `devices.json` |
| ✅ | Server keys in `server_keys.json` (auto-generated if missing) |
| ✅ | Timestamp freshness validation |
| ✅ | Nonce replay protection |
| ✅ | Per-device rate limiting |
| ✅ | Optional arming gate (“push-to-type”) |
| ✅ | Optional two-man mode (arm + device approval required) |
| ✅ | Local-only Arm API with token auth |
| ✅ | Injection safety (`allow_newlines`, `max_inject_len`) |
| ✅ | Target policy allow/deny lists |
| ✅ | Config via YAML (preferred) or JSON fallback |
| ✅ | Configurable logging to file w/ rotation + redaction |

---

## Installers

### Linux Installer
The `Installers/install-linux.sh` script installs **NovaKey-Daemon** as a **per-user systemd user service** (*recommended for GUI/keystroke injection*).

Run as the target user with `sudo` from the repository root:
```bash
git clone https://github.com/OsbornePro/NovaKey-Daemon.git
cd NovaKey-Daemon
sudo ./Installers/install-linux.sh
```

#### What the installer does
- Installs the binary to `/usr/local/bin/novakey-linux-amd64`
- Creates per-user directories under `~/.config/novakey`, `~/.local/share/novakey`
- Installs `server_config.yaml` in both config and runtime locations
- Optionally installs `devices.json` if present (otherwise pairing mode on first start)
- Resolves and creates the log directory based on `log_dir` in config
- Creates and enables a systemd user unit (`~/.config/systemd/user/novakey.service`)
- Applies basic systemd hardening
- Adds firewalld service rule for port 60768 (*and 60770 for pairing*) if firewalld is available

The service runs in the user's graphical session, which is required for reliable injection.

Manage the service (*after login*):
```bash
systemctl --user status novakey
systemctl --user restart novakey
journalctl --user -u novakey -f
```

To start at boot even before login (*optional*):
```bash
sudo loginctl enable-linger $USER
```

The installer is safe to re-run.

---

### Windows Installer
The `Installers/install-windows.ps1` script installs **NovaKey-Daemon** as a **per-user Scheduled Task** that starts at logon (*interactive session*).

Run from an elevated PowerShell prompt:
```powershell
Set-Location -Path "$env:USERPROFILE\Downloads\"
Expand-Archive -Path "NovaKey-Daemon-main.zip" -DestinationPath .
Set-Location -Path "NovaKey-Daemon-main"
.\Installers\install-windows.ps1
```

#### What the installer does
- Installs everything under `%LOCALAPPDATA%\NovaKey`
- Copies the binary and `server_config.yaml`
- Copies `devices.json` only if present (otherwise triggers pairing mode)
- Creates a Scheduled Task named **NovaKey** that runs at user logon with limited privileges
- Starts the task immediately
- If pairing mode is active, automatically opens the generated `novakey-pair.png`
- Adds Windows Defender Firewall inbound rules for ports 60768 and 60770 (if running as Administrator)

Manage the task:

```powershell
schtasks /Query /TN NovaKey
schtasks /Run /TN NovaKey
schtasks /End /TN NovaKey
```

The installer is safe to re-run (*removes old task and re-adds it*).

---

### macOS Installer
A dedicated macOS installer script is not yet tested (*in progress*).

The daemon can be built and run manually on macOS, but full iOS-to-macOS injection may be restricted during development. 
Accessibility and Input Monitoring permissions are required for keystroke injection.

---

## Command-line Tools

These tools are here to help security testers and developers.

### `novakey` – the daemon
Loads configuration and listens on the configured address.

Linux/macOS example:
```bash
./dist/novakey-linux-amd64 --config ~/.local/share/novakey/server_config.yaml
```

Windows example:
```powershell
.\dist\novakey-windows-amd64.exe --config "$env:LOCALAPPDATA\NovaKey\server_config.yaml"
```

### `nvclient` – reference/test client
Sends v3 frames (inject or approve).

### `nvpair` – device pairing & key management
Generates pairing material.

(See full examples in the original README sections – unchanged.)

---

## Configuration Files
`server_config.yaml` controls listening address, safety features, logging, etc. Relative paths resolve from the daemon's working directory.

---

## Auto-Type Support Notes
- **Linux**: Works best under X11/XWayland; limited on pure Wayland
- **macOS**: Requires Accessibility + Input Monitoring permissions
- **Windows**: Uses safe control messaging with synthetic fallback

---

## Build from Source

```bash
./build.sh -t linux|darwin|windows
```

Windows:
```powershell
.\build.ps1 -Target windows
```

---

## Configuration Files

NovaKey supports YAML (*preferred*) and JSON (*fallback*). If both exist, YAML wins.

### `server_config.yaml`

Core fields:

* `listen_addr`
* `max_payload_len`
* `max_requests_per_min`
* `devices_file`
* `server_keys_file`

Safety fields:

* `allow_newlines` (default false)
* `max_inject_len`
* `arm_enabled`
* `arm_duration_ms`
* `arm_consume_on_inject`
* `allow_clipboard_when_disarmed`
* `two_man_enabled`
* `approve_window_ms`
* `approve_consume_on_inject`

Target policy allow/deny lists:

* `target_policy_enabled`
* `use_built_in_allowlist`
* `allowed_process_names`, `allowed_window_titles`
* `denied_process_names`, `denied_window_titles`

Arm API fields:

* `arm_api_enabled`
* `arm_listen_addr` (must be loopback)
* `arm_token_file`
* `arm_token_header`

Logging fields:

* `log_dir` or `log_file`
* `log_rotate_mb`
* `log_keep`
* `log_stderr`
* `log_redact`

---

## Arming & Two-Man Safety

### Arming gate

When `arm_enabled: true`, NovaKey will decrypt/validate frames but block injection unless armed.

### Two-man mode

When `two_man_enabled: true`, injection requires:

1. host is armed, and
2. device has sent a recent **typed approve** message (inner msgType=2)

---

## Logging
Configurable via `log_dir`, rotation, and redaction options in `server_config.yaml`.

NovaKey also supports logging to a file with rotation:

```yaml
log_dir: "./logs"        # or "~/.config/share/novakey/logs/"
log_rotate_mb: 10
log_keep: 10
log_stderr: true
log_redact: true
```

**Notes:**

* Passwords are never logged in full (*only a short preview until done with development*).
* With `log_redact: true`, NovaKey redacts configured tokens (*if available*), long blob-like strings, and obvious secret patterns.

---

## Auto-Type Support Notes

NovaKey targets “normal” apps and text fields. Some secure desktops and elevated contexts may block injection.

* **Linux**

  * X11 / XWayland: keystroke injection can work (focused detection via X11 tooling)
  * Wayland native: focused-app detection / typing may be limited; clipboard fallback may be used
* **macOS**

  * Requires Accessibility permissions for automation paths
* **Windows**

  * Uses safe control messaging when possible; falls back to synthetic typing

---

## Build from Source

Requirements:

* Go (*I am using go 1.25.5 with 1.21+ recommended*)
* Standard Go toolchain

Build:

```bash
./build.sh -t linux
./build.sh -t darwin
./build.sh -t windows
```

Windows build script:

```powershell
.\build.ps1 -Target windows -Clean
```

---

## Running NovaKey-Daemon

1. Ensure config exists (`server_config.yaml` preferred).

2. Start daemon:

```bash
./dist/novakey-linux-amd64
```

3. Pair a device:

```bash
./dist/nvpair -id phone
```

4. Focus a text field and test injection:

```bash
./dist/nvclient \
  -addr 127.0.0.1:60768 \
  -device-id phone \
  -key-hex <device_key_hex> \
  -server-kyber-pub-b64 "<server_kyber768_public>" \
  -password "TestPassword123!"
```

5. If `two_man_enabled: true`, approve then inject:

```bash
./dist/nvclient approve \
  -addr 127.0.0.1:60768 \
  -device-id phone \
  -key-hex <device_key_hex> \
  -server-kyber-pub-b64 "<server_kyber768_public>"

./dist/nvclient \
  -addr 127.0.0.1:60768 \
  -device-id phone \
  -key-hex <device_key_hex> \
  -server-kyber-pub-b64 "<server_kyber768_public>" \
  -password "TestPassword123!"
```

---

## Running NovaKey-Daemon
Use the installers for best experience. 
Manual runs are possible but require correct working directory for relative paths.

---

## Known Issues
- Pure Wayland sessions on Linux: limited injection support
- macOS: real-device injection may be blocked in development phase

---

## License
Apache License, Version 2.0 – see `LICENSE.md`.

---

## Contact & Support
* Support: `security@novakey.app`
* Security disclosures: see `SECURITY.md`

Thank you for trying NovaKey!
