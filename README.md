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

NovaKey supports YAML (preferred) or JSON configuration.
Below is the complete list of supported options, their defaults, and what they control.

> **Defaults shown here match the defaults you’ve chosen** (your sample YAML + code defaults).

---

### Core Networking & Limits

| Option                 | Default              | Description                                                                                                                                              |
| ---------------------- | -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `listen_addr`          | `"127.0.0.1:60768"`  | TCP address NovaKey listens on for `/pair` and `/msg`. Use loopback for local-only operation. Exposing to LAN is supported but increases attack surface. |
| `max_payload_len`      | `4096`               | Maximum allowed decrypted payload size (bytes) for injected secrets. Prevents abuse and memory pressure.                                                 |
| `max_requests_per_min` | `60`                 | Per-device rate limit for accepted `/msg` requests. Helps prevent brute-force and flooding from a compromised device.                                    |
| `devices_file`         | `"devices.json"`     | Path to the device store containing paired device PSKs. On non-Windows, this is sealed when possible.                                                    |
| `server_keys_file`     | `"server_keys.json"` | Path to the server’s ML-KEM key material. Treated as sensitive.                                                                                          |

---

### Pairing & Key Management Hardening

| Option                        | Default | Description                                                                                                                                                                                                                                    |
| ----------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `rotate_kyber_keys`           | `false` | If true, rotates the server’s ML-KEM key pair on startup. Requires re-pairing existing devices.                                                                                                                                                |
| `rotate_device_psk_on_repair` | `false` | If true, re-pairing an existing device replaces its PSK instead of reusing it.                                                                                                                                                                 |
| `pair_hello_max_per_min`      | `30`    | Per-IP rate limit for `/pair` hello attempts (in-memory). Mitigates LAN brute-force or QR-token racing.                                                                                                                                        |
| `require_sealed_device_store` | `true`  | **Fail-closed safety flag.** If true, NovaKey refuses to (a) fall back to plaintext device storage when the OS keyring is unavailable and (b) load legacy plaintext `devices.json`. If sealing is unavailable, the daemon exits with an error. |

---

### Logging

| Option          | Default    | Description                                                                                                                                       |
| --------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `log_dir`       | `"./logs"` | Directory for rotating log files. Ignored if `log_file` is set.                                                                                   |
| `log_file`      | *(unset)*  | Single log file path. Overrides `log_dir` when set.                                                                                               |
| `log_rotate_mb` | `10`       | Maximum size (MB) of a log file before rotation.                                                                                                  |
| `log_keep`      | `10`       | Number of rotated log files to retain.                                                                                                            |
| `log_stderr`    | `true`     | If true, logs are also written to stderr. Often disabled on Linux service installs.                                                               |
| `log_redact`    | `true`     | Enables best-effort redaction of secrets, tokens, URL query params, and long base64/hex blobs in logs. Logs should still be treated as sensitive. |

---

### Arming (“Push-to-Type”) Gate

| Option                          | Default | Description                                                                                |
| ------------------------------- | ------- | ------------------------------------------------------------------------------------------ |
| `arm_enabled`                   | `true`  | Enables the arming gate. Injection is blocked unless locally armed.                        |
| `arm_duration_ms`               | `20000` | Duration (ms) the daemon remains armed after arming is triggered.                          |
| `arm_consume_on_inject`         | `true`  | If true, a successful injection consumes the armed state immediately.                      |
| `allow_clipboard_when_disarmed` | `false` | If true, allows clipboard fallback even when not armed. Enabling this increases leak risk. |

---

### Local Arm API (Loopback Only)

| Option             | Default             | Description                                                                                             |
| ------------------ | ------------------- | ------------------------------------------------------------------------------------------------------- |
| `arm_api_enabled`  | `true`              | Enables the local HTTP arm API.                                                                         |
| `arm_listen_addr`  | `"127.0.0.1:60769"` | Address the Arm API binds to. **Must resolve to loopback**, or the daemon will refuse to start the API. |
| `arm_token_file`   | `"arm_token.txt"`   | Path to the Arm API authentication token file. Contents are treated as secret.                          |
| `arm_token_header` | `"X-NovaKey-Token"` | HTTP header name used to supply the Arm API token.                                                      |

---

### Injection Safety

| Option           | Default | Description                                                                                             |
| ---------------- | ------- | ------------------------------------------------------------------------------------------------------- |
| `allow_newlines` | `false` | If false, newline characters are rejected in injected secrets. Prevents multi-line injection accidents. |
| `max_inject_len` | `256`   | Maximum number of characters that may be injected in a single request.                                  |

---

### Two-Man Approval Mode

| Option                         | Default                 | Description                                                                 |
| ------------------------------ | ----------------------- | --------------------------------------------------------------------------- |
| `two_man_enabled`              | `true`                  | Requires a separate approve action before injection is allowed.             |
| `approve_window_ms`            | `15000`                 | Time window (ms) after approval during which injection is allowed.          |
| `approve_consume_on_inject`    | `true`                  | If true, approval is consumed after a successful injection.                 |
| `approve_magic`                | `"__NOVAKEY_APPROVE__"` | Legacy typed approve magic string (still supported but discouraged).        |
| `legacy_approve_magic_enabled` | `false`                 | If true, allows legacy approve magic in addition to typed approve messages. |

