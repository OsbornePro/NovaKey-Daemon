# Configuration

NovaKey-Daemon supports YAML (**preferred**) or JSON configuration files:

* `server_config.yaml`
* `server_config.yml`
* `server_config.json`

The daemon loads configuration from its **WorkingDirectory**.
Installers set this directory so that **relative paths resolve correctly**.

Relative paths such as `devices.json`, `server_keys.json`, and `./logs`
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

Address and port to bind the NovaKey TCP listener.

**Default (code):**

```
0.0.0.0:60768
```

It is highly recommedned to set your devices IP address in the `server_config.yaml` file rather than use the default `0.0.0.0`.
This will ensure if you use a VPN or virtualization your device can still receive TCP communications from your phone.  
  
Below are some commands to get your devices IP address:

```bash
# On Windows
ipconfig

# On Linux
hostname -I || ip a

# On MacOS
ifconfig | grep 'inet '
``` 

**Common values:**

* Local only (*safest because unreachable from anything but your local computer*): `127.0.0.1:60768`
* LAN access: `0.0.0.0:60768`
* Specific LAN IP: `192.168.1.50:60768`

> ⚠️ Binding to `0.0.0.0` increases attack surface.
> If listening on LAN, **target policy is strongly recommended**.

---

### `max_payload_len` (int)

Maximum size (bytes) of a single incoming message.

**Default:** `4096`

---

### `max_requests_per_min` (int)

Per-device rate limit for incoming requests.

**Default:** `60`

---

## Key & device storage

### `devices_file` (string)

Path to the device store.

Stores:

* Paired device identities
* Per-device cryptographic material

**Default:** `devices.json`

On supported platforms, the device store may be **sealed/encrypted** using OS facilities.

---

### `server_keys_file` (string)

Path to server cryptographic keys.

Includes:

* ML-KEM (Kyber) public/private keys
* Long-lived server identity

**Default:** `server_keys.json`

---

### `require_sealed_device_store` (bool)

If `true`, NovaKey **fails closed** if secure/sealed storage cannot be unlocked.

This is a **security-critical option**.

**Default (code):** `false`
**Recommended:** `true`

Linux notes:

* Hardware-backed keyrings (PAM, YubiKey) may require user interaction
* Cancelling unlock repeatedly may require manual cleanup

---

## Pairing hardening

### `rotate_kyber_keys` (bool)

Rotate server Kyber keys on **every service start**.

Effects:

* Invalidates all existing device pairings
* Forces full re-pairing

**Default:** `false`

---

### `rotate_device_psk_on_repair` (bool)

Rotate device PSKs during re-pair / repair flows.

**Default:** `false`

---

### `pair_hello_max_per_min` (int)

Per-IP rate limit for pairing “hello” messages.

* Applies only to pairing
* In-memory only (resets on restart)

**Default:** `30`

---

## Logging

> Logs may be redacted but should still be treated as sensitive.

### `log_dir` (string)

Directory for log files.

**Default behavior:** stderr only
**Common:** `./logs`

---

### `log_file` (string)

Explicit log file path.

Overrides `log_dir` if set.

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

Redacts secrets and sensitive values from logs (best effort).

**Default:** `true`
**Strongly recommended:** keep enabled

---

## Arming (“push-to-type”)

NovaKey uses a **protocol-level arming gate** (not HTTP).

The phone app sends an **Arm** message, opening a time window during which
a secret may be injected.

### `arm_duration_ms` (int)

How long NovaKey remains armed after an Arm message.

**Default:** `20000` (20 seconds)

> The phone app may override this per-arm request.

---

### `arm_consume_on_inject` (bool)

Consumes the armed state after a successful injection.

**Default:** `true`

---

## Clipboard policy

### `allow_clipboard_when_disarmed` (bool)

Allows clipboard fallback **even when injection is blocked by gates**.

**Default:** `false`
**Recommended:** `false`

---

### `allow_clipboard_on_inject_failure` (bool)

Allows clipboard fallback **after gates pass but injection fails**
(e.g. Wayland, permissions).

**Default:**

* All platforms: `false`

---

## Typing fallback

### `allow_typing_fallback` (bool)

Allows an auto-typing fallback when direct injection is not possible.

**Default:** `true`

