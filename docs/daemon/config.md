# Configuration

NovaKey-Daemon supports YAML (**preferred**) or JSON configuration:

- `server_config.yaml`
- `server_config.json`

NovaKey loads configuration from the **current working directory** (the daemon’s WorkingDirectory).  
Installers set the WorkingDirectory so that relative paths like `devices.json`, `server_keys.json`, and `./logs` work as expected.

---

## Config file selection order

If multiple config files exist in the WorkingDirectory, NovaKey uses:

1. `server_config.yaml`
2. `server_config.yml`
3. `server_config.json`

---

## Core networking & limits

> Best practice: bind to a specific interface/IP when possible.  
> Binding to `0.0.0.0` listens on all interfaces and increases attack surface.

- `listen_addr` (string)  
  Address/port to listen on.  
  **Default:** `127.0.0.1:60768` (if not set)  
  **Common values:**
  - Local-only (safest): `127.0.0.1:60768`
  - LAN use: `0.0.0.0:60768` or a specific LAN IP like `10.0.0.10:60768`

- `max_payload_len` (int)  
  Maximum request payload size in bytes.  
  **Default:** `4096`

- `max_requests_per_min` (int)  
  Rate limit for incoming requests.  
  **Default:** `60`

---

## Key and device storage

- `devices_file` (string)  
  Device pairing store path. Stores paired devices and their identity material.  
  May be sealed/encrypted depending on platform and configuration.  
  **Default:** `devices.json`

- `server_keys_file` (string)  
  Server key material path. Stores the daemon’s long-lived cryptographic keys.  
  **Default:** `server_keys.json`

- `require_sealed_device_store` (bool)  
  If `true`, NovaKey fails closed when platform secure storage (keyring / sealed storage) is unavailable.  
  This is the **recommended** security posture.  
  **Default:** `false` (if not set in config)  
  **Recommended:** `true`

---

## Pairing hardening

- `rotate_kyber_keys` (bool)  
  If `true`, server Kyber keys are rotated every time the service is restarted. 
  This setting is here for anyone who desires to rotate device keys and force a new pair with their phone app every time the service is restarted. 
  **Default:** `false`

- `rotate_device_psk_on_repair` (bool)  
  If `true`, device PSKs are rotated during re-pair/repair operations.  
  **Default:** `false`

- `pair_hello_max_per_min` (int)  
  Per-IP rate limit for `/pair` handshake (“hello”) requests (in-memory limiter).  
  **Default:** `30`

---

## Logging

> Logs may be redacted, but should still be treated as sensitive.

- `log_dir` (string)  
  Directory for log files. Can be relative (resolved under WorkingDirectory).  
  **Default behavior:** If unset, logging still works (stderr logging is enabled by default).  
  **Common:** `./logs`

- `log_file` (string)  
  Optional explicit log file path. If set, it overrides `log_dir`.  
  **Default:** unset

- `log_rotate_mb` (int)  
  Log rotation size in MB.  
  **Default:** `10`

- `log_keep` (int)  
  Number of rotated logs to retain.  
  **Default:** `10`

- `log_stderr` (bool)  
  If `true`, logs are emitted to stderr.  
  **Default:** `true`

- `log_redact` (bool)  
  If `true`, secrets are redacted from logs as best effort.  
  **Default:** `true`  
  **Recommended:** keep `true`

---

## Safety gates

### Arming (“push-to-type”)

- `arm_enabled` (bool)  
  Enables the arming gate (local “arm” window required before injection).  
  **Default:** `true`

- `arm_duration_ms` (int)  
  How long the daemon stays armed after arming.  
  **Default:** `20000`

- `arm_consume_on_inject` (bool)  
  If `true`, successful injection consumes the armed state immediately.  
  **Default:** `true`

---

## Clipboard policy

Clipboard can be used as a fallback when injection is blocked or fails.

- `allow_clipboard_when_disarmed` (bool)  
  If `true`, clipboard may be used even when arming/gates block typing.  
  **Default:** `false`  
  **Recommended:** `false`

- `allow_clipboard_on_inject_failure` (bool)  
  If `true`, clipboard may be used if injection fails after gates pass (Wayland, permissions, etc.).  
  **Default:** `true` on Linux, `false` on other platforms

---

## Arm API (local arming endpoint)

This can expose a local endpoint to arm NovaKey programmatically (intended to be local-only).

- `arm_api_enabled` (bool)  
  Enables the arm API endpoint.  
  **Default:** `false` if not set (recommended to explicitly set)

- `arm_listen_addr` (string)  
  Address/port for arm API.  
  **Default:** (no default applied here; set explicitly if using)

- `arm_token_file` (string)  
  Token file path used by the arm API.  
  **Default:** `arm_token.txt`

- `arm_token_header` (string)  
  HTTP header name expected for the token.  
  **Default:** `X-NovaKey-Token`

---

## Injection safety

- `allow_newlines` (bool)  
  If `true`, injected secrets may include newlines.  
  **Default:** `false`  
  **Recommended:** `false`

- `max_inject_len` (int)  
  Maximum length of injected text.  
  **Default:** `256`

---

## Two-man approval

- `two_man_enabled` (bool)  
  Requires an explicit local approval action before injection is allowed.  
  **Default in code:** `true` (if not set)  
  **Note:** Your shipped YAML currently sets this to `false`.

- `approve_window_ms` (int)  
  How long the approval window stays valid.  
  **Default:** `15000`

- `approve_consume_on_inject` (bool)  
  If `true`, approval is consumed after a successful injection.  
  **Default:** `true`

---

## Target policy

Target policy restricts which applications/windows NovaKey is allowed to type into.

- `target_policy_enabled` (bool)  
  Enables target policy enforcement.  
  **Default:** `false`

- `use_built_in_allowlist` (bool)  
  If `true`, uses NovaKey’s built-in allowlist when no explicit lists are configured.  
  **Default behavior:** if target policy is enabled and you provide no allow/deny lists, NovaKey may enable this automatically.

- `allowed_process_names` (list of strings)  
  Allowed process names (example: `chrome`, `firefox`, `notepad`).  
  If non-empty, acts as an allowlist.

- `allowed_window_titles` (list of strings)  
  Allowed window title substrings/patterns (implementation dependent).

- `denied_process_names` (list of strings)  
  Denied process names.

- `denied_window_titles` (list of strings)  
  Denied window title substrings/patterns.

---

## Recommended defaults (most users)

- `listen_addr: "127.0.0.1:60768"` unless you truly need LAN access
- `require_sealed_device_store: true`
- `arm_enabled: true`
- `allow_clipboard_when_disarmed: false`

If you enable LAN listening, strongly consider enabling target policy allowlists.

