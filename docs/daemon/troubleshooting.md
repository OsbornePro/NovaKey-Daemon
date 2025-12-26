# NovaKey-Daemon Troubleshooting

## “Nothing types” / injection fails
### Linux
- Wayland may block injection depending on compositor and security settings.
- Try X11, or rely on clipboard fallback where appropriate.

### macOS
- Accessibility permissions are often required for injection.
- Confirm the daemon has the required permissions.

### Windows
- Some UAC contexts and secure desktop prompts can block injection.

## Clipboard fallback happened (status okClipboard)
This means:
- injection was blocked or denied
- **but** the daemon successfully copied the secret to clipboard

Common causes:
- focus target denied by target policy
- OS permissions missing
- secure input mode enabled by the active app
- Wayland / compositor restrictions

## Not paired / pairing doesn’t show QR
- If a device store exists, the daemon may not regenerate a QR.
- Confirm your intended re-pair workflow.
- Ensure the phone is targeting the correct host/port.

## “Not armed”
- Arm gate is enabled and active.
- Trigger arming locally (or via the Arm API if enabled and loopback-only).

## “Needs approval”
- Two-Man Mode is enabled and injection requires an approve step inside an approval window.

## Timestamp / replay errors
- Ensure system clocks are correct.
- Replay detection can trigger if the same request is re-sent.

## Logs
- Logs may be redacted but should still be treated as sensitive.
- If sharing logs for support, share only what’s necessary.

