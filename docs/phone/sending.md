# Sending Secrets

Secrets can be sent to a paired computer when needed.

## Sending flow
1. Select a secret
2. Choose Send
3. NovaKey authenticates the user
4. Secret is transmitted securely
5. The daemon injects or consumes the secret

## Optional approval
Some systems may require an approval step on the computer before injection.

NovaKey provides both visual and VoiceOver feedback for success or failure.

## How sending works
When you tap **Send**, NovaKey:  
1. Authenticates you (*Face ID / passcode*)  
2. Verifies pairing  
3. Optionally requests approval on the computer (*Two-Man Mode*)  
4. Sends the secret to NovaKey-Daemon for injection  

## Send a secret
1. Tap a secret
2. Tap '**Arm Computer (15s)**' which sends a message to the computer it will receive text to type soon
3. Tap the secret you wish to send 
3. Choose **Send**
4. Confirm biometric prompt
5. Watch for a success message:
   - **Sent to <Computer>**
   - or **📋 Copied to clipboard on <Computer>** (*when injection is blocked*)

> If no Send Target exists, sending is blocked.

## Two-Man Mode (approval)
Some configurations require an explicit approve step on the computer before injection is allowed.

Typical flow:
- iOS app sends **Approve**
- daemon opens a short approval window
- iOS app sends **Inject**
- daemon injects into the focused field; if injection is blocked and policy allows, the daemon may copy the secret to clipboard and report this explicitly 

If approval is required but missing, you’ll see a “*Needs approval*” style error.

## Clipboard Mode Behavior
On some systems (*or when policies deny injection*), the daemon may copy the secret to clipboard instead of typing it.

When this happens, NovaKey treats it as a **successful send**, but indicates it differently:
- “📋 Copied to clipboard on <Computer>”
- optionally showing the daemon’s message if provided

