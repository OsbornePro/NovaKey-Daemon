# Protocol Summary (User-friendly)

This page summarizes the daemon’s `PROTOCOL.md` in a way that helps troubleshoot and understand behavior.

NovaKey uses one TCP listener and routes each connection by a one-line preface:
- `NOVAK/1 /pair\n` for pairing
- `NOVAK/1 /msg\n` for approve/inject

If no preface is sent, the daemon treats it as `/msg` (compatibility).

## Pairing route (`/pair`)
Pairing is a one-time trust bootstrap:
- client sends a one-time token (plaintext JSON line)
- server responds with its ML-KEM public key and fingerprint
- client checks fingerprint matches the QR
- client sends an encrypted register frame
- server stores device keys and acknowledges

## Message route (`/msg`)
Each request is one connection:
- outer frame includes versioning + device id + ML-KEM ciphertext + nonce + AEAD ciphertext
- inner plaintext includes a timestamp + typed message:
  - inject: secret payload
  - approve: opens approval window (payload may be empty)

## Why you sometimes see clipboard instead of typing
Injection can be denied by:
- OS permissions (macOS accessibility)
- compositor restrictions (Wayland)
- secure input mode in focused app
- target policy rules
- arming/two-man gates not satisfied

In those cases, the daemon may return “clipboard success” instead of inject success.

