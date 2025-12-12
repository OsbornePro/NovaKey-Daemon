# 🔐 NovaKey by OsbornePro

**What is NovaKey?**
*NovaKey is a lightweight, cross-platform Go agent that turns your computer into a secure, authenticated password-injection endpoint.*

**Why would I need this?**
Even with a password manager you still need a master password (or other high-value secret). That secret is often the weakest link—either memorised, re-used, or stored in sketchy ways.

NovaKey aims to eliminate “manual typing” of those secrets:

* Your real master password lives only on a trusted device (e.g. your phone).
* You never type it manually on the keyboard.
* Delivery is encrypted and authenticated with modern AEAD (XChaCha20-Poly1305) and per-device keys.
* The NovaKey daemon injects the secret into the currently focused text field on your desktop.

> **Key point:** The secret never passes through the keyboard, and never traverses the network in plaintext.

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
* [License](#license)
* [Contact & Support](#contact--support)

---

## Overview

The NovaKey service (`novakey`) runs on a workstation (*Windows, macOS, or Linux*). It creates a TCP listener (default `127.0.0.1:60768`). One or more clients (e.g. a future mobile app, or the included `nvclient` test tool) connect to this listener, send an encrypted payload, and NovaKey:

1. **Authenticates** the device using a per-device symmetric key.
2. **Decrypts & validates** the request using XChaCha20-Poly1305 with:

   * Per-device keys,
   * Timestamps,
   * Nonce-based replay protection,
   * Per-device rate limiting.
3. **Injects** the resulting password into the currently focused control on the desktop.

All cryptographic operations are done locally; there is no cloud service or third-party relay.

> **Post-quantum note:** Future versions are planned to add a Kyber-based KEM on top of the existing symmetric layer. The current code uses per-device symmetric keys only.

---

## Current Capabilities

| ✅ | Capability                                                                     |
| - | ------------------------------------------------------------------------------ |
| ✅ | Cross-platform daemon (`novakey`) for Linux, macOS, and Windows                |
| ✅ | Encrypted & authenticated password delivery using XChaCha20-Poly1305           |
| ✅ | Per-device keys and device IDs stored in `devices.json`                        |
| ✅ | Message freshness (timestamp) validation                                       |
| ✅ | Nonce-based replay protection per device                                       |
| ✅ | Per-device rate limiting (requests/min)                                        |
| ✅ | Configurable listen address, payload size, and limits via `server_config.json` |
| ✅ | Simple CLI test client (`nvclient`)                                            |
| ✅ | CLI device pairing / key management tool (`nvpair`)                            |

---

## Command-line Tools

All commands live under `cmd/` and are built into binaries under `dist/` by `build.sh` / `build.ps1`.

### `novakey` – the daemon

The main service process:

* Loads configuration from `server_config.json`.
* Loads per-device keys from `devices.json`.
* Listens on the configured TCP address (default `127.0.0.1:60768`).
* For each incoming connection:

  * Reads a single encrypted frame.
  * Decrypts and validates it.
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

By default it logs to stdout/stderr; you can wrap it in systemd / launchd / Windows Service yourself if you want it to run as a background service.

---

### `nvclient` – reference/test client

A simple CLI client that speaks the NovaKey protocol. It’s useful for:

* Testing the daemon.
* Experimenting with passwords and device IDs.
* Serving as a reference implementation for other clients (e.g. mobile apps).

Usage:

```bash
./dist/nvclient \
  -addr 127.0.0.1:60768 \
  -device-id roberts-phone \
  -key-hex 7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234 \
  -password "SuperStrongPassword123!"
```

Flags:

* `-addr` – address of the NovaKey daemon (e.g. `127.0.0.1:60768` or `192.168.x.x:60768`)
* `-device-id` – device ID that must exist in `devices.json`
* `-key-hex` – 32-byte per-device key in hex (matches `key_hex` in `devices.json`)
* `-password` – password/secret string to send and inject

---

### `nvpair` – device pairing & key management

A tiny helper that edits `devices.json` for you:

* Generates a random 32-byte key.
* Adds a new device entry, or updates an existing one (with `-force`).
* Prints out the device ID and key, ready to be used by `nvclient` or a real client.

Example:

```bash
./dist/nvpair -id roberts-phone
```

Output (example):

```text
Added new device "roberts-phone" to /path/to/devices.json
------------------------------------------------------------
 Pairing info
------------------------------------------------------------
Device ID : roberts-phone
Key (hex) : 7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234

Use these values with nvclient or your real client, e.g.:
  nvclient -addr 127.0.0.1:60768 -device-id "roberts-phone" -key-hex 7f0c9e... -password "..."
```

Flags:

* `-devices-file` – path to `devices.json` (default: `devices.json` in CWD)
* `-id` – device ID to add or update (required)
* `-force` – overwrite existing device with the same ID

---

## Configuration Files

### `server_config.json`

Controls how the daemon listens and enforces limits.

Example:

```json
{
  "listen_addr": "127.0.0.1:60768",
  "max_payload_len": 4096,
  "max_requests_per_min": 60,
  "devices_file": "devices.json"
}
```

* `listen_addr` – TCP address to bind to.

  * `127.0.0.1:60768` – **local only** (default, safest).
  * `0.0.0.0:60768` – listen on all interfaces (for LAN usage).
* `max_payload_len` – max allowed payload bytes (before decryption).
* `max_requests_per_min` – per-device rate limit.
* `devices_file` – path to the `devices.json` file.

> **Important:** If you expose `0.0.0.0:60768`, ensure your firewall is configured appropriately. NovaKey uses per-device keys and replay protection, but the port is still a high-value interface.

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
    },
    {
      "id": "roberts-tablet",
      "key_hex": "b8e167a0c4f1d2a3f5e4c3b2a19087654321ffeeddccbbaa9988776655443322"
    }
  ]
}
```

You normally won’t edit this by hand; use `nvpair` instead.

---

## Protocol & Crypto Stack

**Symmetric encryption & auth (current):**

* **Cipher:** XChaCha20-Poly1305 (via `golang.org/x/crypto/chacha20poly1305.NewX`)
* **Per-device keys:** 32-byte symmetric keys stored in `devices.json`
* **Header (AAD):**

  * `version` (currently 2)
  * `msgType` (1 = password frame)
  * `deviceID` length + bytes
* **Plaintext:**

  * 8-byte timestamp (Unix seconds, big-endian)
  * UTF-8 password bytes

**On-wire framing:**

* Outer frame: `[ u16 length ][ length bytes payload ]`
* Payload:

  * `[version][msgType][idLen][deviceID][nonce][ciphertext]`
  * `nonce` is 24 random bytes (XChaCha20)
  * `ciphertext` is AEAD-encrypted plaintext

**Validation on the server:**

* Version and message type check.
* Device ID lookup (`devices.json`).
* Timestamp window enforcement (freshness / clock-skew checks).
* Nonce-based replay cache per device.
* Per-device rate limiting (requests/min).

**Planned / Roadmap crypto (not yet implemented):**

* Add a Kyber-based KEM for post-quantum key exchange on top of the per-device identity.
* Optionally derive per-session keys and rotate them.

For more detail, see the protocol document (`PROTOCOL.md`) once added.

---

## Auto-Type Support Notes

NovaKey’s goal is:

> “NovaKey works on most normal apps/fields, but some weird or high-security ones just aren’t supported.”

Current behavior:

* **Linux**

  * Uses clipboard + `xdotool` (and related utilities) to type/paste into the active control.
  * Works well in:

    * Browser address bars and text inputs
    * Terminal emulators
    * Text editors
  * Wayland/desktop specifics may affect behavior; some environments restrict global key injection.

* **macOS**

  * Uses macOS automation/accessibility APIs to simulate paste/typing in the focused control.
  * Requires the user to grant Accessibility / Input permissions in System Settings.

* **Windows**

  * Uses the standard Windows input APIs and clipboard on the logged-in desktop session.
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

| Feature                                         | Status       |
| ----------------------------------------------- | ------------ |
| Companion mobile app (iOS/Android)              | Planned      |
| QR-based pairing flow                           | Planned      |
| Post-quantum KEM (Kyber) on top of current AEAD | Planned      |
| Installer / service packaging per OS            | Planned      |
| GUI tray icon & config UI                       | Planned      |
| TOTP / MFA code support                         | Planned      |
| Optional “approve before typing” prompts        | Planned      |
| Better lock/login-screen integration            | Experimental |

---

## Security Notes (Current Implementation)

### Design Principles

* **Local-first by design** – no cloud service, no external relays.
* **Per-device authentication** – every message is bound to a device ID with its own key.
* **Encrypted & authenticated transport** – all traffic is AEAD-protected.
* **Defense-in-depth** – timestamps, nonces, replay cache, rate limiting, and strict framing.

### Cryptography & Transport

* **Per-device symmetric keys** in `devices.json`.
* **XChaCha20-Poly1305 AEAD** for confidentiality and integrity.
* **Header authenticated via AAD** to bind device ID and message type to the ciphertext.
* **No plaintext acceptance** – frames must decrypt and pass all checks or they are discarded.

### Freshness, Replay & Abuse Prevention

* **Timestamp validation**
  Messages are only accepted within a limited time window and with a reasonable clock skew.

* **Nonce-based replay protection**
  Each `(deviceID, nonce)` pair is tracked; reuse is rejected.

* **Per-device rate limiting**
  Each device gets a capped number of requests per minute (configurable) to prevent abuse.

### Secret Handling

* Secrets are decrypted in memory and only exist for the duration of handling a single request.
* Password previews in logs are truncated (`"Sup..." (len=23)`), not fully printed.
* No secrets are logged or passed via command-line arguments on the server side.

### Network Trust Model

* Default listen address is `127.0.0.1:60768` (local only).
* You may opt-in to `0.0.0.0:60768` to allow LAN access (e.g. from a phone).
* It is assumed that:

  * The host OS is not fully compromised.
  * Local network is at least semi-trusted or protected (e.g. via WPA2, VPN, etc.).

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
* `dist/nvclient` (when built separately with `go build -o dist/nvclient ./cmd/nvclient`)
* `dist/nvpair` (when built with `go build -o dist/nvpair ./cmd/nvpair`)

### Build (Windows, PowerShell)

A simple example (you can adjust as needed):

```powershell
Set-Location C:\Path\To\NovaKey
.\build.ps1 -Target windows -FileName NovaKey.exe
```

---

## Running NovaKey

1. Ensure `server_config.json` and `devices.json` exist in the working directory.

2. Start the daemon:

   ```bash
   ./dist/novakey-linux-amd64
   ```

   Example log:

   ```text
   Loaded server config from /.../server_config.json
   Loaded 2 device keys from /.../devices.json
   2025/12/12 15:19:50 NovaKey (Linux) service starting (listener=127.0.0.1:60768)
   2025/12/12 15:19:50 NovaKey (Linux) service listening on 127.0.0.1:60768
   ```

3. Use `nvpair` to create a device, and `nvclient` to send a test password.

4. Focus a text field and watch NovaKey type it for you.

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
  * If wrapped as a service, configure your service wrapper to redirect stdout/stderr or log to the Event Log.

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

## License

NovaKey is **proprietary commercial software**. See `EULA.md` for the full terms.

The source code in this repository is provided **as-is** solely for the purpose of building the binary; redistribution of the source or compiled binaries is prohibited without a separate written licence from OsbornePro LLC.

---

## Contact & Support

* **Product website / purchase:** [https://novakey.app](https://novakey.app)
* **Technical support:** [support@novakey.app](mailto:support@novakey.app)
* **PGP key (for encrypted email):** [https://downloads.osbornepro.com/publickey.asc](https://downloads.osbornepro.com/publickey.asc)
* **Security disclosures:** Review the policy **[HERE](https://github.com/OsbornePro/NovaKey/blob/main/SECURITY.md)** (do **not** open vulnerabilities via GitHub Issues).
* **GitHub issues:** Use the Issues tab for bugs, feature requests, or installation help. Please do not submit security findings as GitHub Issues.

