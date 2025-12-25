# Security Policy

**NovaKey-Daemon** is security-critical software: it receives encrypted secrets over TCP and injects them into the active window. We take security seriously and welcome review.

NovaKey-Daemon v3 uses **ML-KEM-768 + HKDF-SHA-256 + XChaCha20-Poly1305**, plus timestamp freshness checks, replay protection, and per-device rate limiting. NovaKey also supports optional **arming** and **two-man** controls to reduce risk from compromised device pairing material.

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
* `security@novakey.app`
* or `rosborne@osbornepro.com` if needed
* My PGP key can be obtained from [HERE](https://downloads.osbornepro.com/publickey.asc)

If you need encrypted comms, include “PGP” in your email and we’ll coordinate.

### What to include

* Steps to reproduce
* Affected version(s) and OS(es)
* Impact
* Proof-of-concept if available
* Relevant logs/config (with secrets redacted)

### What to expect

* Acknowledgment within 24 hours
* Triage response within ~3 business days
* Fix shipped as soon as practical (faster for critical issues)

---

## Security Features

### Per-device identity and secrets

* Each device has a unique device ID and a **32-byte secret** stored in `devices.json`.
* Device secrets are not transmitted in plaintext.
* Pairing output (JSON/QR) contains the device secret and must be protected like a password.

### Post-quantum key encapsulation (ML-KEM-768)

* Each request includes a KEM ciphertext.
* Server decapsulates it to obtain a per-message shared secret.
* This prevents passive sniffers from learning the AEAD session key.

### Session key derivation (HKDF-SHA-256)

Each message derives an ephemeral session key using:

* IKM = per-message KEM shared secret
* salt = per-device PSK
* info = `"NovaKey v3 AEAD key"`

Session keys are single-use and not stored.

### Authenticated encryption (XChaCha20-Poly1305)

* Secrets are encrypted and authenticated with AEAD.
* The server binds the header (including device ID and KEM ciphertext) as AAD to prevent tampering.

### Typed message framing (approve vs inject)

In current v3 usage, the daemon expects an **outer v3 frame** with a fixed outer type, and the decrypted plaintext carries a **typed inner message frame**:

* inner msgType `1` = Inject (payload is the secret string)
* inner msgType `2` = Approve (payload is empty/ignored)

This avoids “magic string” controls and keeps policy decisions explicit.

> There is no “legacy approve magic” mode in current v3 framing.

### Freshness & replay protection

* Each plaintext includes a Unix timestamp.
* Each message includes a random XChaCha nonce.
* Server rejects stale messages and replays of `(deviceID, nonce)` within a TTL window.

### Per-device rate limiting

* NovaKey enforces a per-device accepted message limit (`max_requests_per_min`).
* This mitigates abuse by a compromised client and prevents accidental spamming.

### Arming gate (optional)

When enabled (`arm_enabled: true`):

* Frames can decrypt and validate successfully,
* but injection is blocked unless the host is currently armed.

This reduces the impact of leaked pairing material by requiring a local “push-to-type” gate.

### Two-man mode (optional)

When enabled (`two_man_enabled: true`), injection requires:

1) local arming, **and**
2) a recent per-device **typed approve** message (inner msgType=2).

### Local Arm API (loopback only)

If enabled (`arm_api_enabled: true`):

* Binds only to loopback (recommended: `127.0.0.1:60769`)
* NovaKey refuses non-loopback binds
* Protected by a random token stored in `arm_token_file`, provided in header `arm_token_header`

Security note: any process running as the same user may potentially read the token file. Host compromise is considered game-over (standard assumption).

### Injection safety policies

NovaKey applies safety checks even after crypto succeeds:

* `allow_newlines: false` blocks `\n` and `\r` by default
* `max_inject_len` caps injected text length
* Target policy allow/deny lists can restrict which focused apps are allowed

### Logging safety

NovaKey logs **never** include full secrets.

* Password logs are preview-only (e.g. `"Sup..." (len=23)`).
* Logging can be configured to write to file with rotation:
  * `log_dir` or `log_file`
  * `log_rotate_mb`, `log_keep`
* When `log_redact: true` (default), NovaKey redacts:
  * arm tokens (if available)
  * long blob-like strings
  * obvious `token=`, `password=`, etc patterns

---

## Threat Model (High Level)

### In scope

* Passive sniffing / active tampering on local networks
* Replay attempts
* Malicious clients without valid device secrets
* Rate abuse from a valid device

### Out of scope (assumed)

* Fully compromised host OS / kernel / hypervisor
* Malware running as the same user
* Physical attacks and hardware keyloggers
* Compromise of distribution/build pipeline (repo-level mitigations only)

### Pairing material compromise

If an attacker obtains a device secret (pairing JSON/QR or `devices.json`):

* They can generate valid frames.
* Arming/two-man can prevent silent injection when configured.
* This does not protect against host compromise.

---

Thank you for helping keep NovaKey secure.

— Robert H. Osborne (OsbornePro)  
Maintainer, NovaKey-Daemon
