# Clipboard Behavior (iOS)

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

