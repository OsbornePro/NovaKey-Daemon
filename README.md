# 🔐 NovaKey by OsbornePro

**What is NovaKey?**
*NovaKey is a lightweight, cross-platform Go agent that turns your computer into a secure, authenticated password-injection endpoint.*

**Why would I need this?**
Even with a password manager you still need a master password (or other high-value secret). That secret is often the weakest link—either memorized, re-used, or stored in sketchy ways.

NovaKey aims to eliminate “manual typing” of those secrets:

* Your real master password lives only on a trusted device (e.g. your phone).
* You never type it manually on the keyboard.
* Delivery is encrypted and authenticated using:

  * **ML-KEM-768 (Kyber-768-compatible KEM)** for post-quantum key establishment, plus
  * **XChaCha20-Poly1305 AEAD** with per-device keys and HKDF-derived session keys.
* The NovaKey daemon injects the secret into the currently focused text field on your desktop.

> **Key point:** The secret never passes through the keyboard as raw keystrokes from you, and never traverses the network in plaintext.

> **Status note:** Current code targets *normal logged-in desktop sessions* (browser fields, terminals, editors, etc.). Lock screens / pre-boot PINs / login screens are future/experimental work, not guaranteed or supported yet.

---

## Table of Contents

* [Overview](#overview)
* [Current Capabilities](#current-capabilities)
* [Command-line Tools](#command-line-tools)
* [Configuration Files](#configuration-files)
* [Protocol & Crypto Stack](#protocol--crypto-stack)
* [Auto-Type Support Notes](#auto-type-support-notes)
* [Roadmap](#roadmap)
* [Security Notes (Current Implementation)](#security-notes-current-implementation)
* [Build from Source](#build-from-source)
* [Running NovaKey](#running-novakey)
* [Logging](#logging)
* [Contributing](#contributing)
* [Known Issues](#known-issues)
* [License](#license)
* [Contact & Support](#contact--support)

---

## Overview

The NovaKey service (`novakey`) runs on a workstation (*Windows, macOS, or Linux*). It creates a TCP listener (default `0.0.0.0:60768`). One or more clients (e.g. a future mobile app, or the included `nvclient` test tool) connect to this listener, send an encrypted payload, and NovaKey:

1. **Authenticates** the device via a per-device symmetric key (PSK) stored on the host.
2. **Derives a per-message session key** using **ML-KEM-768** (Kyber-768-compatible) and **HKDF-SHA-256**.
3. **Decrypts & validates** the request using XChaCha20-Poly1305 with:

   * Per-device PSK as salt,
   * Fresh per-message KEM shared secret as input key material,
   * Timestamps,
   * Nonce-based replay protection,
   * Per-device rate limiting.
4. **Injects** the resulting password into the currently focused control on the desktop.

All cryptographic operations are done locally; there is no cloud service or third-party relay.

The protocol version in use is **v3** (see `PROTOCOL.md`).

---

## Current Capabilities

| ✅ | Capability                                                                                 |
| - | ------------------------------------------------------------------------------------------ |
| ✅ | Cross-platform daemon (`novakey`) for **Linux**, **macOS**, and **Windows**                |
| ✅ | Encrypted & authenticated password delivery using **XChaCha20-Poly1305**                   |
| ✅ | **Post-quantum key establishment** via **ML-KEM-768 (Kyber-768-compatible)**               |
| ✅ | Per-device keys and device IDs stored in `devices.json`                                    |
| ✅ | Automatic generation & persistence of server Kyber keys in `server_keys.json`              |
| ✅ | Message freshness (timestamp) validation                                                   |
| ✅ | Nonce-based replay protection per device                                                   |
| ✅ | Per-device rate limiting (requests/min)                                                    |
| ✅ | Configurable listen address, payload size, and limits via `server_config.json`             |
| ✅ | Simple CLI test client (`nvclient`) that speaks the v3 protocol                            |
| ✅ | CLI pairing / key management tool (`nvpair`) that emits JSON suitable for QR-based pairing |

---

## Command-line Tools

All commands live under `cmd/` and are typically built into binaries under `dist/` by `build.sh` / `build.ps1`.

### `novakey` – the daemon

The main service process:

* Loads configuration from `server_config.json`.
* Loads per-device keys from `devices.json`.
* Loads (or auto-generates) server Kyber/ML-KEM-768 keys in `server_keys.json`.
* Listens on the configured TCP address (default `0.0.0.0:60768`).
* For each incoming connection:

  * Reads a single framed message (`[u16 length][payload]`).
  * Decapsulates the KEM ciphertext to get a per-message shared secret.
  * Derives a session key with HKDF-SHA-256 using the per-device key as salt.
  * Decrypts and validates the payload with XChaCha20-Poly1305.
  * Applies timestamp, replay, and rate-limit checks.
  * Injects the password into the focused control.
  * Closes the connection.

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

A simple CLI client that speaks the **v3 NovaKey protocol**. It’s useful for:

* Testing the daemon.
* Experimenting with passwords and device IDs.
* Serving as a reference implementation for other clients (e.g. mobile apps).

Usage (example):

```bash
./dist/nvclient \
  -addr 192.168.8.244:60768 \
  -device-id roberts-phone \
  -key-hex 7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234 \
  -server-kyber-pub-b64 "<base64-encoded server Kyber public key>" \
  -password "SuperStrongPassword123!"
```

Flags (current implementation):

* `-addr` – address of the NovaKey daemon (e.g. `127.0.0.1:60768` or `192.168.x.x:60768`)
* `-device-id` – device ID that must exist in `devices.json`
* `-key-hex` – 32-byte per-device key in hex (matches `key_hex` in `devices.json`)
* `-server-kyber-pub-b64` – base64-encoded ML-KEM-768 public key from `server_keys.json`
* `-password` – password/secret string to send and inject

Internally, `nvclient`:

1. Encapsulates to the server’s Kyber public key (ML-KEM-768).
2. Derives a 32-byte XChaCha20-Poly1305 key via HKDF-SHA-256 using the device key as salt.
3. Builds the v3 frame and sends it to the daemon.

---

### `nvpair` – device pairing & key management

A helper that:

* Generates a random 32-byte per-device key.
* Adds a new device entry to `devices.json`, or updates an existing one (with `-force`).
* Reads `server_config.json` and `server_keys.json`.
* Emits a **pairing JSON blob** that contains everything a client/phone app needs:

  * `device_id`
  * `device_key_hex`
  * `server_addr`
  * `server_kyber768_pub` (base64)

Example:

```bash
./dist/nvpair -id roberts-phone
```

Example output (simplified):

```text
Added new device "roberts-phone" to /path/to/devices.json
------------------------------------------------------------
 Pairing info (JSON)
------------------------------------------------------------
{
  "v": 1,
  "device_id": "roberts-phone",
  "device_key_hex": "7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234",
  "server_addr": "192.168.8.244:60768",
  "server_kyber768_pub": "<base64-encoded public key>"
}

Use this pairing info in your phone app to configure NovaKey v3.
```

If you have QR tooling available, you can render the JSON as a QR code (for example, via `qrencode`, or via the optional `go-qrcode` integration in the source).

Flags:

* `-devices-file` – path to `devices.json` (default: `devices.json` in CWD)
* `-config-file` – path to `server_config.json` (default: `server_config.json`)
* `-id` – device ID to add or update (required)
* `-force` – overwrite an existing device with the same ID
* `-qr` – optionally print instructions or an ASCII QR for pairing (implementation-dependent)

---

## Configuration Files

### `server_config.json`

Controls how the daemon listens and enforces limits.

Example:

```json
{
  "listen_addr": "0.0.0.0:60768",
  "max_payload_len": 4096,
  "max_requests_per_min": 60,
  "devices_file": "devices.json",
  "server_keys_file": "server_keys.json"
}
```

Fields:

* `listen_addr` – TCP address to bind to.

  * `127.0.0.1:60768` – **local only**
  * `0.0.0.0:60768` – listen on all interfaces (for LAN usage)
* `max_payload_len` – max allowed payload bytes (before decryption).
* `max_requests_per_min` – per-device rate limit.
* `devices_file` – path to the `devices.json` file.
* `server_keys_file` – path to `server_keys.json` (ML-KEM-768 server keypair).

> **Important:** If you expose `0.0.0.0:60768`, ensure your firewall is configured appropriately. NovaKey enforces authentication and replay protection, but the port is still a high-value interface.

---

### `devices.json`

Defines which devices are allowed to talk to NovaKey and what keys they use.

Example:

```json
{
  "devices": [
    {
      "id": "roberts-phone",
      "key_hex": "7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234"
    }
  ]
}
```

You normally won’t edit this by hand; use `nvpair` instead.

---

### `server_keys.json`

Holds the long-lived **ML-KEM-768 (Kyber) keypair** for the host.

Generated automatically by the daemon if missing:

```json
{
  "kyber768_public": "<base64-encoded ML-KEM-768 public key>",
  "kyber768_secret": "<base64-encoded ML-KEM-768 private key>"
}
```

* `kyber768_public` is safe to distribute to clients (via pairing JSON/QR).
* `kyber768_secret` MUST be kept private and never leaves the host.

---

## Protocol & Crypto Stack

For complete details, see [`PROTOCOL.md`](PROTOCOL.md). Here’s the short version.

### Protocol Version

* Current protocol version is **3**.
* Frames with `version != 3` are rejected.

### Transport & Framing

* **Transport:** TCP

* **Default port:** `60768`

* **Frame format:**

  ```text
  [ u16 length ][ length bytes of payload ]
  ```

* Payload (`frame`) layout (v3):

  ```text
  [0]             = version (3)
  [1]             = msgType (1 = password)
  [2]             = idLen (N)
  [3..3+N-1]      = deviceID bytes
  [..]            = kemCt (ML-KEM-768 ciphertext)
  [..]            = nonce (24 bytes, XChaCha20)
  [..]            = ciphertext (AEAD output)
  ```

The header `version || msgType || idLen || deviceID` is used as AEAD **associated data (AAD)**.

### Crypto Stack

* **KEM:** ML-KEM-768 (Kyber-768-compatible) via `filippo.io/mlkem768`
* **AEAD:** XChaCha20-Poly1305 (`golang.org/x/crypto/chacha20poly1305.NewX`)
* **KDF:** HKDF-SHA-256 (`golang.org/x/crypto/hkdf`)

Per-message key derivation:

1. Client encapsulates to server’s Kyber public key → `(kemCt, kemShared)`.

2. Server decapsulates using Kyber private key → `kemShared`.

3. Both sides run HKDF:

   ```text
   IKM  = kemShared (32 bytes from KEM)
   salt = per-device key (device_key_hex -> 32 bytes)
   info = "NovaKey v3 session key"
   K    = HKDF-SHA256(IKM, salt, info, outLen = 32)
   ```

4. `K` is used as the XChaCha20-Poly1305 key for that single message.

### AEAD Plaintext

After decryption, the plaintext is:

```text
[0..7]   = timestamp (uint64, Unix seconds, big-endian)
[8..end] = password bytes (UTF-8)
```

The daemon then applies:

* Version/msgType check
* Device ID lookup
* Timestamp freshness
* Nonce-based replay protection
* Per-device rate limiting

If everything passes, the password is injected.

---

## Auto-Type Support Notes

NovaKey’s goal is:

> “NovaKey works on most normal apps/fields, but some weird or high-security ones just aren’t supported.”

Current behavior:

* **Linux**

  * On **X11/Xwayland** sessions:

    * Uses `xdotool` and `xclip` to type or paste into the active control.
    * Works well in:

      * Browser address bars and text inputs
      * Terminal emulators
      * Text editors
  * On **pure Wayland** sessions:

    * Keystroke injection is currently **not supported** (see [Known Issues](#known-issues)).
    * The daemon logs a clear message and focuses on clipboard behavior where possible.

* **macOS**

  * Uses AppleScript / Accessibility APIs to simulate paste/typing in the focused control.
  * Requires the user to grant Accessibility / Input permissions in System Settings.

* **Windows**

  * Uses a mixture of clipboard and standard input APIs.
  * Attempts text-control specific messaging where safe, then falls back to synthetic typing.
  * Works in:

    * Notepad and typical desktop apps
    * PowerShell ISE
    * Browser address bars / text fields
  * Some elevated / secure desktops may not accept synthetic input.

> **Lock screens, pre-boot PINs, BitLocker, login screens, DMs, etc.**
> These are *future targets* and may require OS-specific hacks or may be impossible in secure configurations. They are not advertised as working today.

---

## Roadmap

Features planned or on deck:

| Feature                                  | Status       |
| ---------------------------------------- | ------------ |
| Companion mobile app (iOS/Android)       | Planned      |
| Smooth QR-based pairing UX               | In progress  |
| Installer / service packaging per OS     | Planned      |
| GUI tray icon & config UI                | Planned      |
| TOTP / MFA code support                  | Planned      |
| Optional “approve before typing” prompts | Planned      |
| Better lock/login-screen integration     | Experimental |

(ML-KEM-768 / Kyber and protocol v3 are **already implemented**.)

---

## Security Notes (Current Implementation)

### Design Principles

* **Local-first by design** – no cloud service, no external relays.
* **Per-device authentication** – every message is bound to a device ID with its own key.
* **Post-quantum aware transport** – KEM-based key establishment (ML-KEM-768) plus modern AEAD.
* **Defense-in-depth** – timestamps, nonces, replay cache, rate limiting, and strict framing.

### Cryptography & Transport

* **Per-device symmetric keys** in `devices.json`.
* **ML-KEM-768** to derive a per-message secret.
* **HKDF-SHA-256** to derive the AEAD session key from `(kemShared, deviceKey)`.
* **XChaCha20-Poly1305 AEAD** for confidentiality and integrity.
* **Header authenticated via AAD** to bind version, message type, and device ID to the ciphertext.
* **No plaintext acceptance** – frames must decrypt and pass all checks or they are discarded.

### Freshness, Replay & Abuse Prevention

* **Timestamp validation**
  Messages are accepted only within a limited time window and a small clock skew.

* **Nonce-based replay protection**
  Each `(deviceID, nonce)` pair is tracked; reuse is rejected.

* **Per-device rate limiting**
  Each device gets a capped number of requests per minute (configurable) to prevent abuse.

### Secret Handling

* Secrets are decrypted in memory and only exist for the duration of handling a single request.
* Password previews in logs are truncated (`"Sup..." (len=23)`), not fully printed.
* No secrets are logged or stored on disk by the daemon.

### Network Trust Model

* Default listen address is `127.0.0.1:60768` (local only).
* You may opt-in to `0.0.0.0:60768` to allow LAN access (e.g. from a phone).
* It is assumed that:

  * The host OS is not fully compromised.
  * The local network is at least semi-trusted or protected (e.g. via WPA2, VPN, etc.).

---

## Build from Source

You’ll need:

* Go (1.21+ recommended)
* Git (if cloning)
* Standard Go build toolchain

### Clone

```bash
git clone https://github.com/OsbornePro/NovaKey.git
cd NovaKey
```

### Build (Linux/macOS, Bash)

From repo root:

```bash
# Build for Linux
./build.sh -t linux

# Build for macOS (run this on a Mac)
./build.sh -t darwin

# Build for Windows (cross-compile, or run on Windows with bash)
./build.sh -t windows
```

Artifacts are written to `./dist/`, for example:

* `dist/novakey-linux-amd64`
* `dist/NovaKey-darwin-amd64`, `dist/NovaKey-darwin-arm64`
* `dist/NovaKey.exe`

You can also build the helper tools individually:

```bash
go build -o dist/nvclient ./cmd/nvclient
go build -o dist/nvpair   ./cmd/nvpair
```

### Build (Windows, PowerShell)

Example:

```powershell
Set-Location -Path "C:\Path\To\NovaKey"
.\build.ps1 -Target windows -FileName NovaKey.exe
```

---

## Running NovaKey

1. Ensure `server_config.json` exists (NovaKey will default some values).

2. Start the daemon from the directory containing your config files:

   ```bash
   ./dist/novakey-linux-amd64
   ```

   Example log:

   ```text
   2025/12/13 12:41:18 server keys file /.../server_keys.json not found; generating new Kyber keypair
   2025/12/13 12:41:18 Generated new server Kyber keys at /.../server_keys.json
   Loaded 1 device keys from /.../devices.json
   2025/12/13 12:41:18 NovaKey (Linux) service starting (listener=0.0.0.0:60768)
   2025/12/13 12:41:18 NovaKey (Linux) service listening on 0.0.0.0:60768
   ```

3. Use `nvpair` to create a device and get pairing info.

4. Use `nvclient` (or your phone app) with that pairing info to send a test password.

5. Focus a text field and watch NovaKey type it for you (subject to the Auto-Type notes and Known Issues).

You can wrap `novakey` in systemd, launchd, or a Windows Service as you prefer.

---

## Logging

Currently, logs are written to stdout/stderr by default.

Typical patterns:

* **Linux (manual run):**

  ```bash
  ./dist/novakey-linux-amd64 2>&1 | tee novakey.log
  ```

* **Linux (with systemd):**

  ```bash
  journalctl -u novakey.service
  ```

* **macOS (with launchd):**

  * Configure `StandardOutPath` / `StandardErrorPath` in your plist.
  * Or inspect via `log show` for your service label.

* **Windows:**

  * If run as a console app: logs appear in the console.
  * If wrapped as a service, configure your wrapper to redirect stdout/stderr or log to the Event Log.

---

## Contributing

We welcome contributions! Please:

1. Fork the repository and create a feature branch (`git checkout -b feat/your-feature`).
2. Write tests (`go test ./...`).
3. Run linters (e.g. `golangci-lint run`) if you use them.
4. Update documentation if you change flags, behavior, or protocol details.
5. Submit a Pull Request and link any relevant issue.

> **NOTE:** All contributions are accepted under the same commercial licence (*the contributor assigns the rights to OsbornePro, LLC.*). By submitting a PR you agree to this arrangement.

---

## Known Issues

### Linux Wayland sessions

On Linux **Wayland** sessions (`XDG_SESSION_TYPE=wayland`), NovaKey:

* **Does** handle the crypto, validation, and clipboard aspects, but
* **Does *not*** currently perform keystroke injection into the focused window.

This is because the current Linux injector relies on X11/Xwayland tooling (`xdotool`, `xclip`), which does not work reliably against native Wayland windows. Rather than silently failing, NovaKey:

* Logs that Wayland keystroke injection is **not implemented yet**, and
* Focuses on what it can safely do (e.g., clipboard behavior).

**Workarounds:**

* Log in using an **Xorg/X11 session** instead of Wayland, or
* Run target apps under **Xwayland** where possible (e.g., `MOZ_ENABLE_WAYLAND=0 firefox`), or
* Use NovaKey in a **clipboard-only** style and paste manually (`Ctrl+V`).

For more detail and ideas for contributors, see **[`KNOWN_ISSUES.md`](KNOWN_ISSUES.md)**.

---

## License

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

NovaKey (by OsbornePro) is licensed under the Apache License, Version 2.0.
See [`LICENSE.md`](LICENSE.md) for the full license text.

Copyright © 2025 OsbornePro – NovaKey

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at:

```text
http://www.apache.org/licenses/LICENSE-2.0
```

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an **"AS IS" BASIS**,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

---

## Contact & Support

* **Product website / purchase:** [https://novakey.app](https://novakey.app)
* **Technical support:** [support@novakey.app](mailto:support@novakey.app)
* **PGP key (for encrypted email):** [https://downloads.osbornepro.com/publickey.asc](https://downloads.osbornepro.com/publickey.asc)
* **Security disclosures:** Review the policy **[HERE](https://github.com/OsbornePro/NovaKey/blob/main/SECURITY.md)** (do **not** open vulnerabilities via GitHub Issues).
* **GitHub issues:** Use the Issues tab for bugs, feature requests, or installation help. Please do not submit security findings as GitHub Issues.

