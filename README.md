# 🔐 NovaKey-Daemon

**NovaKey-Daemon** is a cross-platform Go agent that receives authenticated secrets from a trusted device and injects them into the currently focused text field.

NovaKey prioritizes direct injection into focused controls; clipboard and auto-typing fallbacks are used only when required by OS or focus constraints, and are fully configurable or disableable for environments that require stricter handling.

It’s built for cases where you don’t want to type high-value secrets (*master passwords, recovery keys, etc.*) on your desktop keyboard:

- the secret lives on a trusted device (*e.g. your phone*)
- delivery is encrypted and authenticated
- the daemon injects into the focused control (*with optional clipboard mode when injection is not possible*)

---

## Current Design

### One port, routed by a preface line

NovaKey listens on one TCP address (`listen_addr`, default `0.0.0.0:60768`) and routes each incoming connection by a one-line preface:

- `NOVAK/1 /pair\n` — pairing (*and pairing subroutes*)
- `NOVAK/1 /msg\n` — encrypted approve/arm/disarm/inject messages

Clients must send a route preface line (NOVAK/1 /msg\n or NOVAK/1 /pair\n). Connections without a valid preface are rejected.

### Crypto (Protocol v3)

NovaKey uses:

- **ML-KEM-768** (*Kyber*) for per-message KEM shared secret establishment
- **HKDF-SHA-256** for key derivation
- **XChaCha20-Poly1305** for authenticated encryption
- timestamp freshness checks, replay protection, and per-device rate limiting

---

## Message Model (*Required Inner Frame*)

All `/msg` requests decrypt to the following plaintext structure:

1. **8-byte timestamp** (`uint64`*, big-endian, unix seconds*)
2. **Inner Message Frame v1** (*required*)

The inner frame is:

- versioned (`frame_version = 1`)
- includes `device_id`, `msg_type`, and `payload`
- authenticated by the outer AEAD (*and validated for device-id consistency*)

Supported inner `msg_type` values:

- **Inject** — payload is the secret string
- **Approve** — payload optional/empty (*two-man mode*)
- **Arm** — payload optional JSON: `{"ms":15000}`
- **Disarm** — payload typically empty

Messages that do not contain a valid **Inner Message Frame v1** are rejected.

---

## Injection Outcomes & Client Signaling

After crypto validation and policy gates, NovaKey attempts injection into the currently focused control.

Depending on OS support, permissions, and configuration, one of the following outcomes occurs:

| Outcome | Description | Server Reply |
| ------ | ----------- | ------------ |
| Direct injection | Secret inserted without clipboard or auto-typing fallback | `status=OK`, `reason=ok` |
| Auto-typing fallback | Secret typed programmatically (optional) | `status=OK`, `reason=typing_fallback` |
| Clipboard paste injection | Clipboard set and paste executed by daemon | `status=OK`, `reason=clipboard_fallback` |
| Clipboard-only fallback | Clipboard set; user must paste manually | `status=OK_CLIPBOARD`, `reason=clipboard_fallback` |
| Wayland clipboard fallback | Injection unavailable; clipboard set | `status=OK_CLIPBOARD`, `reason=inject_unavailable_wayland` |

**Clipboard is never touched unless explicitly allowed by configuration.**

---

## Safety controls (optional)

- arming (*“push-to-type”*)
- two-man approval window (*approve then inject*)
- injection safety rules (`allow_newlines`, `max_inject_len`)
- target policy allow/deny lists (*focused app/window*)
- local Arm API (*loopback only, token protected*)
- clipboard policy controls:
  - `allow_clipboard_when_disarmed` (*allows clipboard use when blocked by gates/policy*)
  - `allow_clipboard_on_inject_failure` (*allows clipboard use when injection fails after gates pass; default false*)
- typing fallback control:
  - `allow_typing_fallback` (*allows auto-typing fallback when direct injection is not possible*)
