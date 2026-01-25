# 📱 Getting Started (Phone → Computer Pairing)

This guide walks you through everything you need to do after installing the **NovaKey phone app**, from installing the desktop companion to sending your first secret.

---

## What you need

* An Phone with the **NovaKey** app installed
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

## Step 0 — Install NovaKey-Daemon on your computer

NovaKey-Daemon is the **free desktop companion** that receives secrets from your phone.

### ✅ Recommended: Native installers (modern)

These installers are the **supported and recommended** way to install NovaKey-Daemon.

Download the installer that matches your system from:

```
https://downloads.novakey.app/Installers/<file name>
```
See the [Project Links](https://novakey.app/en/latest/links/) in the left menu pane for the latest download links.  

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
![NovaKey-iOS-App Go to Listeners](https://novakey.app/en/latest/assets/screenshots/Go-To-Listeners.png)
3. Tap **Add Listener**
4. Enter:

   * **Name:** e.g. “My Desktop”
   * **Host/IP:** your computer’s LAN IP or hostname
   * **Port:** `60768`
5. Enable **Make Send Target**
6. Tap **Add**
![NovaKey-iOS-App Add Listener](https://novakey.app/en/latest/assets/screenshots/Fill-In-Listener-Info.png)

You will now see your listener added.
![NovaKey-iOS-App shows Listener Added](https://novakey.app/en/latest/assets/screenshots/You-Will-See-Listener-Added.png)

> ⚠️  A Send Target must be selected to pair or send secrets.

---

## Step 2 — Pair the phone and computer (QR code)

### On your phone

1. Go to **Listeners**
![NovaKey-iOS-App Go to Listeners](https://novakey.app/en/latest/assets/screenshots/Go-To-Listeners.png)
2. Select your listener (*or swipe right*)
3. Tap **Pair**
![NovaKey-iOS-App Pair](https://novakey.app/en/latest/assets/screenshots/novakey-swipe-pair-send-listener.PNG)
4. Tap **Scan QR Code**
![NovaKey-iOS-App Scan QR](https://novakey.app/en/latest/assets/screenshots/novakey-scan-qr.PNG)
5. Allow camera access if prompted
![NovaKey-iOS-App Approve Camera Access](https://novakey.app/en/latest/assets/screenshots/Approve-Camera-Access.png)

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
![NovaKey-iOS-App Verify Pairing](https://novakey.app/en/latest/assets/screenshots/Verify-IP-Port-Select-Pair.png)
3. Allow local network access when prompted

When pairing succeeds, the listener will show **Paired**.
![NovaKey-iOS-App Allow NovaKey to find devices on the local network](https://novakey.app/en/latest/assets/screenshots/Allow-NovaKey-Local-Net-Access.png)

---

## Step 3 — Add your first secret

1. Tap **+**
2. Enter a label and secret
![NovaKey-iOS-App New Secret](https://novakey.app/en/latest/assets/screenshots/novakey-new-secret.PNG)
3. Confirm and tap **Save**

NovaKey intentionally never displays secrets again after saving.

---

## Step 4 — Send a secret

1. Tap the secret
2. Tap **Arm Computer**
![NovaKey-iOS-App Arm Computer](https://novakey.app/en/latest/assets/screenshots/Select-Arm-Computer.png)  
![NovaKey-iOS-App Successful Arm Message](https://novakey.app/en/latest/assets/screenshots/Successful-Arm-Message.png)  
3. Tap **Send**
![NovaKey-iOS-App Send Secret](https://novakey.app/en/latest/assets/screenshots/Select-Send-Secret.png)
![NovaKey-iOS-App Successful Sent Secret](https://novakey.app/en/latest/assets/screenshots/Successfully-Inject-Secret-On-Remote-Device.png)  
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

