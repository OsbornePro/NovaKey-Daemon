# Protocol Summary

This page summarizes the daemon’s `PROTOCOL.md` in a way that helps troubleshoot and understand behavior.

NovaKey uses a custom secure protocol designed for:
- Mutual authentication
- Replay protection
- Forward secrecy

Each message includes:
- Nonce
- Timestamp
- Device identifiers
- Cryptographic authentication

Invalid or replayed messages are rejected.

NovaKey uses one TCP listener and routes each connection by a one-line preface:
- `NOVAK/1 /pair\n` for pairing
- `NOVAK/1 /msg\n` for approve/inject

Clients must send a route preface line (NOVAK/1 /msg\n or NOVAK/1 /pair\n). Connections without a valid preface are rejected.

### Message Types (Protocol v3)

All `/msg` requests decrypt to a timestamp followed by a **required inner typed message frame (v1)**.
Exactly one inner message type (1–4) is permitted per request:

| Type | Name    | Description                                                                            |
| ---- | ------- | -------------------------------------------------------------------------------------- |
| 1    | Inject  | Injects the secret payload into the currently focused field (subject to policy gates). |
| 2    | Approve | Opens a short approval window allowing a subsequent Inject (Two-Man Mode).             |
| 3    | Arm     | Arms the daemon for a limited duration, enabling injection (“push-to-type”).           |
| 4    | Disarm  | Clears the armed state immediately, blocking further injection.                        |

This table is normative for all NovaKey documentation; other pages must reference this section rather than restating message types.

Messages that do not contain a valid **Inner Message Frame v1** with one of the above types are rejected.
There is no legacy or untyped message support.


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
inner plaintext includes a timestamp + typed message:
- inject: secret payload
- approve: opens approval window
- arm: arms the daemon for a limited duration
- disarm: clears armed state

## Why you sometimes see clipboard instead of typing
Injection can be denied by:
- OS permissions (macOS accessibility)
- compositor restrictions (Wayland)
- secure input mode in focused app
- target policy rules
- arming/two-man gates not satisfied

In those cases, the daemon may return “clipboard success” instead of inject success.

