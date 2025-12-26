# 🔐 NovaKey-Daemon

**NovaKey-Daemon** is a cross-platform Go agent that receives authenticated secrets from a trusted device and injects them into the currently focused text field.

It’s built for cases where you don’t want to type high-value secrets (master passwords, recovery keys, etc.) on your desktop keyboard:

- the secret lives on a trusted device (e.g. your phone)
- delivery is encrypted and authenticated
- the daemon injects into the focused control (with optional clipboard fallback when blocked)

---

## Current Status

- Linux / Windows / macOS daemon
- **One TCP listener** (`listen_addr`, default `127.0.0.1:60768`)
- Connection routing via a single-line preface:
  - `NOVAK/1 /pair`
  - `NOVAK/1 /msg`
- `/msg` uses **ML-KEM-768 + HKDF-SHA-256 + XChaCha20-Poly1305** (protocol v3)
- Optional loopback-only Arm API for local arming/disarming (token protected)

---

## Docs

- `SECURITY.md` — threat model + security properties
- `PROTOCOL.md` — current wire format for `/pair` and `/msg`

---

## How it works

1) NovaKey starts and loads/creates `server_keys.json` (ML-KEM-768 keypair).
2) NovaKey loads device secrets from `devices.json`.
3) If there are no paired devices, NovaKey generates a QR (`novakey-pair.png`) and waits for pairing.
4) Clients connect to the daemon on the same TCP port and choose a route:
   - `/pair` to pair
   - `/msg` to send encrypted approve/inject requests

---

## Pairing (single-port)

Pairing uses the `/pair` route on the same listener.

**Client → server (plaintext JSON line):**
`{"op":"hello","v":1,"token":"<b64url>"}\n`

**Server → client (plaintext JSON line):**
`{"op":"server_key","v":1,"kid":"1","kyber_pub_b64":"...","fp16_hex":"...","expires_unix":...}\n`

Then the client sends an encrypted register message using ML-KEM + XChaCha20-Poly1305. If `device_id` / `device_key_hex` are omitted, the server assigns them and saves `devices.json`.

Treat pairing output as sensitive.

---

## Configuration

NovaKey supports YAML (preferred) or JSON.

Core fields:

- `listen_addr`
- `max_payload_len`
- `max_requests_per_min`
- `devices_file`
- `server_keys_file`

Safety gates:

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

## Contact

- Security: `security@novakey.app`