- macOS preference:
  - `macos_prefer_clipboard` (*prefer clipboard paste injection over AppleScript keystroke typing; default true*)

---

## Pairing (single-port)

When there are no paired devices (*missing/empty device store*), the daemon generates a QR code (`novakey-pair.png`) at startup.

Pairing uses the `/pair` route on the same TCP listener. Clients must send the route preface:

```text
NOVAK/1 /pair\n
```

### High-level flow:

1. Client sends a hello JSON line containing a one-time token:

   `{"op":"hello","v":1,"token":"<b64url>"}\n`

2. Server replies with the ML-KEM public key and a short fingerprint:

   `{"op":"server_key","v":1,"kid":"1","kyber_pub_b64":"...","fp16_hex":"...","expires_unix":...}\n`

3. Client verifies `fp16_hex` matches the fingerprint embedded in the QR.

4. Client sends an encrypted register request (*Kyber + XChaCha20-Poly1305*). The server saves the device PSK and reloads device keys.

Pairing output is sensitive (*treat it like a password*).

### Pairing subroutes

NovaKey also supports `/pair/*` subroutes on the same listener (*routed by the same preface line*). These exist for alternative pairing workflows used by clients.

---

## Device Store & Key Vault Behavior

NovaKey stores per-device static keys in the device store referenced by `devices_file` (*default* `devices.json`).

### Windows

Device store is sealed using **DPAPI**.

### macOS / Linux

Device store is stored in one of two forms:

* **Sealed wrapper (*preferred*)**: encrypted-at-rest using an OS keyring–stored sealing key.
* **Plaintext JSON (*explicit opt-in*)**: only used when the OS keyring is unavailable and plaintext storage is explicitly allowed.

**Important:** On some Linux systems (*especially headless services or logins backed by hardware tokens*), the daemon may not be able to access the user keyring from a system service context. In those environments, you may need to explicitly allow plaintext device storage.

Control this with:

* `require_sealed_device_store`:

  * `true` → **fail closed** if the store is not sealed / keyring unavailable (*recommended default*)
  * `false` → allows plaintext `devices.json` only when the keyring is unavailable, using strict `0600` perms (*enable only if you must*)

---

## Configuration

NovaKey supports YAML (*preferred*) or JSON configuration.

### Core Networking & Limits

| Option                 | Default              | Description                                                     |
| ---------------------- | -------------------- | --------------------------------------------------------------- |
| `listen_addr`          | `"0.0.0.0:60768"`    | TCP address NovaKey listens on for `/pair*` and `/msg`.         |
| `max_payload_len`      | `4096`               | Maximum allowed decrypted payload size (*bytes*).               |
| `max_requests_per_min` | `60`                 | Per-device rate limit for accepted `/msg` requests.             |
| `devices_file`         | `"devices.json"`     | Path to the device store containing paired device keys.         |
| `server_keys_file`     | `"server_keys.json"` | Path to the server’s ML-KEM key material. Treated as sensitive. |

### Device store hardening

| Option                        | Default | Description                                                                                                 |
| ----------------------------- | ------- | ----------------------------------------------------------------------------------------------------------- |
| `require_sealed_device_store` | `false` | If true, NovaKey refuses to run when the OS keyring is unavailable or when the devices store is not sealed. |

* If you want it fail-closed by default, set it to `true` in your shipped config.

### Pairing & key management hardening

| Option                        | Default | Description                                                                       |
| ----------------------------- | ------- | --------------------------------------------------------------------------------- |
| `rotate_kyber_keys`           | `false` | If true, rotates the server’s ML-KEM key pair on startup (*requires re-pairing*). |
| `rotate_device_psk_on_repair` | `false` | If true, re-pairing an existing device replaces its stored key.                   |
| `pair_hello_max_per_min`      | `30`    | Per-IP rate limit for `/pair` hello attempts (*in-memory*).                       |

### Logging