> Note: auto-typing may be observable by keyloggers with sufficient privileges. Disable this in higher-assurance environments.

---

## macOS injection preference

### `macos_prefer_clipboard` (bool)

On macOS, prefer clipboard paste injection over AppleScript keystroke typing.

**Default:** `true`

> This default is chosen to reduce exposure to keylogger-style input capture where possible.

---

## Injection safety

### `allow_newlines` (bool)

Allow injected secrets to contain newline characters.

**Default:** `false`
**Recommended:** `false`

---

### `max_inject_len` (int)

Maximum length of injected text.

**Default:** `256`

---

## Two-Man approval

Requires explicit local approval before injection.

### `two_man_enabled` (bool)

Enable two-man approval.

**Default (code):** `true`

If disabled, injections proceed without local confirmation.

---

### `approve_window_ms` (int)

Approval validity window.

**Default:** `15000` (15 seconds)

---

### `approve_consume_on_inject` (bool)

Consumes approval after a successful injection.

**Default:** `true`

---

## Target policy (application / window allowlists)

Target policy restricts **which applications or windows NovaKey is allowed to type into**.

This is your **primary mitigation** when listening on LAN.

---

### `target_policy_enabled` (bool)

Master switch for target policy enforcement.

* `false` → no target checks at all
* `true` → enforcement enabled

**Default:** `false`

---

### Allow / deny list precedence

When target policy is enabled, NovaKey evaluates rules in this order:

1. **Denied process names**
2. **Denied window titles**
3. **Allowed process names**
4. **Allowed window titles**
5. **Built-in allowlist (optional fallback)**

If any deny rule matches → injection is blocked.
If allow rules exist → **at least one must match**.

---

### `use_built_in_allowlist` (bool)

Controls behavior **only when target policy is enabled AND no allow/deny lists are provided**.

* `true` → restrict to NovaKey’s built-in allowlist
* `false` → allow all targets

**Default:** `false`
**Auto-enabled:** when target policy is enabled and all lists are empty

---

### `allowed_process_names` (list)

Allowed process names (case-insensitive, normalized).

Examples:

```yaml
allowed_process_names:
  - chrome
  - firefox
  - notepad
```

---

### `allowed_window_titles` (list)

Allowed window title substrings.

Example:

```yaml
allowed_window_titles:
  - "Password"
  - "Login"
```

---

### `denied_process_names` (list)

Explicitly denied process names.

Always override allow rules.

---

### `denied_window_titles` (list)

Explicitly denied window title substrings.

Always override allow rules.

---

## Recommended baselines

### Local-only (default-safe)

```yaml
listen_addr: "127.0.0.1:60768"
require_sealed_device_store: true
```

---

### LAN-exposed (recommended)

```yaml
listen_addr: "0.0.0.0:60768"
require_sealed_device_store: true
target_policy_enabled: true
use_built_in_allowlist: true
```

For tighter control, replace the built-in allowlist with explicit allow/deny rules.

Perfect — below is an **expanded drop-in continuation** you can append to the configuration document (or integrate inline).
It adds **dangerous vs safe examples**, **documents the built-in allowlist**, and includes a **security-levels table** that maps cleanly to how NovaKey actually behaves.

Everything here matches your current code and defaults.

---

## Target policy examples (dangerous vs safe)

### ❌ Dangerous configurations (do not use on LAN)

#### 1) Listening on LAN with no target policy

```yaml
listen_addr: "0.0.0.0:60768"
target_policy_enabled: false
```

**Why this is dangerous**

* Any paired device can inject into *any focused window*
* Malware or a compromised phone can type into terminals, password prompts, or admin dialogs
* This is equivalent to “remote keyboard access”

---

#### 2) Target policy enabled, but empty rules and no built-in allowlist

> ⚠️ Wayland note (Linux):
> On Wayland-based desktops (*GNOME Wayland, KDE Wayland, etc.*), NovaKey cannot enforce target/window policy
> because the compositor does not expose the same process/window metadata needed for policy checks.
> For this reason, `target_policy_enabled` **must remain `false`** on Wayland Linux devices.
> If you need target policy enforcement on Linux, use an X11 session (*or XWayland where supported*) instead.


```yaml
target_policy_enabled: true
use_built_in_allowlist: false
```

