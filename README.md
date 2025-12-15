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

> **Important:** In v3, the *outer* frame type is fixed (server expects outer `msgType=1`).
> “Approve vs Inject” is represented by an **inner message frame** inside the AEAD plaintext.

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

> Note: older configs may include fields like `approve_magic` / `legacy_*`.
> Current protocol uses **typed approve messages**, not magic strings.

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

* Support: `support@novakey.app`
* Security disclosures: see `SECURITY.md` (do not open security findings as public issues)

```
