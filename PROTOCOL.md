# NovaKey Wire Protocol

NovaKey uses a single TCP listener (`listen_addr`, default `0.0.0.0:60768`) and routes each connection by an initial ASCII preface line (**required**):

- `NOVAK/1 /pair\n` — pairing (*Pairing Protocol v1*)
- `NOVAK/1 /msg\n`  — encrypted messages (*Protocol v3*)

Connections that do not begin with one of these exact lines are rejected before any cryptographic processing.

---

## 1) Transport and Routing

### Listener

- Protocol: TCP
- Address: `listen_addr`

### Route preface (required)

Each client connection **must** begin with exactly one of:

```text
NOVAK/1 /pair\n
NOVAK/1 /msg\n
````

Routing is performed strictly by this ASCII preface line.

Connections without a valid preface are rejected immediately and closed.

### Connection lifetime

Each TCP connection handles exactly **one request**:

* `/pair`: one pairing exchange
* `/msg`: one typed message (approve / arm / disarm / inject)

The server enforces read/write deadlines and closes idle or stalled connections.
Clients must open a new TCP connection for each request.

---

## 2) Pairing Protocol v1 (`/pair`)

Pairing uses:

* One-time pairing token
* ML-KEM-768
* HKDF-SHA-256
* XChaCha20-Poly1305

Pairing occurs only when no devices are currently paired (empty or missing device store).

A typical QR payload uses an application-defined URI scheme, for example:

```text
novakey://pair?v=4&host=<host>&port=<port>&token=<b64url>&fp=<fp16hex>&exp=<unix>
```

(*Exact QR payload format is application-defined but should remain stable once clients depend on it.*)

---

### 2.1 Hello (plaintext JSON line)

Client → Server:

```text
{"op":"hello","v":1,"token":"<b64url>"}\n
```

Rules:

* `op` must be `"hello"`
* `v` must be `1`
* `token` must be valid, unexpired, and unused

---

### 2.2 Server key (plaintext JSON line)

Server → Client:

```text
{"op":"server_key","v":1,"kid":"1","kyber_pub_b64":"...","fp16_hex":"...","expires_unix":...}\n
```

Fingerprint calculation:

```text
fp16_hex = hex( sha256(pubkey)[0:16] )
```

Clients must verify that `fp16_hex` matches the fingerprint embedded in the QR code.

---

### 2.3 Register (encrypted binary frame)

Client encapsulates:

```text
ct, ss = MLKEM768_Encapsulate(serverPub)
```

Client sends:

```text
[ctLen u16 BE][ct bytes][nonce 24][ciphertext...]
```

#### Key derivation

```text
K = HKDF-SHA256(
  IKM  = ss,
  salt = tokenBytes,
  info = "NovaKey v3 Pair AEAD",
  outLen = 32
)
```

#### AEAD

* Algorithm: XChaCha20-Poly1305
* Nonce: 24 random bytes

#### AAD

```text
AAD = "PAIR" || ct || nonce
```

#### Plaintext JSON

```json
{"op":"register","v":1,"device_id":"...","device_key_hex":"..."}
```

If `device_id` or `device_key_hex` is empty, the server assigns:

* `device_id = "ios-" + randHex(8)`
* `device_key_hex = randHex(32)` (32 bytes)

The server persists device keys and reloads the device store.

---

### 2.4 Ack (encrypted)

Server replies (no length prefix):

```text
[nonce 24][ciphertext...]
```

Plaintext JSON:

```json
{"op":"ok","v":1,"device_id":"..."}
```

AAD:

```text
AAD = "PAIR" || ct || ackNonce
```

---

## 3) Message Protocol v3 (`/msg`)

### 3.1 TCP outer framing

```text
[u16 length (big-endian)][payload bytes...]
```

---

### 3.2 v3 payload layout

```text
[0]                = version (u8, must be 3)
[1]                = outer msgType (u8, must be 1)
[2]                = idLen (u8)
[3 : 3+idLen]      = deviceID bytes (UTF-8)

H = 3 + idLen
[H : H+2]          = kemCtLen (u16 BE)
[H+2 : ...]        = kemCt (kemCtLen bytes)

K = H + 2 + kemCtLen
[K : K+24]         = nonce (24 bytes)
[K+24 : end]       = ciphertext (AEAD output)
```

#### AAD

```text
AAD = payload[0 : K]
```

---

### 3.3 Plaintext inside AEAD (required)

After decrypting the AEAD ciphertext, the plaintext is:

```text
[0..7]   = timestamp (uint64 BE, unix seconds)
[8..end] = inner typed frame (v1)
```

Messages that do not contain a valid inner typed frame are rejected.

---

### 3.4 Inner typed message frame (v1)

```text
[0]   = innerVersion (u8) = 1
[1]   = innerMsgType (u8)
[2:4] = deviceIDLen (u16 BE)
[4:8] = payloadLen  (u32 BE)
[..]  = deviceID bytes (UTF-8)
[..]  = payload bytes
```

Rules:

* Inner `deviceID` **must** match the outer `deviceID`
* Payload is UTF-8 text (may be empty depending on message type)

#### Inner message types

| Type | Name    | Description                          |
| ---- | ------- | ------------------------------------ |
| 1    | Inject  | Inject secret into the focused field |
| 2    | Approve | Opens approval window (Two-Man Mode) |
| 3    | Arm     | Arms injection gate for a duration   |
| 4    | Disarm  | Immediately clears armed state       |

Only these message types are accepted.

---

### 3.5 `/msg` key schedule

Algorithms:

* ML-KEM-768
* HKDF-SHA-256
* XChaCha20-Poly1305

Key derivation:

```text
K = HKDF-SHA256(
  IKM  = kemShared,
  salt = deviceKey (32 bytes),
  info = "NovaKey v3 AEAD key",
  outLen = 32
)
```

---

## 4) Injection Result Signaling

After processing a `/msg` Inject request, the server replies with:

* a numeric `status`
* a semantic `reason`

These fields allow clients to present accurate user feedback.

### Relevant `reason` values

| Reason                       | Meaning                                              |
| ---------------------------- | ---------------------------------------------------- |
| `ok`                         | Direct injection succeeded                           |
| `typing_fallback`            | Auto-typing fallback was used                        |
| `clipboard_fallback`         | Clipboard paste or clipboard-only fallback           |
| `inject_unavailable_wayland` | Injection unavailable on Wayland; clipboard fallback |

### Clipboard-only indication

A `status` value of `OK_CLIPBOARD` indicates:

* the clipboard was set by the daemon
* the user must manually paste to complete insertion

Clients should present a clear visual cue in this case.

---

## Security Notes

* NovaKey is designed for LAN or local use
* Do not expose the daemon directly to the public Internet
* Use host firewall rules to restrict access as appropriate
