# NovaKey Protocol v3 (Kyber + XChaCha20)

**Status:** implemented (Kyber key schedule + v3 framing)
**Scope:** Typing daemon (Linux / macOS / Windows) ⇄ clients (`nvclient`, future phone app)

This describes how clients send “*type this password*” requests to the NovaKey service over TCP using:

* **ML-KEM-768** (Kyber-768-compatible KEM) for post-quantum key establishment
* **XChaCha20-Poly1305** for authenticated encryption of the payload
* Per-device pre-shared keys as an extra layer of authentication

There is **no backward compatibility** with protocol v2. Frames with `version != 3` are rejected.

---

## 1. Transport

* **Protocol:** TCP
* **Default port:** `60768`
* **Default listen address:** configured in `server_config.json`, typically:

  * `127.0.0.1:60768` (dev)
  * `0.0.0.0:60768` (LAN / VPN use)

Each request is a single TCP connection:

1. Client connects.
2. Client sends one framed message.
3. Server processes, injects, then closes the connection.

No connection reuse or multiplexing; each “type this secret” is independent.

---

## 2. Pairing & Key Material

NovaKey v3 has **three** relevant secrets:

1. **Server Kyber keypair** (ML-KEM-768)
2. **Per-device pre-shared key** (PSK) for the AEAD
3. **Per-message KEM shared secret**, derived from Kyber for that request

### 2.1 Server Kyber keys (`server_keys.json`)

On first startup, the daemon will create `server_keys.json` if it does not exist:

```json
{
  "kyber768_public": "<base64-encoded ML-KEM-768 public key>",
  "kyber768_secret": "<base64-encoded ML-KEM-768 private key>"
}
```

These are **long-lived** for the workstation and used only for:

* Accepting KEM ciphertexts from paired devices
* Decapsulating to obtain per-message shared secrets

If the file is missing or invalid, it is **regenerated automatically**.

### 2.2 Device keys (`devices.json`)

Per-device PSKs are still used as an additional authentication layer and to bind pairing to a particular phone/device.

`devices.json` looks like:

```json
{
  "devices": [
    {
      "id": "roberts-phone",
      "key_hex": "7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234"
    }
  ]
}
```

* `key_hex` is a 32-byte random value (XChaCha20-Poly1305 key material) encoded as hex.
* Generated and written by the `nvpair` tool.

### 2.3 Pairing JSON (for phone app / nvclient)

The `nvpair` tool does all of this:

* Creates or updates the device entry in `devices.json`
* Reads `server_config.json` and `server_keys.json`
* Emits pairing info as JSON (and optionally as a QR code)

Example structure:

```json
{
  "v": 1,
  "device_id": "roberts-phone",
  "device_key_hex": "7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234",
  "server_addr": "192.168.8.244:60768",
  "server_kyber768_pub": "<base64-encoded public key>"
}
```

A phone app or external client can just:

* Parse/scan this JSON (or QR)
* Store:

  * `device_id`
  * `device_key_hex`
  * `server_addr`
  * `server_kyber768_pub`

There is no user-typed secret during pairing; everything is generated on the host, exported via JSON/QR, and imported by the client.

---

## 3. On-the-wire framing

At the TCP level:

```text
[ u16 length ][ length bytes of payload ]
```

* `length`: unsigned 16-bit big-endian (0–65535)
* `payload`: a single **v3** encrypted message (`frame` in the Go code)

---

## 4. Payload (v3) layout

The `payload` / `frame` bytes are:

```text
[ 0 ]               = version        (u8, must be 3)
[ 1 ]               = msgType        (u8, 1 = password)
[ 2 ]               = idLen          (u8, length of deviceID in bytes)
[ 3 .. 3+idLen-1 ]  = deviceID       (ASCII/UTF-8)
[ ... ]             = kemCt          (ML-KEM-768 ciphertext, fixed length)
[ ... ]             = nonce          (24 bytes, XChaCha20-Poly1305 nonce)
[ ... ]             = ciphertext     (AEAD output)
```

More explicitly:

```text
offset  description
------  -----------------------------------------------
0       version = 0x03
1       msgType = 0x01 (password)
2       idLen = N
3..3+N-1 deviceID (N bytes)
H       = 3 + N
H..H+K-1  kemCt (K bytes; ML-KEM-768 ciphertext)
H+K..H+K+23  nonce (24 bytes)
H+K+24..end  ciphertext (XChaCha20-Poly1305)
```

Where:

