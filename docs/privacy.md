# Privacy Policy

**Effective date:** January 2026

NovaKey is designed to minimize data collection and maximize user privacy. This policy explains what data NovaKey does and does not collect, and how data is handled on your devices.

---

## Overview

NovaKey is a local-first application for securely storing and sending secrets between your phone and your own computers.

- NovaKey does **not** require an account
- NovaKey does **not** use cloud storage
- NovaKey does **not** track users
- NovaKey does **not** sell or share data

All sensitive data remains under your control.

---

## Data We Do Not Collect

NovaKey does **not** collect, store, or transmit:

- Names, email addresses, or account information
- Analytics or telemetry
- Advertising identifiers
- Location data
- Usage tracking data
- Crash reports tied to personal identity

NovaKey does not use third-party analytics or advertising SDKs.

---

## Data Stored on Your Device

### Secrets
Secrets you add to NovaKey:
- Are stored **locally on your device**
- Are protected by the iOS Keychain
- Are never displayed again after saving
- Require device authentication (Face ID / Touch ID / passcode) to access

NovaKey cannot read your secrets without your explicit authentication.

### Pairing Information
When you pair a computer with NovaKey:
- Cryptographic pairing material is stored locally
- Pairing data is used only to authenticate your devices
- Pairing data is never shared externally

---

## Clipboard Handling

### Phone Clipboard
When you copy a secret:
- The secret is placed on the **local iOS clipboard**
- Clipboard contents can be configured to auto-clear
- Clipboard may be cleared when the app moves to the background (if enabled)

Clipboard use always requires explicit user action and authentication.

### Computer Clipboard (NovaKey-Daemon)
On supported systems, NovaKey-Daemon may inject a secret into the computer’s clipboard as part of delivery.

NovaKey does not monitor or read clipboard contents beyond the requested operation.

---

## Network Communication

NovaKey communicates only with:
- Computers you explicitly pair
- Over encrypted connections
- Using cryptographic authentication

NovaKey does **not** communicate with NovaKey-controlled servers.

---

## Accessibility Permissions

On some platforms, NovaKey-Daemon may request:
- Accessibility permissions
- Input Monitoring permissions

These permissions are required only to perform explicit actions requested by you (such as securely injecting secrets). NovaKey does not monitor keystrokes or user activity outside of those actions.

---

## Third-Party Services

NovaKey does not integrate with:
- Advertising networks
- Analytics providers
- Data brokers

The app relies only on system-provided services (such as iOS Keychain and biometric authentication).

---

## Children’s Privacy

NovaKey does not knowingly collect personal data from children. The app contains no content targeted specifically at children and does not require personal information to function.

---

## Data Deletion

You can delete your data at any time by:
- Removing secrets from the app
- Removing paired listeners
- Uninstalling the app

Uninstalling NovaKey removes all locally stored data.

---

## Changes to This Policy

If this Privacy Policy changes, the updated version will be published with a new effective date.

---

## Contact

If you have questions about this Privacy Policy or NovaKey’s privacy practices:

- Website: https://osbornepro.com
- Maintainer: Robert Osborne
- Email: security@novakey.app 
