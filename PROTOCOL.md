# NovaKey Protocol v2

**Status:** implemented
**Scope:** Typing daemon (Linux/macOS/Windows) ⇄ clients (nvclient, future phone app)

This describes how clients send “type this password” requests to the NovaKey service over TCP.

---

### 1. Transport

* **Protocol:** TCP
* **Default port:** `60768`
* **Default listen address:** configured in `server_config.json`, typically:

  * `127.0.0.1:60768` (dev)
  * `0.0.0.0:60768` (LAN testing)

Each request is a single TCP connection:

1. Client connects.
2. Client sends one framed message.
3. Server processes, injects, then closes the connection.

---

### 2. High-level flow

1. Client has:

   * `deviceID` (string)
   * 32-byte secret key (hex) for that device (from `devices.json` / `nvpair`).

2. Client builds **one request**:

   * Packs timestamp + password into a plaintext structure.
   * Encrypts with XChaCha20-Poly1305 using device key.
   * Wraps in a versioned frame with deviceID + nonce.

3. Server:

   * Reads frame.
   * Extracts deviceID.
   * Looks up that device’s AEAD cipher (from `devices.json`).
   * Decrypts + verifies timestamp, replay, rate limits.
   * Injects password into focused control.

---

### 3. On-the-wire framing

At the TCP level:

```text
[ u16 length ][ length bytes of payload ]
```

* `length` is unsigned 16-bit big-endian (0–65535).
* `payload` is a v2 encrypted message.

---

### 4. Payload (v2) layout

`payload` bytes (what Go calls `frame`) are:

```text
[ 0 ]             = version           (u8, must be 2)
[ 1 ]             = msgType           (u8, 1 = password)
[ 2 ]             = idLen             (u8, length of deviceID in bytes)
[ 3 .. 3+idLen-1] = deviceID bytes    (ASCII/UTF-8)
[ ... ]           = nonce             (24 bytes, XChaCha20-Poly1305 nonce)
[ ... ]           = ciphertext        (rest, AEAD output)
```

More explicitly:

```text
offset  description
------  -----------------------------------------------
0       version (0x02)
1       msgType (0x01)
2       idLen (N)
3..3+N-1 deviceID (N bytes)
H       = 3 + N
H..H+23 nonce (24 bytes)
H+24..  ciphertext (AEAD)
```

* The **AEAD additional data (AAD)** is the header:

  * `header = payload[0 : 3+idLen]`
  * i.e. `version || msgType || idLen || deviceID`

---

### 5. Cipher & keys

* Cipher: **XChaCha20-Poly1305** (via `golang.org/x/crypto/chacha20poly1305.NewX`).

* Key: 32 bytes, unique per device, stored in `devices.json`:

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

* Nonce: 24 random bytes, new for each message.

---

### 6. Plaintext layout (inside AEAD)

After decryption, the plaintext is:

```text
[ 0..7 ]   = timestamp (uint64, big-endian Unix seconds)
[ 8..end ] = password bytes (UTF-8)
```

* `timestamp` is the client’s wall-clock time in seconds.

---

### 7. Server-side validation

Once NovaKey decrypts, it applies several checks:

#### 7.1 Version & msgType

* Reject if:

  * `version != 2`
  * `msgType != 1`

#### 7.2 Device lookup

* Extract `deviceID` from header.
* Look up in `devices.json`:

  * Reject if unknown device ID.

#### 7.3 Timestamp freshness

Let:

* `now = time.Now().Unix()` (seconds)
* `ts = timestamp from plaintext`

Configuration constants (current defaults in code):

* `maxClockSkewSec = 120` (±2 minutes)
* `maxMsgAgeSec = 300` (5 minutes)

Checks:

* Reject if `ts > now + maxClockSkewSec`:

  * “timestamp is in the future”
* Reject if `now - ts > maxMsgAgeSec`:

  * “message too old”

#### 7.4 Replay protection

NovaKey maintains an in-memory map:

```text
replayCache[deviceID][nonceHex] = timestamp
```

On each message:

1. Convert nonce to hex (`nonceHex`).
2. If `replayCache[deviceID][nonceHex]` already exists:

   * Reject as replay.
3. Otherwise:

   * Store `replayCache[deviceID][nonceHex] = ts`.
4. Periodically (on insert), delete entries older than `replayCacheTTL` (currently 600 seconds).

This means:

* Reusing the **same nonce** from the same device will be rejected, even if timestamp is fresh.
* Old nonce entries get dropped after ~10 minutes.

#### 7.5 Rate limiting

NovaKey also tracks a simple sliding-ish window per device:

```text
rateState[deviceID] = {
    windowStart int64 // Unix seconds
    count       int
}
```

Defaults:

* `maxRequestsPerDevicePerMin = 60`
* Overridden by `max_requests_per_min` in `server_config.json`.

On each **accepted** message (before injection):

1. If `now - windowStart >= 60` seconds or `windowStart == 0`:

   * Reset `windowStart = now`, `count = 0`.
2. Increment `count`.
3. If `count > maxRequestsPerDevicePerMin`:

   * Reject as “rate limit exceeded”.

---

### 8. Injection behavior (summary)

After all checks pass, NovaKey:

* Calls `InjectPasswordToFocusedControl(password)`:

  * **Linux**:

    * Copies password to clipboard (async).
    * Uses `xdotool` to type or paste into focused control (depending on environment).
  * **macOS**:

    * Uses AppleScript / Accessibility APIs to simulate paste/typing.
  * **Windows**:

    * Uses `SendInput` approach (and fallback) plus optional clipboard set.

* Logs:

  * Device ID.
  * Obscured password preview (`"Sup..." (len=23)`).
  * Success/failure of injection.

Clipboard behavior is best-effort and will be further tuned per OS.

---

### 9. Example: building a client (language-agnostic)

To talk to NovaKey, a client must:

1. Know `deviceID` and 32-byte key.
2. Implement XChaCha20-Poly1305.
3. Implement TCP framing + this layout.

Pseudo-steps:

```pseudo
deviceID = "roberts-phone"
key = decode_hex("7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234")
now = current_unix_time_seconds()
password = "SuperStrongPassword123!"

// Build plaintext = [timestamp || password]
plaintext = byte_array(8 + len(password))
write_uint64_be(plaintext[0:8], now)
copy(plaintext[8:], utf8_bytes(password))

// Header (AAD)
idBytes = utf8_bytes(deviceID)
if len(idBytes) > 255: error
header = [0x02, 0x01, len(idBytes)] || idBytes

// AEAD(XChaCha20-Poly1305)
nonce = random_24_bytes()
ciphertext = AEAD_Encrypt(key, nonce, plaintext, aad=header)

// Frame payload
payload = header || nonce || ciphertext

// Outer framing
length = len(payload)  // must fit in 16 bits
frame = uint16_be(length) || payload

// Send over TCP
tcp_send("server:60768", frame)
```

If everything is correct, NovaKey will:

* Validate everything,
* Inject the password into the focused control.

---

### 10. Files involved (Go implementation)

For reference in the repo:

* `cmd/novakey/crypto.go`

  * Implements device key loading, AEAD, timestamp/replay/rate checks.
* `cmd/novakey/linux_main.go`

  * Linux main + handler.
* `cmd/novakey/darwin_main.go`

  * macOS main + handler.
* `cmd/novakey/windows_main.go`

  * Windows main + handler.
* `cmd/nvclient/`

  * Reference Go client implementation.
* `cmd/nvpair/`

  * CLI tool to add/update devices in `devices.json`.

---