---

### Target Policy (Focused App / Window Restrictions)

| Option                   | Default   | Description                                                                                     |
| ------------------------ | --------- | ----------------------------------------------------------------------------------------------- |
| `target_policy_enabled`  | `false`   | Enables focused target enforcement before injection.                                            |
| `use_built_in_allowlist` | `false`   | If true, applies NovaKey’s built-in safe allowlist when no explicit rules are set.              |
| `allowed_process_names`  | *(empty)* | List of allowed process names. Normalized (lowercase, path stripped, `.exe` / `.app` stripped). |
| `allowed_window_titles`  | *(empty)* | Case-insensitive substrings that must appear in the focused window title.                       |
| `denied_process_names`   | *(empty)* | Processes that are always denied, even if otherwise allowed.                                    |
| `denied_window_titles`   | *(empty)* | Window title substrings that are always denied.                                                 |

**Normalization note:**
Process names are normalized by:

* lowercasing
* stripping path components
* removing `.exe` (Windows) and `.app` (macOS)

This avoids configuration errors caused by platform-specific naming.

---

## Recommended defaults by environment

Use these as **starting points**. They are intentionally opinionated.

| Environment                                | Recommended `listen_addr`              | Recommended safety gates                                                             | Notes                                                                                                    |
| ------------------------------------------ | -------------------------------------- | ------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------- |
| **Personal desktop / laptop** (most users) | `"127.0.0.1:60768"`                    | `arm_enabled: true`, `two_man_enabled: true`                                         | Safest default: local-only + “push-to-type” + approval window.                                           |
| **Home LAN** (phone → desktop over Wi-Fi)  | `"0.0.0.0:60768"` (or specific LAN IP) | `arm_enabled: true`, `two_man_enabled: true`, `require_sealed_device_store: true`    | Only do this on a trusted network. Add target policy allowlist for browsers/password managers.           |
| **Workstation shared with others**         | `"127.0.0.1:60768"`                    | `arm_enabled: true`, `two_man_enabled: true`, `allow_clipboard_when_disarmed: false` | Prefer loopback-only. Reduce side-channels and accidental injection.                                     |
| **Headless/server** (not recommended)      | `"127.0.0.1:60768"`                    | `arm_enabled: true`, `two_man_enabled: true`                                         | NovaKey is designed for desktop sessions with a focused text control. Consider not deploying on servers. |

---

## If you only change 5 settings

If you don’t want to think about all of this, start here:

1. **Keep it local unless you truly need LAN:**
   `listen_addr: "127.0.0.1:60768"`

2. **Fail closed on insecure device storage:**
   `require_sealed_device_store: true`

3. **Keep injection gated:**
   `arm_enabled: true`

4. **Require an approve step:**
   `two_man_enabled: true`
   (and keep `approve_consume_on_inject: true`)

5. **Disable clipboard fallback unless you explicitly want it:**
   `allow_clipboard_when_disarmed: false`

Then optionally add:

* `target_policy_enabled: true` with an allowlist for browsers / password managers
* `log_redact: true` (keep it on) and treat logs as sensitive anyway

---

## Example configuration (your sample)

```yaml
rotate_kyber_keys: false
listen_addr: "0.0.0.0:60768"
max_payload_len: 4096
max_requests_per_min: 60
devices_file: "devices.json"
server_keys_file: "server_keys.json"

log_dir: "./logs"          # or "/var/log/novakey" on Linux
# log_file: "./logs/novakey.log"   # optional; overrides log_dir if set
log_rotate_mb: 10
log_keep: 10
log_stderr: true
log_redact: true

require_sealed_device_store: true

target_policy_enabled: false
use_built_in_allowlist: false
allowed_process_names:
  - msedge
  - chrome
  - chromium
  - brave
  - firefox
  - safari
  - opera
  - vivaldi
  - duckduckgo
  - ecosia
  - aloha
  - 1password
  - bitwarden
  - lastpass
  - dashlane
  - keeper
  - nordpass
  - protonpass
  - roboform
  - totalpassword
  - avira
  - norton
  - aura
  - notepad
  - textedit
  - gedit
  - kate
allowed_window_titles: []
denied_process_names: []
denied_window_titles: []

arm_enabled: false
arm_duration_ms: 20000
arm_consume_on_inject: true
allow_clipboard_when_disarmed: true

arm_api_enabled: true
arm_listen_addr: "127.0.0.1:60769"
arm_token_file: "arm_token.txt"
arm_token_header: "X-NovaKey-Token"

allow_newlines: false
max_inject_len: 256

two_man_enabled: true
approve_window_ms: 15000
approve_consume_on_inject: true
```

> Note: In the sample above, `arm_enabled: false` and `allow_clipboard_when_disarmed: true` are less safe. For most users, prefer `arm_enabled: true` and `allow_clipboard_when_disarmed: false`.

---

## Contact

* Security: `security@novakey.app`