| Option          | Default    | Description                                                     |
| --------------- | ---------- | --------------------------------------------------------------- |
| `log_dir`       | `"./logs"` | Directory for rotating log files. Ignored if `log_file` is set. |
| `log_file`      | (*unset*)  | Single log file path. Overrides `log_dir` when set.             |
| `log_rotate_mb` | `10`       | Maximum size (*MB*) of a log file before rotation.              |
| `log_keep`      | `10`       | Number of rotated log files to retain.                          |
| `log_stderr`    | `true`     | If true, logs are also written to stderr.                       |
| `log_redact`    | `true`     | Best-effort redaction of tokens/secrets and long blobs.         |

### Arming (“Push-to-Type”) gate

| Option                  | Default | Description                                                         |
| ----------------------- | ------- | ------------------------------------------------------------------- |
| `arm_duration_ms`       | `20000` | Duration (*ms*) the daemon remains armed after arming is triggered. |
| `arm_consume_on_inject` | `true`  | If true, a successful injection consumes the armed state.           |

### Clipboard policy

| Option                              | Default | Description                                                                          |
| ----------------------------------- | ------- | ------------------------------------------------------------------------------------ |
| `allow_clipboard_when_disarmed`     | `false` | Allows clipboard use when blocked by gates/policy. Use with care.                    |
| `allow_clipboard_on_inject_failure` | `false` | Allows clipboard use when injection fails after gates pass (*Wayland, perms, etc.*). |

### Typing fallback

| Option                  | Default | Description                                                        |
| ----------------------- | ------- | ------------------------------------------------------------------ |
| `allow_typing_fallback` | `true`  | Allows auto-typing fallback when direct injection is not possible. |

### macOS injection preference

| Option                   | Default | Description                                                        |
| ------------------------ | ------- | ------------------------------------------------------------------ |
| `macos_prefer_clipboard` | `true`  | Prefer clipboard paste injection over AppleScript typing on macOS. |

### Injection safety

| Option           | Default | Description                                              |
| ---------------- | ------- | -------------------------------------------------------- |
| `allow_newlines` | `false` | Reject newline characters in secrets when false.         |
| `max_inject_len` | `256`   | Maximum number of characters allowed in a single inject. |

### Two-man approval mode

| Option                      | Default | Description                                                 |
| --------------------------- | ------- | ----------------------------------------------------------- |
| `two_man_enabled`           | `true`  | Requires an approve message before injection is allowed.    |
| `approve_window_ms`         | `15000` | Window (*ms*) after approval in which injection is allowed. |
| `approve_consume_on_inject` | `true`  | If true, approval is consumed after a successful injection. |

### Target policy (*Focused app / window restrictions*)

| Option                   | Default   | Description                                                              |
| ------------------------ | --------- | ------------------------------------------------------------------------ |
| `target_policy_enabled`  | `false`   | Enables focused target enforcement before injection.                     |
| `use_built_in_allowlist` | `false`   | Applies a built-in allowlist when enabled and no explicit rules are set. |
| `allowed_process_names`  | *(empty)* | Allowed process names (*normalized*).                                    |
| `allowed_window_titles`  | *(empty)* | Case-insensitive substrings required in the focused window title.        |
| `denied_process_names`   | *(empty)* | Always-denied processes.                                                 |
| `denied_window_titles`   | *(empty)* | Always-denied window title substrings.                                   |

---

## Recommended defaults

* Keep `listen_addr` on loopback unless you *need* LAN.
* Prefer `require_sealed_device_store: true` (*fail closed*) unless your Linux service environment cannot access the OS keyring.
* Keep arming and two-man enabled for safest operation.
* On Linux Wayland, injection may not be possible; rely on clipboard mode (`allow_clipboard_on_inject_failure`) if you explicitly enable it.

---

## Docs

* `SECURITY.md` — threat model + security properties
* `PROTOCOL.md` — wire formats for `/pair*` and `/msg`

---

## Contact

* Security: `security@novakey.app`
