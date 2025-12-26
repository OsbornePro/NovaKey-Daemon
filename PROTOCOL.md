# NovaKey Wire Protocol

NovaKey uses a single TCP listener (`listen_addr`, default `127.0.0.1:60768`) and routes each connection by an initial ASCII line:

- `NOVAK/1 /pair\n` — pairing (Pairing Protocol v1)
- `NOVAK/1 /msg\n`  — encrypted messages (Protocol v3)

If the route line is absent, the connection is treated as `/msg`.

---

## 1) Transport and Routing

### Listener
- TCP
- address: `listen_addr`

### Route preface
The client may begin with:

- `NOVAK/1 /pair\n`
- `NOVAK/1 /msg\n`

After this line, the remaining bytes are interpreted by the selected route.

### Connection lifetime

Each TCP connection handles exactly one request:

- `/pair`: one pairing exchange
- `/msg`: one approve or inject message

The server enforces read/write deadlines and closes idle or stalled connections.
Clients must open a new connection for each request.

---

## 2) Pairing Protocol v1 (`/pair`)

Pairing uses:
- one-time pairing token (base64 raw URL)
- ML-KEM-768
- HKDF-SHA-256
- XChaCha20-Poly1305

### 2.1 Hello (plaintext JSON line)

Client → Server:
```text
{"op":"hello","v":1,"token":"<b64url>"}\n
````

Rules:

* `op` must be `"hello"`
* `v` must be `1`
* token is one-time and expires (server-side TTL)

### 2.2 Server key (plaintext JSON line)

Server → Client:

```text
{"op":"server_key","v":1,"kid":"1","kyber_pub_b64":"...","fp16_hex":"...","expires_unix":...}\n
```

Fingerprint:

* `fp16_hex = hex( sha256(pubkey)[0:16] )`

Client should verify `fp16_hex` matches what the QR indicated.

### 2.3 Register (encrypted binary frame)

Client encapsulates:

* `ct, ss = MLKEM768_Encapsulate(serverPub)`

Client sends:

```text
[ctLen u16 BE][ct bytes][nonce 24][ciphertext...]
```

Key derivation:

* `K = HKDF-SHA256(IKM=ss, salt=tokenBytes, info="NovaKey v4 Pair AEAD", outLen=32)`

AEAD:

* XChaCha20-Poly1305
* `nonce` = 24 random bytes

AAD:

```text
AAD = "PAIR" || ct || nonce
```

Plaintext JSON:

```json
{"op":"register","v":1,"device_id":"...","device_key_hex":"..."}
```

If `device_id` or `device_key_hex` is empty, the server assigns:

* `device_id = "ios-" + randHex(8)`
* `device_key_hex = randHex(32)` (32 bytes)

Server persists device keys and reloads them.

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
[ u16 length (big-endian) ][ payload bytes... ]
```

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

AAD:

```text
AAD = payload[0 : K]
```

### 3.3 Plaintext inside AEAD

```text
[0..7]   = timestamp (uint64 BE unix seconds)
[8..end] = inner typed frame (v1)
```

### 3.4 Inner typed message frame (v1)

```text
[0]   = innerVersion (u8) = 1
[1]   = innerMsgType (u8) = 1 inject, 2 approve
[2:4] = deviceIDLen (u16 BE)
[4:8] = payloadLen  (u32 BE)
[..]  = deviceID bytes (UTF-8)
[..]  = payload bytes (UTF-8)
```

Rules:

* inner deviceID must match outer deviceID
* msgType=1 inject: payload is the secret string
* msgType=2 approve: payload may be empty (ignored)

### 3.5 `/msg` key schedule

Algorithms:

* ML-KEM-768
* HKDF-SHA-256
* XChaCha20-Poly1305

Key derivation:

* `K = HKDF-SHA256(IKM=kemShared, salt=deviceKey(32), info="NovaKey v3 AEAD key", outLen=32)`

---

## 4) Server validation (summary)

* timestamp freshness (skew + max age)
* replay detection on `(deviceID, nonce)` within a TTL window
* per-device rate limiting
* optional arming gate and two-man gate (policy-level)

---

## 5) Notes

* Device IDs are sent in plaintext for routing. Do not use sensitive identifiers.

```
