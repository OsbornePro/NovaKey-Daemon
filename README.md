# 🔐 NovaKey-Daemon

**NovaKey-Daemon** is a lightweight, cross-platform Go agent that turns your computer into a secure, authenticated password-injection endpoint.

It’s designed for cases where you don’t want to type high-value secrets (master passwords, recovery keys, etc.) on your desktop keyboard:

- the secret lives on a trusted device (e.g. your phone)
- delivery is encrypted and authenticated
- the daemon injects into the currently focused text field

> Secrets do not traverse the network in plaintext.

---

## Current Status

- The daemon runs on Linux / Windows / macOS.
- Pairing + message transport are implemented on a **single TCP port** (default `:60768`).
- A loopback-only Arm API exists for local control (optional).

---

## Security Review Invited

NovaKey uses **ML-KEM-768 + HKDF-SHA-256 + XChaCha20-Poly1305**, with freshness checks, replay protection, and per-device rate limiting.

Safety controls:

- arming (“push-to-type”)
- two-man approval gate (per-device approve window)
- injection safety rules (`allow_newlines`, `max_inject_len`)
- optional target policy allow/deny lists

Docs:
- Protocol format: `PROTOCOL.md`
- Security model: `SECURITY.md`

---

## Overview

NovaKey listens on `listen_addr` (default `127.0.0.1:60768`) and routes each inbound TCP connection by a short route line:

- `NOVAK/1 /pair` — pairing exchange
- `NOVAK/1 /msg`  — encrypted message exchange (inject/approve)

For backward compatibility, clients that do not send a route line are treated as `/msg`.

---

## Pairing

If no devices are paired (missing/empty `devices.json`), the daemon generates a QR code image (`novakey-pair.png`).

Scanning the QR provides the phone app (or a pairing tool) with:

- daemon host + port
- a short-lived **pair token**
- a fingerprint of the daemon’s ML-KEM public key (sanity check)
- expiration timestamp

The client then connects to the daemon on the **same port** and performs pairing via `/pair`. Pairing returns the device ID + a per-device secret + the daemon public key material needed to send encrypted `/msg` requests.

Treat pairing output as sensitive (like a password).

---

## Configuration

NovaKey supports YAML (preferred) or JSON.

Core fields:

- `listen_addr`
- `max_payload_len`
- `max_requests_per_min`
- `devices_file`
- `server_keys_file`

Arming / two-man:

- `arm_enabled`
- `arm_duration_ms`
- `arm_consume_on_inject`
- `two_man_enabled`
- `approve_window_ms`
- `approve_consume_on_inject`
- `allow_clipboard_when_disarmed`

Arm API (optional):

- `arm_api_enabled`
- `arm_listen_addr` (must be loopback)
- `arm_token_file`
- `arm_token_header`

Logging:

- `log_dir` or `log_file`
- `log_rotate_mb`
- `log_keep`
- `log_stderr`
- `log_redact`

---

## Running

Run the daemon in a logged-in desktop session for reliable injection behavior.

- On startup:
  - server keys are loaded/created (`server_keys.json`)
  - devices are loaded (`devices.json`)
  - if not paired, a pairing QR is generated

---

## Contact & Support

- Security disclosures: see `SECURITY.md`
- Email: `security@novakey.app`
