# 🔐 NovaKey-Daemon by OsbornePro

### If you try downloading this and something does not work at the moment bare with me as I am writing the install scripts for Windows and MacOS still. The Linunx install script is written, tested, and verified works

**NovaKey-Daemon** is a lightweight, cross-platform Go agent that turns your computer into a secure, authenticated password-injection endpoint.

It’s designed for a world where you don’t want to type high-value secrets (master passwords, recovery keys, etc.) on your desktop keyboard:

* The secret lives on a trusted device (e.g. your phone).
* Delivery is encrypted and authenticated.
* The daemon injects into the currently focused text field.

> **Key point:** Secrets do not traverse the network in plaintext.

> **Status note:** NovaKey targets normal logged-in desktop sessions (browser fields, terminals, editors, etc). Login screens / lock screens are future/experimental and not guaranteed.

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

  * [`novakey` – the daemon](#novakey--the-daemon)
  * [`nvclient` – reference/test client](#nvclient--referencetest-client)
  * [`nvpair` – device pairing & key management](#nvpair--device-pairing--key-management)
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

NovaKey-Daemon (`novakey`) runs on a workstation (Windows, macOS, Linux). Clients connect to the daemon’s TCP listener (default `:60768`), send an encrypted payload, and NovaKey:

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

---

## Current Capabilities

| ✅ | Capability                                                    |
| - | ------------------------------------------------------------- |
| ✅ | Cross-platform daemon for Linux / macOS / Windows             |
| ✅ | ML-KEM-768 + HKDF-SHA-256 + XChaCha20-Poly1305 transport      |
| ✅ | Per-device keys in `devices.json`                             |
| ✅ | Server keys in `server_keys.json` (auto-generated if missing) |
| ✅ | Timestamp freshness validation                                |
| ✅ | Nonce replay protection                                       |
| ✅ | Per-device rate limiting                                      |
| ✅ | Optional arming gate (“push-to-type”)                         |
| ✅ | Optional two-man mode (arm + device approval required)        |
| ✅ | Local-only Arm API with token auth                            |
| ✅ | Injection safety (`allow_newlines`, `max_inject_len`)         |
| ✅ | Target policy allow/deny lists                                |
| ✅ | Config via YAML (preferred) or JSON fallback                  |
| ✅ | Configurable logging to file w/ rotation + redaction          |

---

## Installers

### Linux Installer

The `Installers/install-linux.sh` script performs a full system-level installation of **NovaKey-Daemon** on Linux systems using `systemd`.
It is intended to be run as root (`sudo`) from the repository root.

```bash
git clone https://github.com/OsbornePro/NovaKey-Daemon.git
cd NovaKey-Daemon*
sudo ./Installers/install-linux.sh
```

#### What the installer does

**1. Creates a dedicated service user**

* Creates a system user named `novakey` (no login shell, no home directory).
* The daemon always runs as this unprivileged user.

**2. Installs the NovaKey daemon binary**

* Copies the built binary from:

  ```
  ./dist/novakey-linux-amd64
  ```

  to:

  ```
  /usr/local/bin/novakey-linux-amd64
  ```

**3. Installs configuration**

* Installs `server_config.yaml` to:

  ```
  /etc/novakey/server_config.yaml
  ```

  and also copies it to:

  ```
  /var/lib/novakey/server_config.yaml
  ```

* If `devices.json` exists in the repo at install time, it is installed to:

  ```
  /etc/novakey/devices.json
  ```

* If `devices.json` does **not** exist, the installer intentionally does **not** create it (pairing mode on first start).

**4. Sets up filesystem layout**

Creates:

* **Configuration**

  ```
  /etc/novakey/
  ```

  * Owned by `root:novakey` (750)
  * Config files readable by the service group

* **Runtime / state (WorkingDirectory)**

  ```
  /var/lib/novakey/
  ```

  * Owned by `novakey` (700)
  * Used as the daemon working directory

* **Logs**

  The installer reads `log_dir` from `server_config.yaml` (best-effort):

  * If `log_dir` is relative (e.g. `./logs`), it resolves under the working directory:

    ```
    /var/lib/novakey/logs/
    ```

  * If `log_dir` is absolute (e.g. `/var/log/novakey`), that directory is created and permitted

  The resolved log directory is created and owned by `novakey`.

**5. systemd service**

* Installs:

  ```
  /etc/systemd/system/novakey.service
  ```

* The service:

  * Runs as user/group `novakey`

  * Uses `/var/lib/novakey` as `WorkingDirectory`

  * Creates + permissions the resolved log directory before startup (`ExecStartPre`)

  * Starts with:

    ```
    /usr/local/bin/novakey-linux-amd64 --config /etc/novakey/server_config.yaml
    ```

  * Restarts on failure

  * Uses systemd hardening (`ProtectSystem=strict`, `NoNewPrivileges`, `LockPersonality`, `MemoryDenyWriteExecute`, etc.)

  * Allows writes only to:

    * `/var/lib/novakey`
    * `/etc/novakey`
    * the resolved `log_dir` path

You can manage the service with:

```bash
sudo systemctl status novakey
sudo systemctl restart novakey
sudo journalctl -u novakey -f
```

**6. Firewall configuration**

* If `firewalld` is present:

  * Installs a service definition:

    ```
    /etc/firewalld/services/novakey.xml
    ```

  * Enables the service if not already enabled

  * Opens TCP port `60768`

#### What the installer does NOT do

* It does not require `server_keys.json`; the daemon generates it automatically on first run if missing.
* It does not create `devices.json` unless you supply one; if absent, pairing mode is used on first start.
* It does not re-add the firewall rule if it already exists.

#### Where to modify behavior

* **Listening address / ports / limits**

  ```
  /etc/novakey/server_config.yaml
  ```

* **Logging location**

  * Change `log_dir` in `server_config.yaml`
  * Restart the service:

    ```bash
    sudo systemctl restart novakey
    ```

* **Paired devices**

  * If `devices_file` is relative (default), it resolves under `WorkingDirectory`:

    ```
    /var/lib/novakey/devices.json
    ```

  * If you set an absolute `devices_file`, it will be wherever you specify.

* **Server cryptographic identity**

  * If `server_keys_file` is relative (default), it resolves under `WorkingDirectory`:

    ```
    /var/lib/novakey/server_keys.json
    ```

* **systemd behavior**

  ```
  /etc/systemd/system/novakey.service
  ```

After modifying the service file:

```bash
sudo systemctl daemon-reload
sudo systemctl restart novakey
```

This installer is designed to be safe to re-run on an existing installation.

---

### Windows Installer

The Windows installer script installs **NovaKey-Daemon** as a hardened Windows Service using native Windows facilities.
It must be run from an **elevated PowerShell session** (*Administrator*).

```powershell
Set-Location -Path "$env:USERPROFILE\Downloads\"
Expand-Archive -Path "$env:USERPROFILE\Downloads\NovaKey-Daemon-main.zip" -DestinationPath .
Set-Location -Path "$env:USERPROFILE\Downloads\NovaKey-Daemon-main"
.\Installers\install-windows.ps1
```

#### What the installer does

**1. Installs the NovaKey daemon binary**

* Copies the executable from:

  ```
  .\Installers\dist\novakey-windows-amd64.exe
  ```

  to:

  ```
  C:\Program Files\NovaKey\novakey-windows-amd64.exe
  ```

**2. Creates the installation layout**

Creates:

* **Program directory**

  ```
  C:\Program Files\NovaKey\
  ```

* **Logs directory**

  ```
  C:\Program Files\NovaKey\logs\
  ```

**3. Creates a Windows Service**

* Service name:

  ```
  NovaKey
  ```

* Display name:

  ```
  NovaKey Service
  ```

* The service:

  * Starts automatically at boot

  * Runs as a **virtual service account**:

    ```
    NT SERVICE\NovaKey
    ```

  * Does **not** run as Administrator or LocalSystem

**4. Applies least-privilege filesystem permissions**

Locks down `C:\Program Files\NovaKey\` so that:

* `NT SERVICE\NovaKey`

  * Has **Modify** permissions (required for logs and runtime state)

* `Administrators`

  * Have **Full Control**

* `Users`

  * Have **Read & Execute** only

Inheritance is disabled to prevent accidental permission leaks.

**5. Firewall configuration**

* Creates a Windows Defender Firewall rule named:

  ```
  NovaKey TCP Listener
  ```

* Allows inbound **TCP** traffic on port:

  ```
  60768
  ```

* The rule is only added if it does not already exist.

**6. Service lifecycle**

* If an existing NovaKey service is found:

  * It is stopped
  * Deleted
  * Re-created cleanly

* The service is started at the end of installation.

You can manage the service with:

```powershell
Get-Service -Name NovaKey
Start-Service -Name NovaKey
Stop-Service -Name NovaKey
Restart-Service -Name NovaKey
```

#### Configuration and runtime behavior

* The installer creates an install directory and log directory, but does not install `server_config.yaml` or `devices.json` by default.
* Runtime files and relative paths typically resolve relative to the process working directory; for predictable behavior, use absolute paths in `server_config.yaml` or ensure the service is launched with explicit config arguments.

#### Where to modify behavior

* **Firewall rule**

  * Managed via Windows Defender Firewall (`wf.msc`)
  * Rule name: *NovaKey TCP Listener*

* **Service**

  * Managed via Services (`services.msc`)
  * Service name: *NovaKey*

---

### macOS Installer

The macOS installer deploys NovaKey-Daemon as a **system LaunchDaemon** and runs it with **least privilege** using a dedicated service account.

#### How to run

Run from the repository root:

```bash
sudo ./Installers/install-macos.sh
```

#### What it installs

**Binary**

* Installs the daemon binary to:

  * `/usr/local/bin/novakey`

**Application Support layout**

Creates:

* `/Library/Application Support/NovaKey/`
* `/Library/Application Support/NovaKey/config/` (configuration)
* `/Library/Application Support/NovaKey/data/` (runtime working directory)

**Logging directory**

The installer reads `log_dir` from `server_config.yaml` (best-effort) and resolves it as follows:

* If `log_dir` is relative (e.g. `./logs`), it resolves under the daemon `WorkingDirectory`:

  * `/Library/Application Support/NovaKey/data/logs`

* If `log_dir` is absolute (e.g. `/var/log/novakey`), it uses that absolute path.

The resolved log directory is created and owned by the service user.

**LaunchDaemon**

Installs:

* `/Library/LaunchDaemons/com.osbornepro.novakey.plist`

Starts the daemon with:

* `/usr/local/bin/novakey --config /Library/Application Support/NovaKey/config/server_config.yaml`

Sets:

* `WorkingDirectory` to:

  * `/Library/Application Support/NovaKey/data`

Routes stdout/stderr to the resolved `log_dir`:

* `out.log`
* `err.log`

#### Security model and permissions

**Dedicated service user**

* Creates a system user and group:

  * `novakey:novakey`

* Runs the LaunchDaemon as:

  * `novakey:novakey` (not root)

**Filesystem permissions**

* Root-owned:

  * `/usr/local/bin/novakey` → `root:wheel` (755)
  * `/Library/Application Support/NovaKey/` → `root:wheel` (755)
  * `/Library/Application Support/NovaKey/config/` → `root:wheel` (755)

* Config files readable by the daemon (group readable), writable only by root:

  * `/Library/Application Support/NovaKey/config/server_config.yaml` → `root:novakey` (640)
  * `/Library/Application Support/NovaKey/config/devices.json` (if installed) → `root:novakey` (640)

* Service-owned (writable only by the daemon):

  * `/Library/Application Support/NovaKey/data/` → `novakey:novakey` (700)
  * resolved `log_dir` path → `novakey:novakey` (700)

#### Pairing behavior (`devices.json`)

* If `devices.json` is **not** present in the repo when installing, the installer **does not create it**.
* On first start, the daemon will enter pairing mode and display a QR code to bootstrap the first device.
* If `devices.json` *is* provided, it is installed to:

  * `/Library/Application Support/NovaKey/config/devices.json`

#### Service management

Check service status:

```bash
sudo launchctl list | grep com.osbornepro.novakey
```

Reload the service after modifying the plist:

```bash
sudo launchctl bootout system /Library/LaunchDaemons/com.osbornepro.novakey.plist
sudo launchctl bootstrap system /Library/LaunchDaemons/com.osbornepro.novakey.plist
sudo launchctl kickstart -k system/com.osbornepro.novakey
```

View logs (paths depend on `log_dir`):

```bash
sudo tail -f "/Library/Application Support/NovaKey/data/logs/out.log"
sudo tail -f "/Library/Application Support/NovaKey/data/logs/err.log"
```

#### macOS permissions required for typing

NovaKey requires OS permission to inject keystrokes:

* System Settings → Privacy & Security → **Accessibility**
* System Settings → Privacy & Security → **Input Monitoring**

Without these permissions, the daemon may accept requests but fail to type into applications.

---

## Command-line Tools

### `novakey` – the daemon

Loads:

* `server_config.yaml` (preferred) or `server_config.json` (fallback)
* `devices.json`
* `server_keys.json`

Then listens and handles one framed message per connection.

Linux/macOS:

```bash
./dist/novakey-linux-amd64
# or macOS
./dist/novakey-darwin-amd64
# (or arm64 build if you produce it)
```

Windows (PowerShell):

```powershell
.\dist\novakey-windows-amd64.exe
```

### `nvclient` – reference/test client

Sends v3 frames to the daemon using a typed inner message frame.

Inject example:

```bash
./dist/nvclient \
  -addr 127.0.0.1:60768 \
  -device-id phone \
  -key-hex <device_key_hex> \
  -server-kyber-pub-b64 "<server_kyber768_public>" \
  -password "SuperStrongPassword123!"
```

Approve example (for two-man mode):

```bash
./dist/nvclient approve \
  -addr 127.0.0.1:60768 \
  -device-id phone \
  -key-hex <device_key_hex> \
  -server-kyber-pub-b64 "<server_kyber768_public>"
```

#### Arm API helper

If `arm_api_enabled: true`:

```bash
./dist/nvclient arm --addr 127.0.0.1:60769 --token_file arm_token.txt --ms 20000
```

### `nvpair` – device pairing & key management

Generates a device key and emits pairing JSON:

```bash
./dist/nvpair -id phone
```

---

## Configuration Files

NovaKey supports YAML (preferred) and JSON (fallback). If both exist, YAML wins.

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

By default, logs go to stderr/stdout.

NovaKey also supports logging to a file with rotation:

```yaml
log_dir: "./logs"        # or "/var/log/novakey"
log_rotate_mb: 10
log_keep: 10
log_stderr: true
log_redact: true
```

Notes:

* Passwords are never logged in full (only a short preview).
* With `log_redact: true`, NovaKey redacts configured tokens (if available), long blob-like strings, and obvious secret patterns.

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

* Go (1.21+ recommended)
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

## Known Issues

### Linux Wayland sessions

On native Wayland sessions (`XDG_SESSION_TYPE=wayland`), NovaKey may not support focused-app detection or keystroke injection yet. Clipboard fallback can be used depending on configuration.

---

## License

NovaKey-Daemon is licensed under the Apache License, Version 2.0. See `LICENSE.md`.

---

## Contact & Support

* Support: `security@novakey.app`
* Security disclosures: see `SECURITY.md` (do not open security findings as public issues)

