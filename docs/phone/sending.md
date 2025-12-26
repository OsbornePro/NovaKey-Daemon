# Sending Secrets to a Computer

## How sending works
When you tap **Send**, NovaKey:
1. Authenticates you (*Face ID / passcode*)  
2. Verifies pairing  
3. Optionally requests approval on the computer (*Two-Man Mode*)  
4. Sends the secret to NovaKey-Daemon for injection  

## Send a secret
1. Tap a secret
2. Choose **Send**
3. Confirm biometric prompt
4. Watch for a success message:
   - **Sent to <Computer>**
   - or **📋 Copied to clipboard on <Computer>** (*when injection is blocked*)

> If no Send Target exists, sending is blocked.

## Two-Man Mode (approval)
Some configurations require an explicit approve step on the computer before injection is allowed.

Typical flow:
- iOS app sends **Approve**
- daemon opens a short approval window
- iOS app sends **Inject**
- daemon injects into the focused field

If approval is required but missing, you’ll see a “Needs approval” style error.

## Clipboard fallback behavior
On some systems (or when policies deny injection), the daemon may copy the secret to clipboard instead of typing it.

When this happens, NovaKey treats it as a **successful send**, but indicates it differently:
- “📋 Copied to clipboard on <Computer>”
- optionally showing the daemon’s message if provided

