# Clipboard Handling (Pro Feature)

Clipboard access is available **only in the Pro tier** of NovaKey.
NovaKey allows secrets to be copied to the clipboard when explicitly requested.

This restriction is intentional and designed to reduce the risk of accidental secret exposure via system-wide clipboard access.

## Clipboard behavior (Pro only)
- Secrets may be copied only after explicit user action
- Clipboard contents auto-clear after a configurable timeout
- Clipboard is cleared when the app moves to background (if enabled)
- Clipboard access always requires biometric or device authentication

Free-tier users will see a clear prompt explaining that clipboard access requires the Pro unlock.
Clipboard is local and intentionally constrained.

## Copying a secret
When you copy a secret:
- it is placed on the **local iOS clipboard**
- NovaKey can auto-clear after a configurable timeout
- NovaKey clears clipboard on background (unless disabled)

## Auto-clear timer
Configurable options:
- Never
- 15s / 30s / 60s / 2m / 5m

## Clear Clipboard Now
From the main menu:
- **Clear Clipboard Now** immediately clears any clipboard contents NovaKey owns.

## Universal Clipboard note
NovaKey aims to keep clipboard handling local and predictable.
If you rely on Universal Clipboard across devices, use caution with any secrets.

