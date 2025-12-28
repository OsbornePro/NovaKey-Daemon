# Security Policy

**NovaKey-Daemon** is security-critical software: it receives encrypted secrets over TCP and injects them into the active window. We take security seriously and welcome review.

NovaKey currently implements:

- **/msg (Protocol v3):** ML-KEM-768 + HKDF-SHA-256 + XChaCha20-Poly1305, with timestamp freshness checks, replay protection, and per-device rate limiting.
- **/pair (Pairing v1):** one-time pairing token + ML-KEM-768 + XChaCha20-Poly1305 registration on the same TCP listener.

Optional safety controls:

- The shipped sample configuration enables arming and two-man by default (*text inject requires local arm and a recent approve window*)
- **Arming** gate (*“push-to-type”*)
- **Two-man** mode (*typed approve then inject*)
- Injection safety rules (`allow_newlines`, `max_inject_len`)
- Target policy allow/deny lists
- Arm API (*token protected*)

> Reviewer note: Please test only on systems you own/operate and do not expose your test daemon to the public Internet. This project is designed for LAN/local testing and normal desktop sessions.

---

## Supported Versions

Security updates are provided for the **latest stable release only**.

| Version        | Supported     | Notes                            |
| -------------- | ------------- | -------------------------------- |
| Latest release | Supported     | Receives security fixes promptly |
| All others     | Not supported | Upgrade recommended              |

---

## Reporting a Vulnerability

Please **do not** open public GitHub issues for security problems.

Email:
- `security@novakey.app`
- or `rosborne@osbornepro.com` if needed
- PGP key: https://downloads.osbornepro.com/publickey.asc

If you need encrypted comms, include “PGP” in your email and we’ll coordinate.

### What to include

- Steps to reproduce
- Affected version(s) and OS(es)
- Impact
- Proof-of-concept if available
- Relevant logs/config (*with secrets redacted*)

---

## Current Attack Surface

### One listening port + connection router

NovaKey listens on **one** TCP address: `listen_addr` (*default* `127.0.0.1:60768`).

Each connection is routed by an initial ASCII preface line (**required**):

- `NOVAK/1 /pair\n` → pairing handler
- `NOVAK/1 /msg\n`  → message handler

Connections that do not begin with one of these exact lines are rejected before any cryptographic processing.

### Connection routing

Routing is performed using a fixed ASCII preface line:

- `NOVAK/1 /pair\n`
- `NOVAK/1 /msg\n`

Routing occurs **before** any cryptographic processing.

Security properties are unchanged by routing:

- `/pair` is protected by a one-time token and ML-KEM
- `/msg` requires a valid per-device PSK

---

## Pairing Security (No TLS)

Pairing is initiated when there are no paired devices (*missing/empty device store*).

### Pairing token

The daemon creates a **one-time** pairing token (*128-bit*) with a TTL (*default 10 minutes*):

- token encoding: base64 **raw URL** (`base64.RawURLEncoding`)
- token is consumed by the first successful `/pair` hello

This prevents random LAN pairing attempts.

### Public key fingerprint

The daemon exposes its ML-KEM public key during pairing, and also provides a short fingerprint:

- `fp16_hex = hex(sha256(pubkey)[0:16])`

The QR should embed this fingerprint so the phone can verify the received public key matches what was scanned (*mitigates “wrong host / wrong key” and some spoofing scenarios on a LAN*).

### `/pair` cryptography

Pair registration uses:

- ML-KEM-768 decapsulation on the daemon
- XChaCha20-Poly1305 AEAD for the register payload
- AEAD key derived with HKDF-SHA-256:
  - `IKM = sharedKem`
  - `salt = tokenBytes`
  - `info = "NovaKey v4 Pair AEAD"`
  - outLen = 32 bytes

AAD binds the request to the encapsulation and nonce:

- `AAD = "PAIR" || ct || nonce`

### Pairing material is sensitive

Successful pairing results in a per-device **32-byte PSK** stored in the device store.
Treat pairing results and the device store as secrets.

**Device store at rest:**

