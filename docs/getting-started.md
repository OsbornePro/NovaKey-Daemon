# Getting Started

**If you purchased the phone app and you can't get it to work please reach out so I can help.**  
[Email Me](mailto:security@novakey.app)  
  

## What you need
- An iPhone running the NovaKey iOS app
- A computer running NovaKey-Daemon
- A local network connection (*or trusted remote connection*)


### Download from the App Store 
![NovaKey-iOS-App Download](assets/screenshots/novakey-ios-qr.png)

## High-level flow
1. Install NovaKey-Daemon on your computer
2. Install NovaKey on your iPhone
3. Pair the phone with the computer
4. Add secrets on the phone
5. Send secrets securely when needed

NovaKey does not sync secrets via cloud services. All data remains local to your devices.  
  
This is the fastest path to “send a secret from iPhone → computer”.

## On the phone (quick tour)

### Main screen
![Main screen](assets/screenshots/novakey-main-page.PNG)

### Add a secret
![New secret](assets/screenshots/novakey-new-secret.PNG)

### Pair by scanning a QR
![Scan QR](assets/screenshots/novakey-scan-qr.PNG)

## Step 0 — Install NovaKey-Daemon

Follow: **NovaKey-Daemon → [Install](https://novakey.app/en/latest/daemon/install/)** and complete the installer for your platform.  
The daemon will start automatically after installation.

**If you want phone → computer over Wi-Fi:** the daemon must listen on a LAN-reachable address (*not* `127.0.0.1`).



## Step 1 — Open NovaKey and add a Listener

1. Open **NovaKey**
2. Tap **Listeners** (*antenna icon*)
3. Add Listener:
   - Name: “My Desktop”
   - Host/IP: your computer’s LAN IP/hostname
   - Port: `60768`
4. Turn on **Make Send Target**
5. Tap **Add**

Yes — it’s mostly clear, but a few small edits will make it **much clearer, more accurate, and more professional**, especially for Windows/macOS users and troubleshooting.

Here’s a polished version with improved clarity and correctness:

---

## Step 2 — Pair via QR

1. On your computer, start NovaKey-Daemon. 
Note this will happen automatically for you after you run the NovaKey-Daemon installer. 
If you are doing things manually for whatever reason you can use these methods:

   * **Linux:** Open a terminal and run

     ```bash
     systemctl --user start novakey.service
     ```

   * **Windows:** Open **Task Scheduler** and run the task named **NovaKey**

   * **macOS:** Open a terminal and run

     ```bash
     launchctl kickstart -k gui/$(id -u)/com.osbornepro.novakey
     ```
2. If there are no paired devices, the daemon enters pairing mode and generates a **time-limited pairing QR**.

> **TROUBLESHOOTING NOTE:**
> The pairing QR token may expire if not scanned quickly enough.
> If this happens:
>
> * Delete the generated files: `novakey-pair.png`, `server_keys.json`, and (if present) `devices.json`
> * Stop the NovaKey-Daemon
> * Start it again
>
> On Windows this is done via **Task Scheduler**.
> On Linux and macOS, this can be done from the terminal.


3. On iOS:

   * **Listeners → select your listener → Pair → Scan QR**

4. Scan the QR.

You will see a prompt on your phone asking you to confirm the device you are pairing with.  
Take note of the IP address shown.  
  
It is strongly recommended to set your device’s default IP address in NovaKey-Daemon’s `server_config.yaml` file for reliability.  
  
Using a VPN or having virtual interfaces from a hypervisor will not impact NovaKey’s functionality.  
  
If you miss the QR code, restart the daemon or see the NovaKey-Daemon troubleshooting guide.

## Step 3 — Add a secret

1. Tap **+**
2. Enter a label + secret + confirm
3. Tap **Save**

NovaKey will never display the secret again (by design).

## Step 4 — Send it

1. Tap the secret
2. Tap **Send**
3. Authenticate with Face ID / passcode

Success outcomes:
- ✅ **Sent to <Computer>** (typed injection)
- ✅ **📋 Copied to clipboard on <Computer>** (injection blocked; clipboard mode used)

## If something doesn’t work

Start with:  
- **Phone App → Troubleshooting**  
- **NovaKey-Daemon → Troubleshooting**  

