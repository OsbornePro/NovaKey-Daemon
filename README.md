Below is a **cleaned-up, drop-in replacement `README.md`** that implements Step 1 (“Security Tester Mode / armed-by-default recommended config”), fixes a few issues in your current README (TOC anchor typo, arm port wording, duplicate “Reporting a Vulnerability” block inside README, clarity around YAML vs JSON), and documents the Arm API + `nvclient arm` clearly.

Copy/paste the entire file as-is.

---

# 🔐 NovaKey-Daemon by OsbornePro

**What is NovaKey-Daemon?**
*NovaKey-Daemon is a lightweight, cross-platform Go agent that turns your computer into a secure, authenticated password-injection endpoint.*

**Why would I need this?**
Even with a password manager you still need a master password (or other high-value secret). That secret is often the weakest link—either memorized, re-used, or stored in sketchy ways.

NovaKey aims to eliminate “manual typing” of those secrets:

* Your real master password lives only on a trusted device (e.g. your phone).
* You never type it manually on the keyboard.
* Delivery is encrypted and authenticated using:

  * **ML-KEM-768 (Kyber-768-compatible KEM)** for post-quantum key establishment
  * **HKDF-SHA-256** for session key derivation
  * **XChaCha20-Poly1305 AEAD** per-message encryption/authentication
* The NovaKey daemon injects the secret into the currently focused text field on your desktop.

> **Key point:** The secret never traverses the network in plaintext.

> **Status note:** Current code targets *normal logged-in desktop sessions* (browser fields, terminals, editors, etc.). Lock screens / pre-boot PINs / login screens are future/experimental work and are not guaranteed or supported yet.

## Security Review Invited

NovaKey-Daemon v3 uses ML-KEM-768 + HKDF + XChaCha20-Poly1305 with per-device secrets and replay / rate-limit controls.
The protocol and security model are documented in `PROTOCOL.md` and `SECURITY.md`.

We’re actively looking for feedback on:

* The v3 key schedule (KEM + per-device secret → AEAD key)
* Replay / freshness logic
* Any injection-path weirdness on Windows/macOS/Linux
* Safety controls around injection (arming gate, newline blocking, length caps)

---

## Table of Contents

