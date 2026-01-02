# NovaKey Apps

NovaKey is a secure secret delivery system designed to send sensitive data from your phone to a trusted computer **without exposing secrets on the screen**.

NovaKey consists of:
- **NovaKey Phone App** – stores secrets locally and sends them securely
- **NovaKey-Daemon** – runs on the target computer and receives secrets

NovaKey is designed with:
- Strong cryptography
- Minimal attack surface
- Accessibility-first UI
- Clear free vs pro feature boundaries

Secrets are never displayed after saving and are only transmitted to explicitly paired devices.

NovaKey is a secure, post-quantum–protected secret injection system:

- Secrets live **only** on your phone (*iOS Keychain or KeyStore*).
- Secrets are transmitted **on demand** to a trusted computer using mutual authentication,
  replay protection, and modern cryptography.
- The computer runs **NovaKey-Daemon**, which injects into the currently focused field
  (*and may fall back to clipboard in constrained environments*).

## NovaKey PHone App

![NovaKey main page](assets/screenshots/novakey-main-page.PNG)

## Quick start

1. Install and run NovaKey-Daemon  
   → See **NovaKey-Daemon → Install**

2. Add a Listener in the NovaKey Phone app  
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
- Stores secrets in the iOS Keychain or Android KeyStore
- Requires Face ID / passcode to copy or send
- Never displays secrets after saving

### NovaKey-Daemon
- Runs on your computer
- Accepts secrets only from paired devices
- Optional safety gates:
  - arming (*“push-to-type”*)
  - two-man approval (*approve then inject*)
  - target allow/deny policy
- Injects into the active application (*injects into the focused field; if injection is blocked and policy allows, the daemon may copy the secret to clipboard and report this explicitly*)

