# 📱 Getting Started (Phone → Computer Pairing)

This guide walks you through everything you need to do after installing the **NovaKey phone app**, from installing the desktop companion to sending your first secret.

---

## What you need

* An iPhone with the **NovaKey** app installed
* A computer running **NovaKey-Daemon** (Windows, macOS, or Linux)
* A local network connection (phone and computer on the same network)

NovaKey does **not** use cloud services.
All secrets remain local to your devices.

---

## High-level setup flow

1. Install **NovaKey-Daemon** on your computer
2. Open NovaKey on your phone
3. Add a **Listener** (your computer)
4. Pair the phone and computer
5. Add a secret on your phone
6. Arm the computer and send secrets when needed

This is the fastest path to:

> **Send a secret from phone → computer**

---

## On the phone (quick tour)

### Main screen

![Main screen](../assets/screenshots/novakey-main-page.PNG)

### Add a secret

![New secret](../assets/screenshots/novakey-new-secret.PNG)

### Pair by scanning a QR code

![Scan QR](../assets/screenshots/novakey-scan-qr.PNG)

---

## Step 0 — Install NovaKey-Daemon on your computer

NovaKey-Daemon is the **free desktop companion** that receives secrets from your phone.

### ✅ Recommended: Native installers (modern)

These installers are the **supported and recommended** way to install NovaKey-Daemon.

Download the installer that matches your system from:

```
https://downloads.novakey.app/Installers/<file name>
```
See the "Project Links" in the left menu pane for the latest download links.  

Or from GitHub Releases:

```
https://github.com/OsbornePro/NovaKey-Daemon/releases
```

### Platform notes

#### Windows

* Download **NovaKey-Setup.exe**
* Double-click and follow the installer
* No admin privileges required
* Creates a per-user Scheduled Task
* Starts automatically at login

#### macOS

* Download the `.pkg` that matches your Mac:

  * Apple Silicon (M1/M2/M3): `*-arm64.pkg`
  * Intel: `*-amd64.pkg`
* Double-click and follow the installer
* Grants permissions under:

  * **Accessibility**
  * **Input Monitoring**
* Registers a LaunchAgent and starts automatically

#### Linux

* Download `.deb` or `.rpm` package
* Install using your package manager
* Runs as a **systemd user service**
* Starts automatically when you log in

---

### ⚠️ Legacy installation (deprecated)

The original shell and PowerShell install scripts still exist for advanced or automated environments but are **no longer recommended for end users**.

They live under:

```
installers/legacy/
```

They may be removed in a future release.

---

## Verify the daemon is running

The daemon must be listening on port `60768`.

**Windows**

```powershell
Get-NetTcpConnection -State Listen -LocalPort 60768
```

**Linux**

```bash
ss -tunlp | grep 60768
```

**macOS**

```bash
lsof -i :60768
```

If the daemon is listening, you’re ready to pair.

---

## Step 1 — Add a Listener (Phone)

1. Open the **NovaKey** app
2. Tap **Listeners** (antenna icon)
3. Tap **Add Listener**
4. Enter:

   * **Name:** e.g. “My Desktop”
   * **Host/IP:** your computer’s LAN IP or hostname
   * **Port:** `60768`
5. Enable **Make Send Target**
6. Tap **Add**

> ⚠️ A Send Target must be selected to pair or send secrets.

---

## Step 2 — Pair the phone and computer (QR code)

### On your phone

1. Go to **Listeners**
2. Select your listener (or swipe right)
3. Tap **Pair**
4. Tap **Scan QR Code**
5. Allow camera access if prompted

---

### On your computer

1. Ensure NovaKey-Daemon is running
2. If no devices are paired yet:

   * The daemon enters pairing mode
   * A **time-limited QR code** is generated and displayed
3. Scan the QR code with your phone

---

### Complete pairing

1. Verify the device information on your phone
2. Tap **Pair**
3. Allow local network access when prompted

When pairing succeeds, the listener will show **Paired**.

---

## Step 3 — Add your first secret

1. Tap **+**
2. Enter a label and secret
3. Confirm and tap **Save**

NovaKey intentionally never displays secrets again after saving.

---

## Step 4 — Send a secret

1. Tap the secret
2. Tap **Arm Computer**
3. Tap **Send**
4. Authenticate with Face ID or device passcode

Possible outcomes:

* **Sent to <Computer>** — typed injection
* **📋 Copied to clipboard on <Computer>** — typing blocked, clipboard used

Both indicate a successful send.

---

## If something doesn’t work

Start with:

* **Phone App → Troubleshooting**
* **NovaKey-Daemon → Troubleshooting**

Most issues are caused by:

* Phone and computer not on the same network
* Firewall blocking port `60768`
* Daemon bound to `127.0.0.1` instead of a LAN address
* Pairing window missed (restart daemon if needed)

