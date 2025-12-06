# Security Policy

**NovaKey** is a security-critical application that receives post-quantum encrypted secrets over Bluetooth Low Energy and injects them into the active window. We take security extremely seriously.

## Supported Versions

Security updates are provided for the **latest stable release only**.

| Version       | Supported          | Notes                                   |
|---------------|--------------------|-----------------------------------------|
| Latest release| Supported          | Receives security fixes promptly        |
| All others    | Not supported      | Upgrade recommended                     |

We follow a **rolling release** model: as soon as a new version is tagged on GitHub, it becomes the only supported version.

## Reporting a Vulnerability

We strongly encourage responsible disclosure of security vulnerabilities.

### How to report
Please **do not** open public GitHub issues for security problems.

Instead, send reports to:  
**security@novakey.app**  
(*or directly to **robert@osbornepro.com** if email is unavailable*)

[PGP key](https://downloads.osbornepro.com/publickey.asc) for rosborne@osbornepro.com (*Recommended for sensitive reports*)


### What to include
- Detailed steps to reproduce
- Affected version(s)
- Potential impact (e.g., credential leakage, code execution, replay attacks)
- Proof-of-concept if available

### What to expect
- You will receive an acknowledgment **within 24 hours** (usually much faster).
- We will validate the report and respond with next steps **within 3 business days**.
- If the report is valid, a fix will be shipped **as soon as possible** — typically within 7 days for critical issues.
- You will be credited in the release notes unless you prefer to remain anonymous.

### Bug bounty
While we do not currently offer monetary rewards, we are deeply grateful for high-quality reports and will:
- Publicly credit you (with your permission)
- Prioritize your future reports
- Send NovaKey stickers and eternal gratitude

### Non-qualifying issues
The following are **out of scope** (but still appreciated if reported):
- Physical access to an unlocked computer
- Issues requiring root/admin privileges on a compromised machine
- Social engineering or phishing
- Denial-of-service against the BLE peripheral

## Security Features

NovaKey is designed with defense-in-depth:

- Post-quantum cryptography (Kyber-768 + XChaCha20-Poly1305)
- Anti-replay protection via monotonic nonce
- Memory zeroing of all secrets
- No root/admin required at runtime
- Code signed on macOS with Bluetooth entitlements
- Runs as unprivileged user on Linux/macOS
- No network connectivity — BLE only

Thank you for helping keep NovaKey secure.

— Robert H. Osborne (OsbornePro)
Maintainer, NovaKey