- On **Windows**, the device store is DPAPI-protected (`*.dpapi.json`).
- On **non-Windows**, the device store is sealed with XChaCha20-Poly1305 using an OS keyring-derived key when available.
- In environments where a daemon process cannot access the user keyring (*commonly headless services or logins backed by hardware tokens*), NovaKey can be configured to allow plaintext device storage with strict permissions (`0600`). This is an explicit opt-in and should be enabled only when required.

If an attacker obtains a device PSK, they can produce valid `/msg` frames (arming/two-man can reduce silent injection risk, but does not protect a compromised host).

---

## `/msg` Security (Protocol v3)

### Per-device identity and secrets

Each device has:

- `device_id` (*string*)
- `device_key_hex` (*32 bytes, hex*)

`device_key_hex` is never sent in plaintext.

### Post-quantum key encapsulation (ML-KEM-768)

Each `/msg` request includes a KEM ciphertext:

- server decapsulates → per-message shared secret

### Session key derivation (HKDF-SHA-256)

Per-message AEAD key:

- `IKM = sharedKem`
- `salt = deviceKey`
- `info = "NovaKey v3 AEAD key"`
- outLen = 32 bytes

### Authenticated encryption (XChaCha20-Poly1305)

- Nonce: 24 bytes (*random per message*)
- AAD: binds the entire header through the KEM ciphertext
- Prevents tampering with device routing / KEM material

### Typed inner message framing (required)

After decrypting, the plaintext includes a timestamp and then an **inner typed frame**:

- `inner msgType = 1` → Inject (payload is secret string)
- `inner msgType = 2` → Approve (payload empty/ignored)
- `inner msgType = 3` → Arm (payload JSON with duration)
- `inner msgType = 4` → Disarm (payload empty)

Only typed frames are accepted; there is no legacy or magic-string control path.

### Freshness & replay protection

- plaintext includes Unix timestamp (*seconds*)
- server rejects stale messages and large clock skew
- server caches `(deviceID, nonce)` for a TTL window to detect replays

### Per-device rate limiting

- server enforces accepted message limits per device (`max_requests_per_min`)

---

## Optional Safety Controls

### Arming gate

When `arm_enabled: true`, frames can decrypt/validate but injection is blocked unless locally armed.

### Two-man mode

When `two_man_enabled: true`, injection requires a recent approve (`inner msgType=2`) from the same device (*per-device approval window*).

### Local Arm API (loopback only)

If `arm_api_enabled: true`:

- binds only to loopback (`arm_listen_addr` must resolve to loopback)
- token-protected via a random token stored in `arm_token_file`, supplied in header `arm_token_header`

Note: processes running as the same user may be able to read the token file; host compromise is considered game-over.

### Injection safety policies

Even after crypto succeeds:

- newline blocking by default (`allow_newlines: false`)
- max injected length (`max_inject_len`)
- optional target allow/deny policy for focused apps (*process/window*)

**Target policy normalization note:**

Process comparisons are normalized to reduce configuration foot-guns:

- lowercased
- path stripped (`/usr/bin/firefox`, `C:\...\chrome.exe`)
- `.exe` stripped
- `.app` stripped (*macOS*)

---

## Logging & Redaction

NovaKey can write logs to stderr and/or a rotating file.

When `log_redact: true`:

- secrets registered via `addSecret()` are replaced with `[REDACTED]`
- long base64/hex-ish blobs are replaced with `[REDACTED_BLOB]`
- common key/value patterns are redacted, including URL query params (*e.g.* `token=...&fp=...`)

Even with redaction enabled, logs should still be treated as potentially sensitive and protected accordingly.

---

## Threat Model

### In scope

- passive sniffing / active tampering on LAN
- replay attempts
- unauthorized clients without device PSK
- rate abuse from a valid device

### Out of scope

- fully compromised host OS / same-user malware
- physical attacks / hardware keyloggers
- compromised build pipeline
- QR exposure == pairing exposure during TTL

---

Thank you for helping keep NovaKey secure.

— Robert H. Osborne (OsbornePro)  
Maintainer, NovaKey-Daemon
````

---