* [Overview](#overview)
* [Current Capabilities](#current-capabilities)
* [Command-line Tools](#command-line-tools)

  * [`novakey` – the daemon](#novakey--the-daemon)
  * [`nvclient` – reference/test client](#nvclient--referencetest-client)
  * [`nvpair` – device pairing & key management](#nvpair--device-pairing--key-management)
* [Configuration Files](#configuration-files)
* [Arming Gate](#arming-gate)
* [Security Tester Mode](#security-tester-mode)
* [Protocol & Crypto Stack](#protocol--crypto-stack)
* [Auto-Type Support Notes](#auto-type-support-notes)
* [Roadmap](#roadmap)
* [Build from Source](#build-from-source)
* [Running NovaKey-Daemon](#running-novakey-daemon)
* [Logging](#logging)
* [Contributing](#contributing)
* [Known Issues](#known-issues)
* [License](#license)
* [Contact & Support](#contact--support)

---

## Overview

The NovaKey-Daemon service (`novakey`) runs on a workstation (Windows, macOS, or Linux). One or more clients (e.g. a future mobile app, or the included `nvclient` test tool) connect to the daemon’s TCP listener (default `:60768`), send an encrypted payload, and NovaKey-Daemon:

1. **Authenticates** the device via a per-device symmetric key (PSK) stored on the host.
2. **Derives a per-message session key** using **ML-KEM-768** and **HKDF-SHA-256**.
3. **Decrypts & validates** the request using XChaCha20-Poly1305 with:

   * Per-device PSK as salt
   * Fresh per-message KEM shared secret as input key material
   * Timestamps
   * Nonce-based replay protection
   * Per-device rate limiting
4. **Injects** the resulting password into the currently focused control on the desktop.

### Optional safety: Armed Injection Gate (push-to-type)

NovaKey can run in an **armed injection** mode where secrets are only injected after a **local arm action** (push-to-type), reducing the impact of a compromised device secret. Arming is a local control gate applied after decrypt/validation.

> The Arm API is **local-only** and binds to `127.0.0.1:60769` (it refuses non-loopback binds).

All cryptographic operations are done locally; there is no cloud service or third-party relay.

The protocol version in use is **v3** (see `PROTOCOL.md`).

---

## Current Capabilities

| ✅ | Capability                                                                                  |
| - | ------------------------------------------------------------------------------------------- |
| ✅ | Cross-platform daemon (`novakey`) for **Linux**, **macOS**, and **Windows**                 |
| ✅ | Encrypted & authenticated password delivery using **XChaCha20-Poly1305**                    |
| ✅ | **Post-quantum key establishment** via **ML-KEM-768 (Kyber-768-compatible)**                |
| ✅ | Per-device keys and device IDs stored in `devices.json`                                     |
| ✅ | Automatic generation & persistence of server Kyber keys in `server_keys.json`               |
| ✅ | Message freshness (timestamp) validation                                                    |
| ✅ | Nonce-based replay protection per device                                                    |
| ✅ | Per-device rate limiting (requests/min)                                                     |
| ✅ | Configurable limits via `server_config.yaml` (preferred) or `server_config.json` (fallback) |
| ✅ | CLI test client (`nvclient`) that speaks v3                                                 |
| ✅ | CLI pairing / key management tool (`nvpair`) that emits JSON suitable for QR-based pairing  |
| ✅ | Optional **armed injection gate** (blocks injection unless locally armed)                   |
| ✅ | Local-only Arm API on `127.0.0.1:60769` with token auth (for hotkey binding / testing)      |
| ✅ | Auto-generation of `arm_token.txt` when Arm API is enabled                                  |
| ✅ | Injection safety policies: newline blocking by default (`\n`, `\r`) + `max_inject_len`      |

---

## Command-line Tools

All commands live under `cmd/` and are typically built into binaries under `dist/` by `build.sh` / `build.ps1`.

### `novakey` – the daemon

The main service process:

* Loads configuration from `server_config.yaml` (preferred) or `server_config.json` (fallback).
* Loads per-device keys from `devices.json`.
* Loads (or auto-generates) server ML-KEM-768 keys in `server_keys.json`.
* Listens on the configured TCP address (default `0.0.0.0:60768`).
* For each incoming connection:

  * Reads a single framed message (`[u16 length][payload]`)
  * Decapsulates the KEM ciphertext to get a per-message shared secret
  * Derives a session key with HKDF-SHA-256 using the per-device key as salt
  * Decrypts and validates the payload with XChaCha20-Poly1305
  * Applies timestamp, replay, and rate-limit checks
  * Applies arming gate / safety policies (optional)
  * Injects the password into the focused control
  * Closes the connection

Typical usage (Linux/macOS):

```bash
./dist/novakey-linux-amd64
# or on macOS
./dist/NovaKey-darwin-arm64
```

Typical usage (Windows, PowerShell):

```powershell
.\dist\NovaKey.exe
```

By default it logs to stdout/stderr; you can wrap it in systemd / launchd / a Windows Service if you want it to run as a background service.

---

### `nvclient` – reference/test client

A simple CLI client that speaks the **v3 NovaKey-Daemon protocol**. It’s useful for:

* Testing the daemon
* Experimenting with passwords and device IDs
* Serving as a reference implementation for other clients (e.g. mobile apps)

Usage (example):

```bash
./dist/nvclient \
  -addr 127.0.0.1:60768 \
  -device-id roberts-phone \
  -key-hex 7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234 \
  -server-kyber-pub-b64 "<base64-encoded server Kyber public key>" \
  -password "SuperStrongPassword123!"
```

#### Arming the daemon (local-only)

If the daemon has `arm_api_enabled: true`, `novakey` will create `arm_token.txt` (if missing). Then you can arm via:

```bash
# Arms NovaKey for 20 seconds (requires arm_api_enabled:true)
./dist/nvclient arm --addr 127.0.0.1:60769 --token_file arm_token.txt --ms 20000
```

---

### `nvpair` – device pairing & key management

A helper that:

* Generates a random 32-byte per-device key
* Adds a new device entry to `devices.json`, or updates an existing one (with `-force`)
* Reads `server_config.yaml` or `server_config.json` and `server_keys.json`
* Emits a pairing JSON blob containing:

  * `device_id`
  * `device_key_hex`
  * `server_addr`
  * `server_kyber768_pub` (base64)

Example:

```bash
./dist/nvpair -id roberts-phone
```

---

## Configuration Files

NovaKey supports **YAML** (preferred) and **JSON** (fallback). If **both** exist, the daemon prefers `server_config.yaml`.

### `server_config.yaml` (preferred) / `server_config.json` (fallback)

Core fields:

* `listen_addr` – TCP address to bind to (`127.0.0.1:60768` local-only, `0.0.0.0:60768` LAN)
* `max_payload_len` – max allowed payload bytes (before decryption)
* `max_requests_per_min` – per-device rate limit
* `devices_file` – path to `devices.json`
* `server_keys_file` – path to `server_keys.json`

Arming gate + safety fields:

* `arm_enabled` – if true, injection requires arming
* `arm_duration_ms` – default arm duration used by `/arm` if no `ms` override
* `arm_consume_on_inject` – if true, disarms after the first injection
* `allow_clipboard_when_disarmed` – if true, may still set clipboard when injection is blocked
* `arm_api_enabled` – enables local control API
* `arm_listen_addr` – **must be loopback** (recommended `127.0.0.1:60769`)
* `arm_token_file` – token file path (auto-generated if missing when API is enabled)
* `arm_token_header` – header name (default `X-NovaKey-Token`)
* `allow_newlines` – default false; blocks `\n` and `\r`
* `max_inject_len` – max length of injected text (separate from payload length)

> **Security note:** The Arm API is intentionally loopback-only. **DO NOT** bind it to `0.0.0.0`.

---

## Arming Gate

When `arm_enabled: true`, the daemon will still decrypt and validate frames, but it will **block injection** unless the service is armed.

Arming can be done via the local Arm API:

* `POST /arm?ms=20000` (token required)
* `GET /status` (token required)

The Arm API is local-only and binds to `127.0.0.1:60769`.

---

## Security Tester Mode

If you’re security testing or demoing NovaKey and want safer defaults, use this configuration. It makes injection explicitly “push-to-type” and removes clipboard side-effects.

Create `server_config.yaml`:

```yaml
listen_addr: "0.0.0.0:60768"
max_payload_len: 4096
max_requests_per_min: 30
devices_file: "devices.json"
server_keys_file: "server_keys.json"

# Require local arming before injection
arm_enabled: true
arm_duration_ms: 20000
arm_consume_on_inject: true

# Strict mode: do NOT set clipboard when disarmed
allow_clipboard_when_disarmed: false

# Enable local-only arm API (loopback)
arm_api_enabled: true
arm_listen_addr: "127.0.0.1:60769"
arm_token_file: "arm_token.txt"
arm_token_header: "X-NovaKey-Token"

# Injection safety
allow_newlines: false
max_inject_len: 256
```

Test workflow:

1. Start the daemon:

```bash
./dist/novakey-linux-amd64
```

2. Arm for 20 seconds:

```bash
./dist/nvclient arm --addr 127.0.0.1:60769 --token_file arm_token.txt --ms 20000
```

3. Focus a text field, then send a frame:

```bash
./dist/nvclient \
  -addr 127.0.0.1:60768 \
  -device-id phone \
  -key-hex <device_key_hex> \
  -server-kyber-pub-b64 "<server_kyber768_public>" \
  -password "TestPassword123!"
```

---

## Protocol & Crypto Stack

For complete details, see [`PROTOCOL.md`](PROTOCOL.md). Here’s the short version.

* **KEM:** ML-KEM-768 (`filippo.io/mlkem768`)
* **KDF:** HKDF-SHA-256 (`golang.org/x/crypto/hkdf`)
* **AEAD:** XChaCha20-Poly1305 (`golang.org/x/crypto/chacha20poly1305.NewX`)
* **Transport:** TCP framed as `[u16 length][payload]`
* **Protocol version:** v3

> Note: the **device ID is sent in the clear** (as part of the header/AAD) to allow routing and logging. Do not use sensitive identifiers as device IDs.

---

## Auto-Type Support Notes

NovaKey-Daemon’s goal is:

> “NovaKey-Daemon works on most normal apps/fields, but some weird or high-security ones just aren’t supported.”

Current behavior:

* **Linux**

  * **X11/Xwayland:** uses `xdotool` and `xclip` to type/paste into the active control.
  * **Wayland:** keystroke injection is not implemented yet; NovaKey can operate in clipboard-only style depending on config.
* **macOS**

  * Uses Accessibility APIs / AppleScript-style automation.
  * Requires granting Accessibility / Input permissions in System Settings.
* **Windows**

  * Uses clipboard and standard input APIs; falls back to synthetic typing where needed.
  * Some elevated / secure desktops may not accept synthetic input.

---

## Roadmap

Features planned or on deck:

| Feature                                  | Status       |
| ---------------------------------------- | ------------ |
| Companion mobile app (iOS/Android)       | Planned      |
| QR-based pairing UX                      | In progress  |
| Installer / service packaging per OS     | Planned      |
| GUI tray icon & config UI                | Planned      |
| TOTP / MFA code support                  | Planned      |
| Optional “approve before typing” prompts | Planned      |
| Better lock/login-screen integration     | Experimental |

---

## Build from Source

You’ll need:

* Go (1.21+ recommended)
* Git (if cloning)
* Standard Go build toolchain

Clone:

```bash
git clone https://github.com/OsbornePro/NovaKey-Daemon.git
cd NovaKey-Daemon
```

Build (Linux/macOS, Bash):

```bash
./build.sh -t linux
./build.sh -t darwin
./build.sh -t windows
```

Build (Windows, PowerShell):

```powershell
Set-Location -Path "C:\Path\To\NovaKey-Daemon"
.\build.ps1 -Target windows -FileName NovaKey.exe
```

---

## Running NovaKey-Daemon

1. Ensure `server_config.yaml` or `server_config.json` exists.

2. Start the daemon:

```bash
./dist/novakey-linux-amd64
```

3. Create a device and pairing info:

```bash
./dist/nvpair -id roberts-phone
```

4. Arm (if enabled):

```bash
./dist/nvclient arm --addr 127.0.0.1:60769 --token_file arm_token.txt --ms 20000
```

5. Send a test frame (focus a text field first):

```bash
./dist/nvclient \
  -addr 127.0.0.1:60768 \
  -device-id roberts-phone \
  -key-hex <device_key_hex> \
  -server-kyber-pub-b64 "<server_kyber768_public>" \
  -password "SuperStrongPassword123!"
```

---

## Logging

Logs are written to stdout/stderr by default.

Examples:

```bash
./dist/novakey-linux-amd64 2>&1 | tee novakey.log
```

---

## Contributing

We welcome contributions! Please:

1. Fork the repository and create a feature branch (`git checkout -b feat/your-feature`)
2. Write tests (`go test ./...`)
3. Run linters if you use them (e.g. `golangci-lint run`)
4. Update docs if you change flags/behavior/protocol
5. Submit a Pull Request

---

## Known Issues

### Linux Wayland sessions

On Linux Wayland sessions (`XDG_SESSION_TYPE=wayland`), NovaKey-Daemon:

* **Does** handle crypto, validation, and (optionally) clipboard behavior
* **Does not** currently perform keystroke injection into native Wayland windows

Workarounds:

* Use an Xorg/X11 session instead of Wayland, or
* Run target apps under Xwayland where possible (e.g., `MOZ_ENABLE_WAYLAND=0 firefox`), or
* Use NovaKey-Daemon in clipboard-only mode and paste manually

For more detail, see: [https://github.com/OsbornePro/NovaKey-Daemon/issues/3](https://github.com/OsbornePro/NovaKey-Daemon/issues/3)

---

## License

NovaKey-Daemon (by OsbornePro) is licensed under the Apache License, Version 2.0.
See `LICENSE.md` for details.

---

## Contact & Support

* Product website: [https://novakey.app](https://novakey.app)
* Technical support: [support@novakey.app](mailto:support@novakey.app)
* Security disclosures: see `SECURITY.md` (do **not** open security findings as GitHub issues)
* GitHub issues: bugs / feature requests / installation help

---
