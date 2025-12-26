# Pairing (Daemon side)

Pairing happens over the same TCP listener as messages, but uses the `/pair` route.

## When does the daemon generate a QR?
Typically when:
- there are **no paired devices**
- the device store is missing/empty

If a device store exists, you may need to explicitly re-pair via your workflow or remove the store (only if you intend to re-pair all devices).

## Pairing is sensitive
Treat pairing output and device keys like passwords.

## Single-port pairing route
Clients must send:

```text
NOVAK/1 /pair\n
```

**Pairing flow summary:**
1. Client sends a plaintext JSON hello with a one-time token
2. Server replies with its ML-KEM public key and fingerprint
3. Client verifies fingerprint matches the QR
4. Client sends an encrypted register request
5. Server persists device keys and acknowledges

For exact wire format details, see Security → Protocol Summary.

---
