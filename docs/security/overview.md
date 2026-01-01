# Security Overview

NovaKey is designed around a few core safety principles:

## Key principles
- Secrets are never displayed after creation
- Secrets are never logged
- All communication is encrypted
- Devices must be explicitly paired

NovaKey does not rely on cloud infrastructure.

## Secrets stay on the phone
- Secrets are stored only in the iOS Keychain.
- Secrets are never displayed again after saving.
- Sending/copying requires Face ID / passcode.

## Explicit trust via pairing
- Devices must be paired before any secrets can be accepted.
- Pairing validates server identity via a fingerprint embedded in the QR.

## Modern cryptography (post-quantum capable)
NovaKey uses:
- ML-KEM-768 (Kyber / ML-KEM class) for session establishment
- HKDF-SHA-256 for key derivation
- XChaCha20-Poly1305 for authenticated encryption

## Replay & abuse resistance
- timestamp freshness checks
- replay protection
- per-device rate limiting

## Policy gates (optional but recommended)
- arming (“push-to-type”)
- two-man approval
- focused target allow/deny lists
- injection safety rules (newline/length)

For wire details, see **Protocol Summary**.

