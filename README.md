# 🔐 NovaKey-Daemon by OsbornePro

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

| ✅ | Capability |
| - | ---------- |
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

**3. Sets up filesystem layout**
The installer creates and configures the following directories:

* **Configuration**

  ```
  /etc/novakey/
  ```

  * Owned by `root:novakey`
  * Readable by the service
  * Contains `server_config.yaml`
  * May contain `devices.json` if provided

* **Runtime / state**

  ```
  /var/lib/novakey/
  ```

  * Owned by `novakey`
  * Used as the daemon working directory
  * Holds runtime-generated files such as:

    * `server_keys.json` (auto-generated on first run)
    * `devices.json` (created after first pairing)
    * logs when `log_dir` is relative (e.g. `./logs`)

* **Logs**

  * The installer reads `log_dir` from `server_config.yaml`
  * If `log_dir` is relative (e.g. `./logs`), logs go to:

    ```
    /var/lib/novakey/logs/
    ```
  * If `log_dir` is absolute (e.g. `/var/log/novakey`), that directory is created and permitted
  * The installer ensures the log directory exists and is writable by `novakey`

**4. Installs configuration**

* Copies `server_config.yaml` to:

  ```
  /etc/novakey/server_config.yaml
  ```

  and also into:

  ```
  /var/lib/novakey/server_config.yaml
  ```

  so relative paths in the config resolve correctly.
* If `devices.json` exists in the repo, it is installed.
* If `devices.json` does **not** exist, it is intentionally **not created** — this allows NovaKey to display a QR code on first startup for initial pairing.

**5. systemd service**

* Installs:

  ```
  /etc/systemd/system/novakey.service
  ```
* The service:

  * Runs as user/group `novakey`
  * Uses `/var/lib/novakey` as `WorkingDirectory`
  * Automatically creates the configured log directory before startup
  * Restarts on failure
  * Uses strong systemd hardening (`ProtectSystem=strict`, `NoNewPrivileges`, etc.)

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
  * Opens TCP port `60768`
  * Will not re-add the rule if it already exists

#### What the installer does NOT do

* It does not create `devices.json` unless you supply one. this file is generated automatically by the daemon on first run.
* It does not require `server_keys.json`; this file is generated automatically by the daemon on first run.
* It does not overwrite existing firewall rules or paired devices unless explicitly provided.

#### Where to modify behavior

* **Listening address / ports / limits**
  Edit:

  ```
  /etc/novakey/server_config.yaml
  ```

* **Logging location**

  * Change `log_dir` in `server_config.yaml`
  * Re-run the installer or restart the service

* **Paired devices**

  ```
  /var/lib/novakey/devices.json
  ```

* **Server cryptographic identity**

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

This installer is safe to re-run and is designed to be idempotent for existing installations.

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

* Copies the executable from the installer directory:

  ```
  .\dist\novakey-windows-amd64.exe
  ```

  to:

  ```
  C:\Program Files\NovaKey\novakey-windows-amd64.exe
  ```

**2. Creates the installation layout**
The installer creates the following directories:

* **Program directory**

  ```
  C:\Program Files\NovaKey\
  ```

  * Contains the NovaKey executable
  * Contains runtime-generated files (*keys, devices, logs*)

* **Logs directory**

  ```
  C:\Program Files\NovaKey\logs\
  ```

  * Used when `log_dir` is set to a relative path in `server_config.yaml`

**3. Creates a Windows Service**

* Creates a Windows service named:

  ```
  NovaKey
  ```

* Display name:

  ```
  NovaKey Service
  ```

* Description:

  ```
  NovaKey secure secret transfer service
  ```

* The service:

  * Starts automatically at boot
  * Runs as a **virtual service account**:

    ```
    NT SERVICE\NovaKey
    ```
  * Does **not** run as Administrator or LocalSystem

**4. Applies least-privilege filesystem permissions**
The installer locks down the install directory so that:

* `NT SERVICE\NovaKey`

  * Has **Modify** permissions (required for logs, keys, pairing data)
* `Administrators`

  * Have **Full Control**
* `Users`

  * Have **Read & Execute** only

Inheritance is disabled to prevent accidental permission leaks from parent directories.

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
* The service is started automatically at the end of installation.

You can manage the service with:

```powershell
Get-Service -Name NovaKey
Start-Service -Name NovaKey
Stop-Service -Name NovaKey
Restart-Service -Name NovaKey
```

#### Configuration and runtime behavior

* The daemon reads `server_config.yaml` from its working directory.
* Relative paths in the configuration (such as `devices.json`, `server_keys.json`, or `./logs`) resolve relative to:

  ```
  C:\Program Files\NovaKey\
  ```
* If `devices.json` does **not** exist on first start:

  * NovaKey enters pairing mode
  * A QR code is displayed for initial device pairing
* `server_keys.json` is generated automatically on first run if missing.

#### Where to modify behavior

* **Listening address, limits, logging**

  * Edit `server_config.yaml` in the install directory

* **Paired devices**

  ```
  C:\Program Files\NovaKey\devices.json
  ```

* **Server cryptographic identity**

  ```
  C:\Program Files\NovaKey\server_keys.json
  ```

* **Logs**

  ```
  C:\Program Files\NovaKey\logs\
  ```

* **Firewall rule**

  * Managed via Windows Defender Firewall (`wf.msc`)
  * Rule name: *NovaKey TCP Listener*

The Windows installer is designed to be **idempotent** and safe to re-run on an existing installation.

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