> Attempting to set `target_policy_enabled: true` on Wayland may cause injections to be blocked or fall back to clipboard,
> depending on your `allow_clipboard_on_inject_failure` setting.


**Why this is dangerous**

* Target policy is technically “on”
* But with no allow/deny rules and built-in disabled, **everything is allowed**
* This gives a false sense of security

---

## ✅ Safe configurations

### 1) Safe-by-default (built-in allowlist fallback)

```yaml
target_policy_enabled: true
use_built_in_allowlist: true
```

**What this does**

* Restricts injection to NovaKey’s built-in allowlist
* No custom rules required
* Good baseline for LAN use

---

### 2) Explicit allowlist (recommended for power users)

```yaml
target_policy_enabled: true
allowed_process_names:
  - chrome
  - firefox
  - 1password
  - bitwarden
```

**What this does**

* Only these processes may receive injected text
* All others are blocked by default
* Built-in allowlist is ignored because explicit rules exist

---

### 3) Allow browser logins, deny terminals (defense-in-depth)

```yaml
target_policy_enabled: true
allowed_process_names:
  - chrome
  - firefox
denied_process_names:
  - terminal
  - powershell
  - cmd
  - bash
  - zsh
```

**What this does**

* Explicitly blocks dangerous targets even if focused
* Deny rules always win

---

### 4) Window-title-based targeting (advanced)

```yaml
target_policy_enabled: true
allowed_window_titles:
  - "Sign in"
  - "Login"
  - "Password"
denied_window_titles:
  - "Terminal"
  - "Administrator"
```

**What this does**

* Allows injection only into specific dialogs
* Useful for kiosk or SSO-style flows
* More fragile (titles change), but very restrictive

---

## Built-in allowlist (exact contents)

When `use_built_in_allowlist: true` is active **and no explicit allow/deny lists are provided**, NovaKey allows injection only into the following **process names** (case-insensitive, normalized):

### Browsers

* `chrome`
* `chromium`
* `msedge`
* `brave`
* `firefox`
* `safari`
* `opera`
* `vivaldi`

### Password managers

* `1password`
* `bitwarden`
* `lastpass`
* `dashlane`
* `keeper`
* `nordpass`
* `protonpass`
* `roboform`

### Text editors (low-privilege)

* `notepad`
* `textedit`
* `gedit`
* `kate`

> ⚠️ Not included:
>
> * Terminals (`bash`, `zsh`, `cmd`, `powershell`)
> * IDEs
> * Admin tools
> * System dialogs

This list is intentionally conservative.

---

## Security levels (recommended profiles)

| Level                  | Intended use           | Key settings             | Risk     |
| ---------------------- | ---------------------- | ------------------------ | -------- |
| **Local-only**         | Single-user machine    | `listen_addr: 127.0.0.1` | Very low |
| **LAN-safe (default)** | Home / trusted LAN     | Built-in allowlist       | Low      |
| **Explicit allowlist** | Power users            | Custom allow rules       | Very low |
| **High-assurance**     | Sensitive environments | Allow + deny + two-man   | Minimal  |
| **Dangerous**          | ❌ Not recommended      | No target policy         | High     |

---

### 🟢 Level 1: Local-only (default-safe)

```yaml
listen_addr: "127.0.0.1:60768"
require_sealed_device_store: true
```

---

### 🟡 Level 2: LAN-safe (recommended baseline)

```yaml
listen_addr: "0.0.0.0:60768"
require_sealed_device_store: true
target_policy_enabled: true
use_built_in_allowlist: true
```

---

### 🟢 Level 3: Explicit allowlist (recommended)

```yaml
listen_addr: "0.0.0.0:60768"
target_policy_enabled: true
allowed_process_names:
  - chrome
  - firefox
  - 1password
```

---

### 🔐 Level 4: High-assurance (strongest)

```yaml
listen_addr: "0.0.0.0:60768"
require_sealed_device_store: true

target_policy_enabled: true
allowed_process_names:
  - 1password
  - bitwarden
denied_process_names:
  - terminal
  - powershell
  - cmd

two_man_enabled: true
arm_consume_on_inject: true
```

**Threat model covered**

* Compromised phone
* Malware on LAN
* Accidental injection into wrong window
* Privilege escalation via terminal injection
