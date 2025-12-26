# NovaKey Protocol (Current)

This document describes the **currently implemented** NovaKey wire behavior:

- One TCP listener (`listen_addr`, default `:60768`)
- Route preface line: `NOVAK/1 /pair` or `NOVAK/1 /msg`
- `/msg` uses **ML-KEM-768 + HKDF-SHA-256 + XChaCha20-Poly1305** (protocol version 3)
- Typed inner message frames distinguish **inject** vs **approve**

---

## 1) Transport and Routing

### Listener
- Protocol: TCP
- Address: `listen_addr` from config (default `127.0.0.1:60768`)

### Route preface (recommended)
A client may begin the TCP stream with a single line:

- `NOVAK/1 /pair\n`
- `NOVAK/1 /msg\n`

After this line, the remainder of the connection is handled by that route.

### Backward compatibility
If a client does **not** send the `NOVAK/1` preface line, the daemon treats the connection as `/msg`.

---

## 2) Pairing (`/pair`)

Pairing is performed over the same TCP port via the `/pair` route.

### QR contents
When the daemon is unpaired, it generates a QR that contains:

- daemon host and port
- a short-lived pairing token
- a fingerprint of the daemon ML-KEM public key (sanity check)
- expiration timestamp

### Pairing properties
- The token is required and expires.
- Pairing returns the per-device secret used to authenticate/encrypt future `/msg` requests.
- Treat pairing output as sensitive.

(Implementation details of exact `/pair` request/response bytes are defined by the current `pairing_proto.go` and are considered the source of truth.)

---

## 3) Message channel (`/msg`) — Protocol v3

This is the encrypted request path used for injection and two-man approval.

### TCP outer framing
A `/msg` request is a single TCP connection with a single length-prefixed payload:

