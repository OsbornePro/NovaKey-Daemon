# FAQ

## Is NovaKey accessible?
Yes. NovaKey supports:
- VoiceOver
- Voice Control
- Dynamic Type
- Reduced Motion
- Non-color-only indicators

## Does NovaKey sync secrets?
No. Secrets remain local to your devices.

## What does Pro unlock?
The Pro unlock is a one-time, non-consumable purchase that:
- Removes the 1-listener limit
- Removes the 1-secret limit
- Enables remote clipboard injection on paired computers

## Is Pro a subscription?
No. Pro is a one-time purchase.

## Why can’t I view secrets after saving?
Because the iOS app is designed to avoid “screen exposure” of secrets after capture. Secrets live in Keychain; you can copy or send them with authentication.

## Why can’t I edit the host/IP after pairing?
Pairing keys are bound to the server address to prevent redirection attacks.
Create a new Listener and re-pair if the address changes.

## Why does it say “Copied to clipboard on computer”?
The daemon could not inject into the focused field, but succeeded at copying to clipboard.
See **Daemon → Troubleshooting**.

## Do I need LAN listening?
Only if your phone must connect to the daemon over Wi-Fi. Loopback-only is safest but not reachable from the phone.

## What’s Two-Man Mode?
An approval step is required before injection is allowed (short time window). It’s designed to reduce accidental or unauthorized injection.