* `K` is the ML-KEM-768 ciphertext size (currently 1088 bytes with the Filippo implementation).

### 4.1 AEAD Associated Data (AAD)

The AEAD **associated data** is the fixed header **before** the KEM ciphertext:

```text
header = frame[0 : 3 + idLen]
       = version || msgType || idLen || deviceID
```

That means:

* Version and device identity are authenticated by AEAD.
* KEM ciphertext itself is integrity-protected by the ML-KEM-768 scheme.

---

## 5. Cipher & key schedule

### 5.1 Algorithms

* **Key Encapsulation Mechanism**: ML-KEM-768 (Kyber-768-compatible)
* **Symmetric AEAD**: XChaCha20-Poly1305
* **Key Derivation**: HKDF-SHA-256

### 5.2 Per-message key derivation (client)

For each request, the client does:

1. **KEM encapsulation**

   ```text
   input: serverKyberPub (from pairing)
   output: kemCt (ciphertext), kemShared (32-byte shared secret)
   ```

2. **Session key derivation with HKDF**

   ```text
   IKM  = kemShared       (from KEM)
   salt = deviceKey       (32-byte PSK from device_key_hex)
   info = "NovaKey v3 session key" (fixed ASCII label)
   K    = HKDF-SHA256(IKM, salt, info, length = 32)
   ```

   `K` becomes the XChaCha20-Poly1305 key for this single message.

3. **AEAD encryption**

   * Build `header` and `plaintext` (see below).
   * Generate random 24-byte nonce.
   * Use XChaCha20-Poly1305 with:

     * Key = `K`
     * Nonce = 24 random bytes
     * AAD = `header`
   * Result is `ciphertext`.

The **plaintext** layout inside AEAD is unchanged from v2:

```text
[ 0..7 ]   = timestamp (uint64, big-endian Unix seconds)
[ 8..end ] = password bytes (UTF-8)
```

Typical client pseudocode, ignoring errors:

```pseudo
now       = unix_time()
password  = "SuperStrongPassword123!"
idBytes   = utf8_bytes(deviceID)

header = [0x03, 0x01, len(idBytes)] || idBytes

plaintext = uint64_be(now) || utf8_bytes(password)

// KEM: derive kemCt, kemShared
kemCt, kemShared = MLKEM768_Encapsulate(serverKyberPub)

// HKDF: derive session AEAD key
deviceKey = hex_decode(device_key_hex)     // 32 bytes
K = HKDF_SHA256(
      ikm  = kemShared,
      salt = deviceKey,
      info = "NovaKey v3 session key",
      outLen = 32)

// AEAD: XChaCha20-Poly1305
nonce      = random_24_bytes()
ciphertext = XChaCha20_Encrypt(K, nonce, plaintext, aad=header)

// Frame payload
payload = header || kemCt || nonce || ciphertext

// Outer framing
length = len(payload)  // must fit in u16
frame  = uint16_be(length) || payload

tcp_send("server:60768", frame)
```

### 5.3 Server-side key derivation

On the server, `decryptPasswordFrame` essentially does the reverse:

1. Parse header (`version`, `msgType`, `idLen`, `deviceID`).

2. Lookup `deviceKey` (`key_hex`) for `deviceID` in `devices.json`.

3. Extract `kemCt`, `nonce`, and `ciphertext` from frame.

4. Base64-decode and unmarshal server Kyber private key from `server_keys.json`.

5. **KEM decapsulation**:

   ```text
   kemShared = MLKEM768_Decapsulate(serverKyberPriv, kemCt)
   ```

6. **HKDF** with the same parameters as the client:

   ```text
   IKM  = kemShared
   salt = deviceKey (32 bytes)
   info = "NovaKey v3 session key"
   K    = HKDF-SHA256(IKM, salt, info, length = 32)
   ```

7. Build a temporary XChaCha20-Poly1305 cipher with `K`.

8. Decrypt `ciphertext` using `nonce`, `header` as AAD.

9. Parse resulting plaintext: `[timestamp || password]`.

10. Run freshness, replay, and rate-limit checks (section 6).

If everything passes, the result is the cleartext password to inject.

---

## 6. Server-side validation

After decrypting the payload, NovaKey applies the same checks as v2 with updated version:

### 6.1 Version & msgType

Reject if:

* `version != 3`
* `msgType != 1`

### 6.2 Device lookup

* Extract `deviceID` from header.
* Look up in `devices.json`.
* Reject if unknown device ID.

### 6.3 Timestamp freshness

Let:

