# Security Policy

**NovaKey-Daemon** is security-critical software: it receives encrypted secrets over TCP and injects them into the active window. We take security seriously and welcome review.

NovaKey is designed to minimize exposure of high-value secrets on desktop systems while preserving usability.

---

## Implemented Cryptography

NovaKey currently implements:

- **/msg (Protocol v3):**
  - ML-KEM-768 (Kyber) for per-message post-quantum key establishment
  - HKDF-SHA-256 for key derivation
  - XChaCha20-Poly1305 for authenticated encryption
  - Timestamp freshness checks
  - Replay protection
  - Per-device rate limiting

- **/pair (Pairing v1):**
  - One-time pairing token
  - ML-KEM-768
  - HKDF-SHA-256
  - XChaCha20-Poly1305
  - Single-port pairing on the same TCP listener

---

## Injection Safety Model

Successful cryptographic validation does **not** imply injection will occur.

NovaKey enforces multiple independent safety gates **after decryption but before injection**, including:

- Arming gate (“push-to-type”)
- Optional two-man approval window
- Injection safety rules (`allow_newlines`, `max_inject_len`)
- Focused target policy allow/deny lists
- Optional clipboard fallback

Injection occurs only if **all enabled gates pass**.

---

## Injection Outcomes & Clipboard Exposure

NovaKey always attempts **direct injection into the focused control first**.

Depending on OS support, permissions, and configuration, one of the following outcomes may occur:

- **Direct injection**
  - Secret is inserted without clipboard or auto-typing.
- **Auto-typing fallback**
  - Secret is typed programmatically using OS input APIs.
- **Clipboard paste injection**
  - Clipboard is set and a programmatic paste action is performed.
- **Clipboard-only fallback**
  - Clipboard is set; the user must paste manually.

### Clipboard policy

Clipboard usage is intentionally restricted.

Clipboard may be used **only** when one of the following conditions is true:

1. Injection is blocked by a gate or target policy **and**
   `allow_clipboard_when_disarmed = true`, or
2. Injection fails after all gates pass **and**
   `allow_clipboard_on_inject_failure = true`

Clipboard is:

- never preloaded
- never set opportunistically
- never used by default
- never touched if not explicitly enabled by configuration

---

## Typing Injection

Auto-typing fallback uses OS-level synthetic input APIs.

Security considerations:

- Auto-typing may be observable by keyloggers with sufficient privileges.
- Typing fallback is optional and can be disabled (`allow_typing_fallback=false`).
- On macOS, clipboard paste injection is preferred over AppleScript keystroke typing by default.

Users should evaluate typing fallback risk based on their threat model.

---

## Arming Gate (“Push-to-Type”)

When arming is enabled:

- Messages decrypt and authenticate normally
- Injection is blocked unless the daemon is locally armed
- Arming automatically expires after `arm_duration_ms`
- Successful injection may consume the armed state

This prevents unattended or background injection.

---

## Two-Man Approval Mode

When `two_man_enabled = true`:

- An **Approve** message is required before injection
- Approval is per-device
- Approval expires after `approve_window_ms`
- Approval may be consumed on injection

Two-man approval reduces the risk of silent injection from a compromised client.

---

## Target Policy Enforcement

NovaKey can restrict injection based on the currently focused application or window.

Target policy rules may include:

- Allowed process names
- Allowed window title substrings
- Denied process names
- Denied window title substrings

Target evaluation occurs **after decryption** but **before injection**.

### Wayland note

On Linux Wayland sessions, focused application detection is limited.

When injection is unavailable due to Wayland constraints:

- Injection fails deterministically
- Clipboard fallback may be used **only if explicitly enabled**
- The daemon reports a distinct result (`inject_unavailable_wayland`)

---

## Device Store & Key Protection

Successful pairing results in a per-device static secret (PSK).

### Windows

- Device store is sealed using **DPAPI**
- Stored as `*.dpapi.json`
- Tied to the local user context

### macOS / Linux

Device store is stored in one of two forms:

1. **Sealed wrapper (preferred)**
   - Encrypted at rest using XChaCha20-Poly1305
   - Sealing key derived from the OS keyring
2. **Plaintext JSON (explicit opt-in)**
   - Used only when the OS keyring is unavailable
   - Requires explicit configuration
   - Stored with strict `0600` permissions

On some Linux systems (e.g. headless services or hardware-token-backed logins), the daemon may be unable to access the user keyring. In those cases, plaintext storage may be required.

Control this behavior with:

- `require_sealed_device_store = true`
  - Fail closed if keyring unavailable (recommended)
- `require_sealed_device_store = false`
  - Allow plaintext fallback when necessary

---

## Pairing Security

Pairing occurs only when no devices are paired.

### Pairing token

- One-time token (128-bit)
- Base64 raw URL encoding
- Server-side TTL
- Consumed on first successful hello

This prevents opportunistic LAN pairing.

### Public key fingerprint

During pairing, the daemon provides:

- ML-KEM public key
- Short fingerprint:

```

fp16_hex = hex( sha256(pubkey)[0:16] )

```

Clients must verify the fingerprint matches the QR code.

---

## Pairing Cryptography Details

Pair registration uses:

- ML-KEM-768 decapsulation
- XChaCha20-Poly1305 AEAD
- HKDF-SHA-256 key derivation

HKDF parameters:

- `IKM = sharedKem`
- `salt = tokenBytes`
- `info = "NovaKey v3 Pair AEAD"`
- `outLen = 32 bytes`

AAD binds the encapsulation and nonce:

```

AAD = "PAIR" || ct || nonce

```

---

## `/msg` Protocol Security (v3)

Each `/msg` request includes:

- Per-message KEM ciphertext
- Per-message AEAD nonce
- Timestamp freshness enforcement
- Replay protection via nonce caching
- Per-device rate limiting

### Key derivation

```

K = HKDF-SHA256(
IKM  = kemShared,
salt = deviceKey,
info = "NovaKey v3 AEAD key",
outLen = 32
)

```

---

## Logging & Redaction

NovaKey supports structured logging with optional redaction.

When `log_redact = true`:

- Secrets are replaced with `[REDACTED]`
- Long base64/hex blobs are replaced with `[REDACTED_BLOB]`
- Common key/value tokens are masked

Logs should still be treated as sensitive.

---

## Threat Model

### In Scope

- Passive or active LAN attackers
- Replay attempts
- Unauthorized devices
- Message tampering
- Rate abuse by a paired device

### Out of Scope

- Fully compromised host OS
- Same-user malware
- Hardware keyloggers
- Compromised build pipeline
- Pairing QR exposure during TTL

A compromised host is considered game-over.

---

## Supported Versions

Security updates are provided for the **latest stable release only**.

| Version        | Supported |
| -------------- | --------- |
| Latest release | Yes       |
| Older versions| No        |

---

## Reporting a Vulnerability

Please **do not** open public issues for security problems.

Contact:

- `security@novakey.app`
- `rosborne@osbornepro.com`

PGP key:
https://downloads.osbornepro.com/publickey.asc

If encrypted communication is required, include “PGP” in your email.

---

Thank you for helping keep NovaKey secure.

— Robert H. Osborne  
Maintainer, NovaKey-Daemon
