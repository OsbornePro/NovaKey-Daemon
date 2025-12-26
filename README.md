# 🔐 NovaKey-Daemon

**NovaKey-Daemon** is a cross-platform Go agent that receives authenticated secrets from a trusted device and injects them into the currently focused text field.

It’s built for cases where you don’t want to type high-value secrets (master passwords, recovery keys, etc.) on your desktop keyboard:

- the secret lives on a trusted device (e.g. your phone)
- delivery is encrypted and authenticated
- the daemon injects into the focused control (with optional clipboard fallback when blocked)

---

## Current Design

### One port, two routes

NovaKey listens on one TCP address (`listen_addr`, default `127.0.0.1:60768`) and routes each incoming connection by a one-line preface:

- `NOVAK/1 /pair\n` — pairing
- `NOVAK/1 /msg\n` — encrypted approve/inject messages

If a client does not send the route line, the daemon treats the connection as `/msg` for compatibility.

### Crypto

- **/pair:** one-time pairing token + ML-KEM-768 + HKDF-SHA-256 + XChaCha20-Poly1305 (Pairing Protocol v1)
- **/msg:** ML-KEM-768 + HKDF-SHA-256 + XChaCha20-Poly1305 (Protocol v3)
- timestamp freshness checks, replay protection, and per-device rate limiting

### Safety controls (optional)

- arming (“push-to-type”)
- two-man approval window (typed approve then inject)
- injection safety rules (`allow_newlines`, `max_inject_len`)
- target policy allow/deny lists (focused app/window)
- local Arm API (loopback only, token protected)

---

## Docs

- `SECURITY.md` — threat model + security properties
- `PROTOCOL.md` — wire formats for `/pair` and `/msg`

---

## Pairing (single-port)

When there are no paired devices (missing/empty device store), the daemon generates a QR code (`novakey-pair.png`) at startup.

Pairing uses the `/pair` route on the same TCP listener. Clients **must** send the route preface:

```text
NOVAK/1 /pair\n
````

Flow:

1. Client sends a hello JSON line containing a one-time token:

   * `{"op":"hello","v":1,"token":"<b64url>"}\n`

2. Server replies with the ML-KEM public key and a short fingerprint:

   * `{"op":"server_key","v":1,"kid":"1","kyber_pub_b64":"...","fp16_hex":"...","expires_unix":...}\n`

3. Client verifies `fp16_hex` matches the fingerprint embedded in the QR.

4. Client sends an encrypted register request. The server saves the device PSK and reloads device keys.

Pairing output is sensitive (treat it like a password).

---

## Configuration

NovaKey supports YAML (preferred) or JSON.

Core:

* `listen_addr`
* `max_payload_len`
* `max_requests_per_min`
* `devices_file`
* `server_keys_file`

Safety gates:

* `arm_enabled`
* `arm_duration_ms`
* `arm_consume_on_inject`
* `two_man_enabled`
* `approve_window_ms`
* `approve_consume_on_inject`
* `allow_clipboard_when_disarmed`

Target policy:

* `target_policy_enabled`
* `use_built_in_allowlist`
* `allowed_process_names`
* `allowed_window_titles`
* `denied_process_names`
* `denied_window_titles`

Arm API:

* `arm_api_enabled`
* `arm_listen_addr` (must be loopback)
* `arm_token_file`
* `arm_token_header`

Logging:

* `log_dir` or `log_file`
* `log_rotate_mb`
* `log_keep`
* `log_stderr`
* `log_redact`

---

## Contact

* Security: `security@novakey.app`
