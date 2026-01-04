# Configuration

NovaKey-Daemon supports YAML (**preferred**) or JSON configuration files:

* `server_config.yaml`
* `server_config.yml`
* `server_config.json`

The daemon loads configuration from its **WorkingDirectory**.
Installers set this directory so that **relative paths work correctly**.

Relative paths such as `devices.json`, `server_keys.json`, `./logs`, and `arm_token.txt`
are resolved relative to the WorkingDirectory.

---

## Config file selection order

If multiple config files exist, NovaKey loads the first one found:

1. `server_config.yaml`
2. `server_config.yml`
3. `server_config.json`

---

## Core networking & limits

### `listen_addr` (string)

Address and port to bind the TCP listener.

**Default (code):**

```
127.0.0.1:60768
```

**Common values:**

* Local only (safest): `127.0.0.1:60768`
* LAN access: `0.0.0.0:60768`
* Specific LAN IP: `10.0.0.10:60768`

> Binding to `0.0.0.0` increases attack surface.
> Use target policy if listening on LAN.

---

### `max_payload_len` (int)

Maximum request payload size in bytes.

**Default:** `4096`

---

### `max_requests_per_min` (int)

Per-client rate limit for incoming requests.

**Default:** `60`

---

## Key & device storage

### `devices_file` (string)

Path to the device store.
Contains paired device identities and metadata.

**Default:** `devices.json`

May be:

* Sealed/encrypted via OS keyring
* Plain JSON fallback if sealed storage fails (platform-dependent)

---

### `server_keys_file` (string)

Path to server cryptographic key material.

Includes:

* ML-KEM (Kyber) public/private keys
* Long-lived server identity

**Default:** `server_keys.json`

---

### `require_sealed_device_store` (bool)

If `true`, NovaKey **fails closed** when sealed/secure storage cannot be unlocked.

This is a **security-critical option**.

**Default (code):** `false`
**Recommended:** `true`
**Your shipped YAML:** `true`

Linux note:

* Hardware-backed auth (YubiKey / PAM) may block keyring unlock
* Cancelling unlock repeatedly may require reinstall or manual cleanup

---

## Pairing hardening

### `rotate_kyber_keys` (bool)

If `true`, server Kyber keys rotate **on every service restart**.

Effect:

* Forces **full re-pairing** of all devices
* Invalidates all existing pairings

**Default:** `false`

---

### `rotate_device_psk_on_repair` (bool)

If `true`, device PSKs rotate during re-pair / repair flows.

**Default:** `false`

---

### `pair_hello_max_per_min` (int)

Per-IP rate limit for `/pair` handshake “hello” messages.

This limiter is **in-memory only**.

**Default:** `30`

---

## Logging

> Logs may be redacted but should still be treated as sensitive.

### `log_dir` (string)

Directory for log files.

May be relative (`./logs`) or absolute.

**Default behavior:** stderr logging only
**Common:** `./logs`

---

### `log_file` (string)

Explicit log file path.
Overrides `log_dir` if set.

**Default:** unset

---

### `log_rotate_mb` (int)

Maximum log file size before rotation.

**Default:** `10`

---

### `log_keep` (int)

Number of rotated logs to retain.

**Default:** `10`

---

### `log_stderr` (bool)

Emit logs to stderr.

**Default:** `true`

---

### `log_redact` (bool)

Redacts secrets and sensitive fields from logs (best effort).

**Default:** `true`
**Strongly recommended:** keep enabled

---

## Arming (“push-to-type”)

### `arm_enabled` (bool)

Enables arming gate. Injection is blocked unless armed.

**Default:** `true`

---

### `arm_duration_ms` (int)

How long the daemon remains armed after arming.

**Default:** `20000` (20s)

---

### `arm_consume_on_inject` (bool)

Consumes armed state after a successful injection.

**Default:** `true`

---

## Clipboard policy

### `allow_clipboard_when_disarmed` (bool)

Allows clipboard fallback **even when arming blocks injection**.

**Default:** `false`
**Recommended:** `false`

---

### `allow_clipboard_on_inject_failure` (bool)

Allows clipboard fallback **after gates pass but injection fails**
(e.g. Wayland, permissions).

**Default:**

* Linux: `true`
* Other platforms: `false`

---

## Arm API (local-only)

### `arm_api_enabled` (bool)

Enables a local HTTP endpoint to arm NovaKey programmatically.

**Default:** `false`

---

### `arm_listen_addr` (string)

Address for Arm API listener.

**Default:** unset (must be explicit)

---

### `arm_token_file` (string)

Path to Arm API token file.

**Default:** `arm_token.txt`

---

### `arm_token_header` (string)

HTTP header name for Arm API token.

**Default:** `X-NovaKey-Token`

---

## Injection safety

### `allow_newlines` (bool)

Allows injected secrets to include newline characters.

**Default:** `false`
**Recommended:** `false`

---

### `max_inject_len` (int)

Maximum length of injected text.

**Default:** `256`

---

## Two-Man approval

### `two_man_enabled` (bool)

Requires explicit local approval before injection.

**Default (code):** `true`
**Your shipped YAML:** `false`

---

### `approve_window_ms` (int)

Approval validity window.

**Default:** `15000`

---

### `approve_consume_on_inject` (bool)

Consumes approval after a successful injection.

**Default:** `true`

---

## Target policy

Restricts which applications/windows NovaKey may type into.

### `target_policy_enabled` (bool)

Enables target policy enforcement.

**Default:** `false`

---

### `use_built_in_allowlist` (bool)

Uses NovaKey’s built-in allowlist if no explicit lists are provided.

**Default:** auto-enabled when target policy is on and lists are empty

---

### `allowed_process_names` (list)

Allowed process names (e.g. `chrome`, `firefox`, `notepad`).

---

### `allowed_window_titles` (list)

Allowed window title substrings.

---

### `denied_process_names` (list)

Explicitly denied process names.

---

### `denied_window_titles` (list)

Explicitly denied window title substrings.

---

## Recommended baseline (most users)

```yaml
listen_addr: "127.0.0.1:60768"
require_sealed_device_store: true
arm_enabled: true
allow_clipboard_when_disarmed: false
```

If listening on LAN, strongly consider enabling **target policy allowlists**.

