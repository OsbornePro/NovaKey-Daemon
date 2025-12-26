# Phone App Troubleshooting

## In-app help screens

![Help 1](../assets/screenshots/novakey-help-1.PNG)
![Help 2](../assets/screenshots/novakey-help-2.PNG)

## About screen

![About](../assets/screenshots/novakey-about.PNG)

## “No Send Target set”
You need a default Listener:
- Listeners → select a Listener → set **Make Send Target**

## “Not paired”
- Listeners → select the Listener → **Re-pair**

## “Computer isn’t armed” / “Not armed”
Your daemon may require arming (“push-to-type”).
- Arm the daemon on the computer
- Then try again

## “Needs approval”
Two-Man Mode is enabled:
- Approve on the computer (or let the iOS app auto-approve if enabled)
- Then send again

## “Copied to clipboard on computer”
Injection was blocked or denied, but the daemon successfully copied the secret.
Common reasons:
- OS injection permissions missing
- Wayland / secure input blocks typing
- target policy denies the focused app/window

See **NovaKey-Daemon → Troubleshooting**.

## “Clock check failed” / timestamp errors
NovaKey uses freshness/replay protection.
- Ensure your phone and computer clocks are correct (auto time recommended).

## Still stuck?
Collect:
- your Listener host/port
- whether you’re on LAN or local-only
- daemon logs (treat as sensitive)
- the exact error toast text

Then check:
- **Daemon → Troubleshooting**
- **FAQ**

