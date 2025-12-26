# NovaKey Documentation

NovaKey is a secure, post-quantum–protected secret injection system:

- Secrets live **only** on your iPhone (iOS Keychain).
- Secrets are transmitted **on demand** to a trusted computer using mutual authentication,
  replay protection, and modern cryptography.
- The computer runs **NovaKey-Daemon**, which injects into the currently focused field
  (and may fall back to clipboard in constrained environments).

## NovaKey iOS App

![NovaKey main page](assets/screenshots/novakey-main-page.PNG)

## Quick start

1. Install and run NovaKey-Daemon  
   → See **NovaKey-Daemon → Install**

2. Add a Listener in the iOS app  
   → See **Phone App → Pairing**

3. Pair via QR  
   → See **Phone App → Pairing**

4. Add secrets and send  
   → See **Phone App → Secrets** and **Phone App → Sending**

## What NovaKey is (and isn’t)

**NovaKey is:**
- A local-first, explicit “send secret now” tool.
- Opinionated about safety: no silent fallbacks, no cloud dependency.

**NovaKey is not:**
- A password manager UI that displays secrets later.
- A cloud sync service.

## Architecture overview

### iOS App
- Stores secrets in the iOS Keychain
- Requires Face ID / passcode to copy or send
- Never displays secrets after saving

### NovaKey-Daemon
- Runs on your computer
- Accepts secrets only from paired devices
- Optional safety gates:
  - arming (“push-to-type”)
  - two-man approval (approve then inject)
  - target allow/deny policy
- Injects into the active application (clipboard fallback when blocked)

