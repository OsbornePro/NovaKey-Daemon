# Pairing (Daemon side)

Pairing is the process by which a NovaKey client device (for example the iOS app)
establishes a trusted cryptographic relationship with the NovaKey Daemon.

Pairing happens over the same TCP listener as normal messages, but uses the
dedicated `/pair` route.

---

## When does the daemon generate a QR?

The daemon typically enters pairing mode and generates a QR code when:

- there are **no paired devices**
- the device store is missing or empty

If a device store already exists, the daemon assumes pairing has occurred and
will not automatically generate a new QR code.

If you intend to re-pair all devices, the existing device store must be removed
or the daemon reinstalled to force a fresh pairing bootstrap.

---

## Pairing attempts are intentionally limited

Pairing is treated as a **high-trust, one-time bootstrap operation**.

The daemon does **not** allow unlimited or indefinite pairing retries.
If pairing is interrupted, partially completed, or fails during secure
storage initialization, the daemon may refuse to re-enter pairing mode
automatically.

This behavior is intentional and designed to prevent:

- replay attacks
- downgrade attacks to weaker storage
- brute-force or repeated pairing attempts
- indefinite pairing windows

In some cases, restarting the daemon is sufficient to re-enter pairing mode.
In other cases, a **full uninstall and reinstall** is required to reset
pairing state.

---

## Linux secure storage considerations

On Linux, pairing depends on access to the system keyring or other
secure storage mechanisms.

If secure storage initialization fails (for example due to:

- hardware-backed authentication such as **YubiKey**
- cancelled keyring unlock prompts
- PAM configurations requiring external confirmation

the daemon may enter a **non-pairable state**.

Depending on configuration and failure timing, the daemon may:

- fall back to a local device store (`devices.json`), **or**
- refuse further pairing attempts until fully reinstalled

This behavior is expected and enforces pairing integrity.

---

## Pairing is security-sensitive

Treat pairing output, QR codes, and generated device keys like passwords.

Anyone who successfully completes pairing is granted the ability to send
secrets to the daemon, subject to configured safety gates.

---

## Single-port pairing route

Pairing uses the same listener and port as normal traffic.

Clients must initiate pairing by sending:

```text
NOVAK/1 /pair\n
```

---

## Pairing flow summary

1. Client sends a plaintext JSON **hello** containing a one-time pairing token
2. Server replies with:

   * its ML-KEM public key
   * a fingerprint rendered in the pairing QR
3. Client verifies the fingerprint matches the QR code
4. Client sends an encrypted **register** request
5. Server persists device keys and acknowledges pairing

Once pairing completes successfully, the daemon exits pairing mode.

For exact wire-format and cryptographic details, see:
**Security → Protocol Summary**

---

## Pairing rate limits

The daemon enforces in-memory rate limits on pairing requests
(for example `/pair` hello messages).

These limits protect against:

* brute-force pairing attempts
* fingerprint probing
* denial-of-service attacks on the pairing route

If the rate limit is exceeded, pairing requests are rejected until
the rate-limit window resets.

---

## Recovery and troubleshooting

If pairing does not complete successfully:

* Restarting the daemon may re-enter pairing mode **if no device store exists**
* If pairing still does not appear, a full uninstall and reinstall may be required

For recovery steps and common pairing issues, see:

```
docs/daemon/troubleshooting.md
```

---

## Why pairing is strict

NovaKey treats pairing as a foundational trust operation.

To ensure pairing reflects the user’s **current security posture** and cannot
be weakened over time:

* pairing tokens are time-limited
* secure storage failures are not retried indefinitely
* pairing state is not silently reset
* recovery may require reinstalling the daemon

This strictness is intentional and protects both users and devices.

