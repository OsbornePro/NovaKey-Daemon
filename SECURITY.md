# Security Policy

**NovaKey** is a security-critical application that receives encrypted secrets over a local TCP connection and injects them into the active window. We take security extremely seriously.

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
* Potential impact (e.g., credential leakage, privilege escalation, replay abuse)
* Proof-of-concept if available
* Any relevant logs, config snippets, or screenshots (with secrets redacted)

### What to expect

* You will receive an acknowledgment **within 24 hours** (usually much faster).
* We will validate the report and respond with next steps **within 3 business days**.
* If the report is valid, a fix will be shipped **as soon as possible** — typically within 7 days for critical issues when practical.
* You will be credited in the release notes unless you prefer to remain anonymous.

### Bug bounty

We do not currently offer monetary rewards, but we are deeply grateful for high-quality reports and will:

* Publicly credit you (with your permission)
* Prioritize your future reports
* Send NovaKey stickers and eternal gratitude

### Non-qualifying issues

The following are **out of scope** (but still appreciated if reported):

* Physical access to an unlocked computer
* Issues requiring root/admin on an already compromised machine
* Social engineering, phishing, or user-training issues
* Denial-of-service via local OS resource limits (e.g., exhausting CPU by running thousands of local clients)
* Attacks that assume full compromise of the host OS, kernel, or hypervisor

---

## Security Features

NovaKey is designed with defense-in-depth. The current implementation includes:

* **Per-device symmetric keys**
  Each device has its own 32-byte key, stored locally in `devices.json`. Only devices that possess the correct key can send valid requests.

* **Authenticated encryption with XChaCha20-Poly1305**
  All payloads are encrypted and authenticated using XChaCha20-Poly1305. The header (including device ID and message type) is covered by AEAD additional data (AAD).

* **Freshness & replay protection**
  Each message carries a Unix timestamp and a random nonce. The server enforces a strict time window, tracks seen nonces per device, and rejects stale or replayed messages.

* **Per-device rate limiting**
  Each device is limited to a configurable number of requests per minute, reducing the impact of abusive or compromised clients.

* **Strict framing & length checks**
  Every request is length-prefixed with a 16-bit size, and the server enforces a configurable `max_payload_len` before attempting decryption.

* **Local-only or LAN-only, no cloud**
  NovaKey listens on a locally configured TCP address (e.g., `127.0.0.1:60768` for local-only, or `0.0.0.0:60768` for LAN use). There are **no external servers** and no cloud backend.

* **No special privileges at runtime**
  NovaKey is designed to run as a normal user-level process or unprivileged service. Elevated rights are only needed for installation/service wiring, not for normal operation.

* **Truncated logging of sensitive data**
  Passwords are never fully logged. Logs use a safe preview (e.g., `Sup... (len=23)`) and never include full secrets or keys.

### Future / Planned Security Features

These are part of the roadmap but **not yet implemented**:

* **Post-quantum key exchange (Kyber-768)** layered on top of per-device identity keys.
* **Automatic session key rotation** derived from a PQ KEM.
* **Packaged, hardened system service configs** for systemd/launchd/Windows Service with least-privilege defaults.
* **BLE or other transport support** when it can be done safely and portably.

---

## Threat Model (High Level)

### In scope

* Network attackers on the **local network** attempting:

  * To read or modify NovaKey traffic.
  * To inject forged packets to trigger typing.
  * To replay previously captured packets.
* Malicious or compromised client apps that know the IP/port but **do not** have valid per-device keys.
* Attempts to abuse the service with excessive requests from a legitimate device.

### Out of scope / assumed trust

* Fully compromised host operating system or kernel.
* Malicious software with the same or greater privilege as NovaKey (e.g., another process with keyboard injection rights or accessibility privileges).
* Physical attacks on the machine (e.g., hardware keyloggers, cold boot attacks).
* User choosing weak passwords or re-using secrets in insecure ways.
* Attacks on build infrastructure or distribution channels outside the scope of this repository.

---

## Cryptography Status Summary

* **Currently used:**

  * XChaCha20-Poly1305 AEAD
  * Per-device symmetric keys from `devices.json`
  * Timestamp + nonce + replay cache + per-device rate limits

* **Planned / not yet active:**

  * Kyber-768 or similar PQ KEM for key agreement
  * Session key derivation and rotation layered on top of device identity

The README and protocol documentation in the repository describe the exact on-wire format and cryptographic choices implemented in each release.

---

Thank you for helping keep NovaKey secure.

— Robert H. Osborne (OsbornePro)
Maintainer, NovaKey

