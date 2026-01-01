# Clipboard Handling

NovaKey supports clipboard usage in two places:

## Phone Clipboard (iOS)
You can copy a secret to the clipboard on the phone after explicit user action.
- Clipboard contents auto-clear after a configurable timeout (if enabled)
- Clipboard may be cleared when the app moves to background (if enabled)
- Copying requires user authentication

## Computer Clipboard (NovaKey-Daemon)
On supported systems, NovaKey-Daemon may place the secret into the computer’s clipboard as part of delivery.

Clipboard support is available in both Free and Pro tiers.

## Clipboard behavior
- Secrets may be copied only after explicit user action
- Clipboard contents auto-clear after a configurable timeout
- Clipboard is cleared when the app moves to background (if enabled)
- Clipboard access always requires biometric or device authentication

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