* `now = time.Now().Unix()` (seconds)
* `ts = timestamp from plaintext`

Defaults (from code):

* `maxClockSkewSec = 120` (±2 minutes)
* `maxMsgAgeSec    = 300` (5 minutes)

Checks:

* Reject if `ts > now + maxClockSkewSec` → “timestamp is in the future”.
* Reject if `now - ts > maxMsgAgeSec` → “message too old”.

### 6.4 Replay protection

In-memory replay cache:

```text
replayCache[deviceID][nonceHex] = timestamp
```

On each message:

1. Convert AEAD nonce to hex (`nonceHex`).
2. If `replayCache[deviceID][nonceHex]` already exists → reject as replay.
3. Otherwise store `replayCache[deviceID][nonceHex] = ts`.
4. Periodically drop entries older than `replayCacheTTL` (default: 600 seconds).

### 6.5 Rate limiting

Simple per-device sliding window:

```text
rateState[deviceID] = {
    windowStart int64 // Unix seconds
    count       int
}
```

Defaults:

* `maxRequestsPerDevicePerMin = 60`
* Can be overridden by `max_requests_per_min` in `server_config.json`.

On each **accepted** message (before injection):

1. If `now - windowStart >= 60` seconds or `windowStart == 0`:

   * Reset `windowStart = now`, `count = 0`.
2. Increment `count`.
3. If `count > maxRequestsPerDevicePerMin`:

   * Reject as “rate limit exceeded”.

---

## 7. Injection behavior (summary)

Once all checks pass, NovaKey calls:

```go
InjectPasswordToFocusedControl(password)
```

Platform-specific behavior:

* **Linux**

  * On X11/XWayland:

    * Uses `xdotool` to type or paste into the active window.
    * Optionally copies password to clipboard (`xclip`) as a convenience.
  * On pure Wayland:

    * Currently **not supported** (see `KNOWN_ISSUES.md`).
    * The daemon logs a clear message indicating the limitation.
* **macOS**

  * Uses AppleScript / Accessibility APIs to simulate paste or keystrokes.
* **Windows**

  * Tries clipboard + `EM_REPLACESEL` / `WM_SETTEXT` on known text controls.
  * Falls back to `keybd_event` typing if necessary.

NovaKey logs:

* Device ID
* A short, obfuscated preview of the password (`"Sup..." (len=23)`)
* Success or failure of the injection path

---

## 8. Example: implementing a non-Go client

To implement an external client (phone app, Rust CLI, etc.) you need:

1. **Pairing info** (either via JSON file or QR code):

   * `device_id`
   * `device_key_hex` (32-byte PSK)
   * `server_addr` (e.g. `192.168.8.244:60768`)
   * `server_kyber768_pub` (base64, ML-KEM-768 public key)
2. ML-KEM-768 implementation compatible with the Go `filippo.io/mlkem768` parameters.
3. XChaCha20-Poly1305 AEAD.
4. HKDF-SHA-256.
5. TCP client for framing: `[u16 length][payload]`.

Follow the pseudocode in section 5.2 and match:

* Exact framing offsets
* KEM cipher parameters
* HKDF inputs (IKM, salt, info)
* AEAD nonce length and AAD

If everything matches, NovaKey will:

* Decrypt successfully
* Validate timestamp / replay / rate limit
* Inject the password into the currently focused control

---

## 9. Files involved (Go implementation)

For reference:

* `cmd/novakey/config.go`
  Loads `server_config.json` and default values.

* `cmd/novakey/keys.go`
  Generates / loads ML-KEM-768 server keypair (`server_keys.json`).

* `cmd/novakey/crypto.go`

  * Loads device keys (`devices.json`)
  * Decapsulates KEM ciphertext
  * Derives per-message session key with HKDF
  * Decrypts payload
  * Enforces timestamp / replay / rate limits

* `cmd/novakey/linux_main.go`, `darwin_main.go`, `windows_main.go`
  TCP listener, framing, and hand-off into `decryptPasswordFrame` and injection.

* `cmd/novakey/inject_*.go`
  Platform-specific implementations of `InjectPasswordToFocusedControl`.

* `cmd/nvclient/`
  Reference Go client:

  * Parses `device_id` + `key_hex` + server addr + server pub
  * Performs KEM encapsulation, HKDF, AEAD, and framing.

* `cmd/nvpair/`
  CLI tool to:

  * Add/update devices in `devices.json`
  * Read `server_config.json` + `server_keys.json`
  * Emit pairing JSON (and ASCII QR) for the phone app.

---
