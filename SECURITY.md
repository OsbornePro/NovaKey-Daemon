# Security Policy

**NovaKey-Daemon** is a security-critical application that receives encrypted secrets over TCP and injects them into the active window. We take security extremely seriously and actively invite review.

NovaKey-Daemon v3 uses **ML-KEM-768 + HKDF + XChaCha20-Poly1305** with per-device secrets, timestamp freshness, replay protection, and rate limiting. In addition, NovaKey supports an optional **armed injection gate** (push-to-type) designed to reduce risk if device secrets are compromised.

---

## Supported Versions

Security updates are provided for the **latest stable release only**.

| Version        | Supported     | Notes                            |
| -------------- | ------------- | -------------------------------- |
| Latest release | Supported     | Receives security fixes promptly |
| All others     | Not supported | Upgrade recommended              |

We follow a **rolling release** model: as soon as a new version is tagged, it becomes the only supported version.

---

## Reporting a Vulnerability

We strongly encourage responsible disclosure of security vulnerabilities.

### How to report

Please **do not** open public GitHub issues for security problems.

Instead, send reports to:
**[security@novakey.app](mailto:security@novakey.app)**
(*or directly to **[robert@osbornepro.com](mailto:robert@osbornepro.com)** if email is unavailable*)

You can use this PGP key for encrypted reports:
**[PGP key](https://downloads.osbornepro.com/publickey.asc)** for `rosborne@osbornepro.com` (*recommended for sensitive reports*).

### What to include

* Detailed steps to reproduce
* Affected version(s) and OS(es)
* Potential impact (e.g., credential leakage, privilege escalation, replay abuse, injection abuse)
* Proof-of-concept if available
* Any relevant logs, config snippets, or screenshots (with secrets redacted)

### What to expect

* You will receive an acknowledgment **within 24 hours** (usually much faster).
* We will validate the report and respond with next steps **within 3 business days**.
* If the report is valid, a fix will be shipped **as soon as possible** — typically within 7 days for critical issues when practical.
* You will be credited in release notes unless you prefer to remain anonymous.

### Bug bounty

We do not currently offer monetary rewards, but we are deeply grateful for high-quality reports and will:

* Publicly credit you (with your permission)
* Prioritize your future reports
* Send NovaKey-Daemon stickers, and gift the companion app when made

### Non-qualifying issues

The following are **out of scope** (but still appreciated if reported):

* Physical access to an unlocked computer
* Issues requiring root/admin on an already compromised machine
* Social engineering, phishing, or user-training issues
* Denial-of-service via local OS resource limits (e.g., exhausting CPU by running thousands of local clients)
* Attacks that assume full compromise of the host OS, kernel, or hypervisor

---

## Security Features

NovaKey-Daemon is designed with defense-in-depth. The current v3 implementation includes:

> **Optional** two-man rule (*local arm + device approval*)

When enabled, injection requires:
* a local arm action, and
* a recent per-device approval message.

This reduces risk from:
* accidental injections into the wrong window while armed,
* other paired devices injecting without the intended device’s approval.

Limitations to state clearly:
* If a device secret is compromised, the attacker can still send approval messages but still cannot inject unless the host is locally armed.
* Does not protect against host compromise / local malware (*same as everything else*).


### Per-device identity and secrets

* Each device has a unique **device ID** and 32-byte secret stored in `devices.json`.
* Only devices that possess the correct secret can derive valid AEAD session keys for accepted frames.
* Device secrets are not transmitted in plaintext and should be kept local to the NovaKey-Daemon host and trusted clients.

### Post-quantum key encapsulation (ML-KEM-768)

* NovaKey-Daemon v3 uses **ML-KEM-768** (Kyber-class) via `filippo.io/mlkem768`.
* The server maintains a long-term KEM keypair in `server_keys.json`:

  * `kyber768_public` is distributed to clients during pairing (e.g., via QR code).
  * `kyber768_secret` remains on the server and is used only to decapsulate client KEM ciphertexts.
* For each frame, the client:

  * Encapsulates to the server’s public key,
  * Obtains a shared KEM secret, and
  * Sends the KEM ciphertext along with the encrypted payload.

### Session key derivation (HKDF)

* Each request derives an **ephemeral session key** using HKDF-SHA256 over:

  * The post-quantum KEM shared secret, and
  * The per-device secret key, plus contextual info.
* Session keys are:

  * Used only for one frame, and
  * Not written to disk.

This means that:

* Even if one layer is compromised (e.g., device secret leakage or server KEM key leakage), the protocol still requires the other component to forge frames or decrypt traffic.

### Authenticated encryption with XChaCha20-Poly1305

* Payloads are encrypted and authenticated using **XChaCha20-Poly1305**.
* The header (version, message type, device ID length, device ID) is bound into the ciphertext via AEAD additional data (AAD).
* This prevents header tampering and ensures:

  * Confidentiality of the password and timestamp.
  * Integrity and authenticity of each frame.

### Freshness & replay protection

Each plaintext contains:

* A **Unix timestamp** (seconds).
* A per-message **XChaCha nonce** used for AEAD.

The server enforces:

* A configurable **freshness window** (`maxMsgAgeSec` and `maxClockSkewSec`).
* An in-memory **replay cache** per device keyed by `(deviceID, nonce)` for a fixed TTL.

> **NOTE:** Device IDs are sent in plaintext for routing/logging; don’t use sensitive identifiers.  

Result:

* Frames outside the acceptable time window are rejected as stale or too far in the future.
* Reuse of the same `(deviceID, nonce)` within the TTL is rejected as a replay.

### Per-device rate limiting

* NovaKey-Daemon tracks basic rate state per device (`windowStart`, `count`).
* Each device is limited to a configurable number of accepted frames per minute (`max_requests_per_min` in `server_config.yaml` or `server_config.json`).
* This protects against:

  * Misbehaving apps spamming injections.
  * Simple abuse by compromised clients.

### Strict framing & length checks

* Every request is framed as `[u16 length][payload bytes]`.
* The server enforces a configurable `max_payload_len` before attempting decryption.
* This mitigates some trivial memory abuse and malformed frame attacks.

### Armed injection gate (optional, recommended for testing)

NovaKey-Daemon supports an optional **arming gate** (“push-to-type”).

* When enabled (`arm_enabled: true`), the daemon may successfully decrypt and validate frames, but it will **block injection** unless it is currently **armed**.
* This reduces the risk of “silent remote injection” if a device secret is compromised:

  * A compromised device secret can still produce valid frames,
  * But it cannot cause keystroke injection unless the workstation is locally armed.
* Depending on configuration, NovaKey may still set clipboard contents while disarmed:

  * `allow_clipboard_when_disarmed: true` (convenience / Wayland compatibility)
  * `allow_clipboard_when_disarmed: false` (strict, no side effects)

> Note: The arm gate is a server-side policy and is **not part of the on-the-wire protocol**. Clients do not change the v3 frame format to use it.

### Local Arm API (loopback only)

To support hotkeys, local tooling, and security testing, NovaKey can expose a local-only HTTP API used to arm the service.

* Enabled via `arm_api_enabled: true`.
* Binds to `arm_listen_addr` (default/recommended `127.0.0.1:60769`).
* The daemon **refuses to bind non-loopback addresses** for this API.
* Protected by a random token stored in `arm_token.txt` (path configurable via `arm_token_file`):

  * Auto-generated if missing when the Arm API is enabled.
  * Sent by clients in a header (`arm_token_header`, default `X-NovaKey-Token`).

Endpoints:

* `POST /arm?ms=20000` — arms NovaKey for a duration (bounded by the server).
* `GET /status` — shows current `armed_until` timestamp.

Security notes:

* The Arm API is a local control surface. Any process running as the same user may be able to read the token file, which is consistent with the broader assumption that **host compromise is game-over**.
* The Arm API is intentionally loopback-only to avoid remote arming.

### Injection safety policies (defense-in-depth)

Because typing/clipboard injection is high-impact, NovaKey applies additional server-side safety checks.

Common controls include:

* `allow_newlines: false` by default:

  * Blocks `\n` and `\r` to reduce “paste + execute” style attacks in terminals.
* `max_inject_len`:

  * Caps how much text can be injected, independent of encrypted payload size.

These checks apply even if the protocol crypto and validation succeed. If blocked, logs will show clear “blocked injection” reasons.

### Local-only or LAN-only; no cloud

* NovaKey-Daemon listens on a locally configured TCP address, e.g.:

  * `127.0.0.1:60768` for local-only, or
  * `0.0.0.0:60768` for LAN usage.
* There is **no external relay or cloud backend**.
* All cryptographic material and injection logic live on the user’s own machine.

### No special privileges at runtime

* NovaKey-Daemon is intended to run as a normal user-level process or unprivileged service.
* Elevated privileges may be required for:

  * Installation,
  * Service wiring,
  * OS-specific auto-start configuration,
    but not for normal operation.

### Truncated logging of sensitive data

* Passwords are never fully logged.
* Logs use a safe preview (e.g., `"Sup..." (len=23)`) and never include full secrets or key material.
* Internal errors avoid printing raw plaintext or keys.

---

## Threat Model (High Level)

### In scope

* Attackers on the **local network** attempting to:

  * Read NovaKey-Daemon traffic (passive sniffing).
  * Modify or inject NovaKey-Daemon traffic (active MITM).
  * Replay previously captured packets.
* Malicious or compromised client apps that know the IP/port but **do not** have valid per-device secrets.
* Attempts to abuse the service via:

  * Excessive requests from an otherwise valid device,
  * Malformed frames aimed at protocol parsing.

### Out of scope / assumed trust

* Fully compromised host operating system, kernel, or hypervisor.
* Malicious software with the same or greater privileges as NovaKey-Daemon (e.g., another process with:

  * Keyboard injection rights,
  * Accessibility / input monitoring permissions).
* Physical attacks on the machine (hardware keyloggers, cold boot, RAM scraping).
* Compromise of the user’s phone or seed secrets used by the mobile app (beyond what per-device rate limiting can mitigate).
* Attacks on build infrastructure or distribution channels outside the scope of this repository.

### Pairing and QR Codes

* Device pairing uses a JSON blob (often via QR code) that includes:

  * Device ID,
  * Device secret (`key_hex`),
  * Server address,
  * Server ML-KEM-768 public key.
* Anyone who obtains that pairing blob can impersonate that device on the network.
* Users should:

  * Treat pairing QR codes as secrets,
  * Avoid screenshots and sharing,
  * Regenerate/revoke device entries via `nvpair` if compromise is suspected.

### If a device secret is compromised

If an attacker obtains a valid device secret (`devices.json` entry or leaked pairing material):

* They can generate valid v3 frames and may pass crypto + replay + rate-limit checks.
* If `arm_enabled` (and/or Arm API gating) is enabled on the host, injection is additionally protected by a local arming step.
* This does **not** protect against a compromised host OS or local malware.

---

## Cryptography Status Summary

### Currently used (NovaKey-Daemon v3)

* **KEM:** ML-KEM-768 (Kyber-class), via `filippo.io/mlkem768`.
* **KDF:** HKDF with SHA-256 for deriving per-message session keys.
* **AEAD:** XChaCha20-Poly1305 for encrypting and authenticating payloads.
* **Per-device secrets:** 32-byte keys in `devices.json` used for device identity and in key derivation.
* **Anti-abuse primitives:**

  * Timestamp-based freshness checks,
  * Per-device nonce replay cache,
  * Per-device rate limiting.

### Future / Planned Enhancements

These are possibilities under consideration and **not guaranteed**:

* Persistent replay state across daemon restarts for stronger replay resistance.
* Configurable “paranoid” modes with tighter time windows and per-device policies.
* Optional “approve before typing” flows that involve user confirmation on the phone.
* More granular control over injection targets (allow-listing certain applications / window classes).
* Additional transports (e.g. BLE, USB) when they can be implemented safely and portably.

The README and protocol documentation in the repository describe the exact on-wire format and cryptographic choices implemented in each release.

---

Thank you for helping keep NovaKey-Daemon secure.

— Robert H. Osborne (OsbornePro)
Maintainer, NovaKey-Daemon
