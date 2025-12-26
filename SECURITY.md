# Security Policy

**NovaKey-Daemon** is security-critical software: it receives encrypted secrets over TCP and injects them into the active window. We take security seriously and welcome review.

NovaKey-Daemon uses:

- **ML-KEM-768 + HKDF-SHA-256 + XChaCha20-Poly1305** for transport security on `/msg`
- timestamp freshness checks, replay protection, and per-device rate limiting
- optional **arming** and **two-man approval** gates to reduce risk from compromised pairing material

> Reviewer note: Please test only on systems you own/operate and do not expose your test daemon to the public Internet. NovaKey is designed for LAN/local testing and normal desktop sessions.

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
* `security@novakey.app`
* or `rosborne@osbornepro.com` if needed
* PGP key: https://downloads.osbornepro.com/publickey.asc

If you need encrypted comms, include “PGP” in your email and we’ll coordinate.

### What to include

* Steps to reproduce
* Affected version(s) and OS(es)
* Impact
* Proof-of-concept if available
* Relevant logs/config (with secrets redacted)

---

## Current Design & Security Properties

### Network surface

NovaKey listens on **one TCP address** configured by `listen_addr` (default `127.0.0.1:60768`).

Connections are routed by a short ASCII preface line:

- `NOVAK/1 /pair` — pairing exchange (one-time token)
- `NOVAK/1 /msg`  — encrypted message exchange (ML-KEM + AEAD)

For backward compatibility, if a client does **not** send the `NOVAK/1` route line, the daemon treats the connection as `/msg`.

### Pairing security (no TLS)

Pairing is initiated by scanning a QR code displayed by the daemon when no devices are paired.

The QR contains:

- the daemon host+port
- a **short-lived pairing token**
- a short fingerprint of the daemon’s ML-KEM public key (for client-side sanity checking)
- an expiration timestamp

Pairing occurs over the same listening port via `/pair`. The token is required and expires; pairing is intended for local/LAN use only.

Security note: pairing returns a device secret. Treat pairing output as sensitive (like a password). If an attacker obtains a device secret, they may be able to produce valid encrypted `/msg` frames.

### Per-device identity and secrets

- Each paired device has a unique device ID and a **32-byte secret** stored in `devices.json` (or platform-specific secure storage if configured).
- Device secrets are never sent in plaintext during normal operation.

### Post-quantum key encapsulation (ML-KEM-768)

Each `/msg` request includes a KEM ciphertext.

- Server decapsulates to obtain a per-message shared secret.
- This prevents passive sniffers from learning the derived AEAD session key.

### Session key derivation (HKDF-SHA-256)

Each message derives an ephemeral session key using:

- IKM = per-message KEM shared secret
- salt = per-device PSK (32 bytes)
- info = `"NovaKey v3 AEAD key"`

Session keys are single-use and not stored.

### Authenticated encryption (XChaCha20-Poly1305)

Secrets are encrypted and authenticated with AEAD.

- The server binds header fields (including device ID and KEM ciphertext) as AAD to prevent tampering.

### Typed message framing (approve vs inject)

The daemon expects an encrypted outer message that decrypts to a typed inner frame:

- inner msgType `1` = Inject (payload is the secret string)
- inner msgType `2` = Approve (payload empty/ignored)

This avoids “magic string” controls and keeps policy decisions explicit.

### Freshness & replay protection

- Each plaintext includes a Unix timestamp.
- Each message includes a random XChaCha nonce.
- Server rejects stale messages and replays of `(deviceID, nonce)` within a TTL window.

### Per-device rate limiting

- NovaKey enforces a per-device accepted message limit (`max_requests_per_min`).

### Arming gate (optional)

When enabled (`arm_enabled: true`), messages may decrypt and validate successfully, but **injection is blocked unless the host is armed**.

### Two-man mode (optional)

When enabled (`two_man_enabled: true`), injection requires:

1) host is armed (if `arm_enabled` is on), and  
2) a recent per-device **typed approve** message.

### Local Arm API (loopback only)

If enabled (`arm_api_enabled: true`):

- Binds only to loopback (recommended `127.0.0.1:60769`)
- NovaKey refuses non-loopback binds
- Protected by a random token stored in `arm_token_file`, provided via header `arm_token_header`

Security note: any process running as the same user may potentially read the token file. Host compromise is considered game-over (standard assumption).

### Injection safety policies

NovaKey applies safety checks even after crypto succeeds:

- `allow_newlines: false` blocks `\n` and `\r` by default
- `max_inject_len` caps injected text length
- optional target policy allow/deny lists restrict which focused apps are allowed

### Logging safety

NovaKey logs do not include full secrets.

- Secrets are preview-only when logged (short prefix + length).
- Optional file logging with rotation and redaction via config.

---

## Threat Model (High Level)

### In scope

- Passive sniffing / active tampering on local networks
- Replay attempts
- Malicious clients without valid device secrets
- Rate abuse from a valid device

### Out of scope (assumed)

- Fully compromised host OS / kernel / hypervisor
- Malware running as the same user
- Physical attacks and hardware keyloggers
- Compromise of distribution/build pipeline (repo-level mitigations only)

### Pairing material compromise

If an attacker obtains a device secret (pairing output or `devices.json`):

- They can generate valid encrypted `/msg` frames.
- Arming/two-man can reduce the chance of silent injection when configured.
- This does not protect against host compromise.

---

Thank you for helping keep NovaKey secure.

— Robert H. Osborne (OsbornePro)  
Maintainer, NovaKey-Daemon
