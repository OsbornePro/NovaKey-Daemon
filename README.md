# PassLink – Quantum‑Resistant BLE Bridge for Password‑Manager Secrets  
**What is PassLink?**  
*PassLink is a one‑tap, post‑quantum‑secure agent that runs as a background service, waits for a phone to push a Kyber‑768 encrypted password/MFA payload over BLE, decrypts it, and auto‑types that secret into a selected text box.*  

**Why would I need this?**  
*Even with a password manager you still have to remember at least one master password, the one that unlocks the vault. 
That password often becomes the weakest link because it’s either memorised or stored insecurely.  
PassLink lets you store a strong, high‑entropy master password on your phone and retrieve it with a single tap via a secure, post‑quantum BLE connection. 
The desktop agent automatically types the secret for you, so you never have to recall or manually enter that critical password again. 
In short, it gives you the security of a truly strong master password without the burden of remembering it.*

---  

<div align="center">

[![GitHub release (latest by date)](https://img.shields.io/github/v/release/yourorg/passlink?label=release)](https://github.com/yourorg/passlink/releases)  
[![Go Report Card](https://goreportcard.com/badge/github.com/yourorg/passlink)](https://goreportcard.com/report/github.com/yourorg/passlink)  
[![License: Commercial](https://img.shields.io/badge/license-Commercial-blue.svg)](./LICENSE.txt)  

</div>  

---  

## Table of Contents
1. [Overview](#overview)  
2. [Features](#features)  
3. [Architecture diagram](#architecture-diagram)  
4. [Prerequisites](#prerequisites)  
5. [Installation](#installation)  
   - [Windows (service)](#windows-service)  
   - [macOS / Linux (systemd / launchd)](#macos--linux-daemon)  
6. [Building from source](#building-from-source)  
7. [Running the agent](#running-the-agent)  
8. [Configuration](#configuration)  
9. [Troubleshooting](#troubleshooting)  
10. [Contributing](#contributing)  
11. [License](#license)  
12. [Contact & support](#contact--support)  

---  

## Overview  

PassLink is a **stand‑alone BLE peripheral** that sits on a workstation (*Windows, macOS, or Linux*).  
* The **phone app** (*your existing Lumo/PassLink mobile client*) acts as a BLE **central**.  
* When the phone discovers the peripheral, it **writes** a single BLE characteristic containing:  
```[Kyber‑768 ciphertext] || [XChaCha20‑Poly1305 encrypted payload]```
* The peripheral **decapsulates** the Kyber ciphertext, derives a 256‑bit session key, **decrypts** the payload, and **auto‑types** the password/MFA code into whatever window currently has focus.  

All cryptographic operations are **post‑quantum‑resistant** (*Kyber‑768 is a NIST‑selected KEM*). No plaintext travels over the air, and the desktop never contacts any external server.

---  

## Features  

| Check Box | Feature |
|----|----------|
| ✅ | **Quantum‑resistant key exchange** – Kyber‑768 (*NIST‑selected*). |
| ✅ | **Authenticated encryption** – XChaCha20‑Poly1305 (*AEAD*). |
| ✅ | **BLE peripheral** (*advertises a custom GATT service*). |
| ✅ | **Zero‑knowledge** – the desktop never learns the phone’s public key; only the derived session key exists in RAM. |
| ✅ | **Auto‑type** via `robotgo` (*human‑like keystroke pacing*). |
| ✅ | **Runs as a background service** on Windows, macOS (*launchd*) and Linux (*systemd*). |
| ✅ | **Configurable** – enable/disable auto‑type, adjust cooldown, change BLE advertisement name. |
| ✅ | **Secure storage** – the desktop’s Kyber public key is persisted in the OS key‑ring; the private key is generated at service start and zeroed on shutdown. |
| ✅ | **Extensible** – the code is deliberately modular (BLE, crypto, UI) for easy future enhancements. |

---  

## Architecture diagram

```
+---------------------------+                               +---------------------------+
| 📱 Phone (Central)        |                               | 💻 Desktop Service        |
|                           |                               | (Peripheral)              |
| 1️⃣ Generate Kyber        |                               | 1️⃣ Advertise GATT service|
|    ciphertext            |                               |    & characteristic       |
| 2️⃣ Encrypt secret        |                               |                           |
|    (XChaCha20‑Poly1305)  |                               | 2️⃣ Wait for BLE write    |
| 3️⃣ Write payload to      |                               |                           |
|    UnlockRequest char    |                               | 3️⃣ Receive payload       |
+------------|--------------+                               |    (Kyber ct + AEAD)      |
             | BLE (Write)                                 |                           |
             v                                            | 4️⃣ Decapsulate Kyber →   |
+---------------------------+                               |    derive session key     |
|  Desktop receives payload |                               |                           |
|  (Kyber ct + encrypted)   |                               | 5️⃣ Decrypt secret with   |
+------------|--------------+                               |    XChaCha20‑Poly1305     |
             |                                            |                           |
             v                                            | 6️⃣ Auto‑type secret into |
+---------------------------+                               |    focused window         |
| 4️⃣ Decapsulate & derive  |                               |                           |
|    session key            |                               | 7️⃣ (Optional) Send ACK   |
+------------|--------------+                               +------------|--------------+
             |                                                   |
             v                                                   v
+---------------------------+                               +---------------------------+
| 5️⃣ Decrypt secret        |                               | 6️⃣ Secret typed into UI  |
+---------------------------+                               +---------------------------+
Underlying crypto: Kyber‑768 → XChaCha20‑Poly1305
```

---

## Prerequisites  

| Platform | Required software |
|----------|-------------------|
| **Windows 10+ (64‑bit)** | • Go ≥ 1.22 (for building) <br>• Bluetooth LE adapter (built‑in on most laptops) |
| **macOS 12+** | • Xcode command‑line tools (`xcode-select --install`) <br>• Bluetooth LE (built‑in) |
| **Linux (Ubuntu 22.04+, Fedora, Arch, etc.)** | • BlueZ ≥ 5.50 <br>• `libbluetooth-dev` (Debian/Ubuntu) or equivalent <br>• Bluetooth LE adapter (most modern laptops) |
| **All** | • Git <br>• Access to a terminal / PowerShell <br>• Administrator / sudo privileges (to install the service) |

---  

## Installation  

### Windows – Service  

1. **Download the latest release**  

   ```powershell
   Invoke-WebRequest -UseBasicParsing -Uri "https://github.com/OsbornePro/PassLink/releases/latest/download/passlink-windows-amd64.zip" -OutFile "$env:USERPROFILE\Downloads\passlink.zip"
   Expand-Archive -Force "$env:USERPROFILE\Downloads\passlink.zip" -DestinationPath "$env:ProgramFiles\PassLink"
   # You can also use tar. Expand-Archive is known to have issues
   tar -xf $env:USERPROFILE\Downloads\passlink.zip -C $env:ProgramFiles\PassLink

2. Install the service (requires admin rights)
   ```powershell
   cd $env:ProgramFiles\PassLink
   .\passlink.exe install
   .\passlink.exe start
   ```
   
The service will now advertise the BLE service 0000c0de‑0000‑1000‑8000‑00805f9b34fb under the name PassLinkAgent.

3. Verify it is running
   ```powershell
   Get-Service PassLink
   # or
   sc query PassLink
   ```

4. Stop / Uninstall (*if you ever need to*)
   ```powershell
   .\passlink.exe stop
   .\passlink.exe uninstall
   ```

### Linux / Unix / OpenBSD – Daemon
**macOS**
   ```bash
# 1. Install binary
sudo mkdir -p /Library/PrivilegedHelperTools/com.passlink.agent
sudo cp passlink-macos-amd64 /Library/PrivilegedHelperTools/com.passlink.agent/passlink
sudo chmod 755 /Library/PrivilegedHelperTools/com.passlink.agent/passlink

# 2. Code-sign with required Bluetooth entitlement
sudo codesign --remove-signature "/Library/PrivilegedHelperTools/com.passlink.agent/passlink" 2>/dev/null || true
sudo /usr/bin/codesign --force --options runtime \
     --entitlements - \
     --sign - \
     "/Library/PrivilegedHelperTools/com.passlink.agent/passlink" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>com.apple.security.device.bluetooth</key><true/>
    <key>com.apple.security.cs.allow-jit</key><true/>
    <key>com.apple.security.cs.allow-unsigned-executable-memory</key><true/>
</dict>
</plist>
EOF

# 3. Install the daemon plist
```bash
cat <<EOF | sudo tee /Library/LaunchDaemons/com.passlink.agent.plist
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.passlink.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Library/PrivilegedHelperTools/com.passlink.agent/passlink</string>
    </array>
    <key>MachServices</key>
    <dict>
        <key>com.passlink.agent</key><true/>
    </dict>
    <key>KeepAlive</key><true/>
    <key>RunAtLoad</key><true/>

    <!-- Use unified logging instead of files -->
    <key>StandardOutPath</key><string>/var/log/com.passlink.agent.stdout.log</string>
    <key>StandardErrorPath</key><string>/var/log/com.passlink.agent.stderr.log</string>

    <!-- Apple-recommended hardening -->
    <key>EnablePressuredExit</key><false/>
    <key>EnableTransactions</key><false/>
</dict>
</plist>
EOF

# 4. Register and start (macOS 13+ preferred way)
sudo /System/Library/Frameworks/ServiceManagement.framework/Versions/A/Resources/SMAppService daemon register \
    /Library/LaunchDaemons/com.passlink.agent.plist
# Fallback for older macOS versions
# sudo launchctl load -w /Library/LaunchDaemons/com.passlink.agent.plist

# Start it
sudo launchctl bootstrap system /Library/LaunchDaemons/com.passlink.agent.plist

# Or simply:
sudo launchctl load -w /Library/LaunchDaemons/com.passlink.agent.plist

# 5. View logs
log show --predicate 'process == "passlink"' --last 15m --info --debug
# or tail the files
tail -f /var/log/com.passlink.agent.{stdout,stderr}.log
```

**Linux (systemd)**

```bash
# 1. Install binary
sudo mkdir -p /opt/passlink
sudo cp passlink-linux-amd64 /opt/passlink/passlink
sudo chmod 755 /opt/passlink/passlink

# 2. Create dedicated unprivileged user
sudo useradd --system --no-create-home --user-group passlink || true

# 3. systemd unit
sudo tee /etc/systemd/system/passlink.service > /dev/null <<EOF
[Unit]
Description=PassLink BLE Agent
After=bluetooth.target
Wants=bluetooth.target

[Service]
ExecStart=/opt/passlink/passlink
Restart=always
RestartSec=5
User=passlink
Group=passlink
SupplementaryGroups=bluetooth
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
NoNewPrivileges=true
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# 4. Enable and start
sudo systemctl daemon-reload
sudo systemctl enable --now passlink.service

# 5. Check status & logs
sudo systemctl status passlink.service
journalctl -u passlink.service -f
```

---  

## Building from source

If you prefer to compile the agent yourself (or want to contribute), follow these steps:

```bash
# 1. Clone and enter the repo
git clone https://github.com/OsbornePro/PassLink.git
cd PassLink

# 2. Make sure you have Go 1.22 or newer
go version   # → should say go1.22 or higher

# 3. Download dependencies
go mod tidy

# Windows (amd64)
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o passlink.exe ./cmd/passlink

# macOS Intel (amd64)
GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o passlink-macos-amd64 ./cmd/passlink

# macOS Apple Silicon (arm64) – recommended for modern Macs
GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o passlink-macos-arm64 ./cmd/passlink

# Linux (amd64)
GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o passlink-linux-amd64 ./cmd/passlink

# Linux (arm64) – Raspberry Pi 4/5, modern servers, etc.
GOOS=linux   GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o passlink-linux-arm64 ./cmd/passlink
```

After building on macOS you need to sign it (required for Bluetooth)
```bash
# Ad-hoc signing (works without paid Apple Developer account)
codesign --remove-signature passlink-macos-* 2>/dev/null || true
codesign --force --options runtime --sign - \
  --entitlements - ./passlink-macos-* <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>com.apple.security.device.bluetooth</key><true/>
    <key>com.apple.security.cs.allow-jit</key><true/>
    <key>com.apple.security.cs.allow-unsigned-executable-memory</key><true/>
</dict>
</plist>
EOF
```

One-liner to build every platform
```bash
goreleaser release --snapshot --clean   # if you ever adopt GoReleaser (highly recommended)
# or manually:
for os in windows darwin linux; do
  for arch in amd64 arm64; do
    ext=""; [ "$os" = "windows" ] && ext=".exe"
    GOOS=$os GOARCH=$arch go build -trimpath -ldflags="-s -w" \
      -o "passlink-$$ {os}- $${arch}${ext}" ./cmd/passlink
  done
done
# then sign the two macOS binaries as shown above
```

The resulting binary is ready to be installed as a service (*see the Installation section*).

---  

## Running the agent

When the service is up, you should see a BLE advertisement named PassLinkAgent (*or whatever you set in BLEAdvertiseName*).

1. Open the companion phone app (*the Lumo/PassLink mobile client*).
2. The app scans for the service UUID `0000c0de‑0000‑1000‑8000‑00805f9b34fb`.
3. Tap "*Unlock*" in the app – the phone encrypts the master password + TOTP seed, writes the payload to the characteristic `0000c0df‑0000‑1000‑8000‑00805f9b34fb`.
4. PassLink receives the data, decapsulates, decrypts, and auto‑types the secret into the currently focused window (*e.g., the password field of your password manager*).

You’ll see a short toast (*Windows*) or a notification (*macOS/Linux*) confirming success, and a log entry in the service log.

---

## Configuration
All runtime options are exposed via environment variables. 
They can be set in the service definition (Windows `sc config`, systemd unit `Environment=` line, or launchd plist `<key>EnvironmentVariables</key>`).

| Variable                     | Default                | Description                                                                                              |
|------------------------------|------------------------|----------------------------------------------------------------------------------------------------------|
| `PASSLINK_ADVERTISE_NAME`   | `PassLinkAgent`        | BLE local name shown to phones.                                                                          |
| `PASSLINK_AUTO_TYPE`        | `true`                 | `true` → auto‑type the secret; `false` → only log it.                                                    |
| `PASSLINK_COOLDOWN_SECONDS` | `2`                    | Minimum seconds to wait after a successful unlock before accepting another request.                       |
| `PASSLINK_LOG_LEVEL`        | `info`                 | Logging verbosity – `debug`, `info`, `warn`, `error`.                                                    |
| `PASSLINK_KEYRING_SERVICE`  | `PassLink`             | Identifier used for the OS key‑ring entry that stores the public key.                                    |
| `PASSLINK_KEYRING_USER`     | `clientKyberPublicKey` | Username for the key‑ring entry.                                                                        |

Example (systemd unit)
```
Environment="PASSLINK_ADVERTISE_NAME=MyOfficePassLink"
Environment="PASSLINK_AUTO_TYPE=false"
Environment="PASSLINK_LOG_LEVEL=debug"
```

---  

## Troubleshooting
| Symptom                         | Likely cause                                                                                                 | Fix                                                                                                                                                                                                 |
|---------------------------------|--------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------|
| No BLE advertisement appears    | Bluetooth adapter disabled or driver missing                                                              | Enable Bluetooth in OS settings; on Linux ensure `bluetooth.service` is running (`sudo systemctl start bluetooth`).       |
| Phone can’t find the service    | Wrong UUID or the service isn’t advertising                                                                | Verify the service UUID in the source (`serviceUUID`). Re‑install the service to reload the binary.                      |
| Auto‑type does nothing          | `PASSLINK_AUTO_TYPE` set to `false` **or** the active window blocks synthetic keystrokes (e.g., admin apps) | Set `PASSLINK_AUTO_TYPE=true`. Run the binary interactively (`passlink.exe run`) to see debug logs.                     |
| “Decapsulation failed” error   | Mismatch between the phone’s public key and the stored desktop public key | Delete the persisted key‑ring entry (`keyring.Delete("PassLink","clientKyberPublicKey")`) and restart the service – a new key pair will be generated.      |
| Service crashes on startup (Windows) | Missing Visual C++ Redistributable (required by `robotgo`)                                                | Install the latest **Microsoft Visual C++ Redistributable** (x64).                                                   |
| Logs are empty                  | Service started with `PASSLINK_LOG_LEVEL=error` and no errors occurred                                      | Change to `debug` or `info` to see more output (`PASSLINK_LOG_LEVEL=debug`).                                            |

Logs are written to:
| OS      | Log location |
|---------|--------------|
| **Windows** | Event Viewer → **Applications and Services Logs → PassLink** |
| **macOS**   | `/var/log/passlink.out` and `/var/log/passlink.err` (*as defined in the launchd plist*) |
| **Linux**   | `journalctl -u passlink.service` |

---  

## Contributing
We welcome contributions! Please follow these steps:

1. Fork the repository and create a feature branch (`git checkout -b feat/your‑feature`).
2. Write tests – the project uses Go’s standard testing package. Run `go test ./...` locally.**
3. Run linters – we use `golangci-lint`. Install with `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` and run `golangci-lint run`.
4. Update documentation – if you add a new flag or change behaviour, update the README.md and/or the EULA.md.
5. Submit a Pull Request – link any related issue, and ensure CI passes.

*Note: All contributions are accepted under the same commercial licence (the contributor assigns the rights to OsbornePro LLC). By submitting a PR you agree to this arrangement.*

---  

## License
PassLink is **proprietary commercial software**. See the full terms in `EULA.md`.
The source code in this repository is provided **as‑is** for the purpose of building the binary; redistribution of the source or compiled binaries is prohibited without a separate written licence from OsbornePro LLC.

---

## Contact & Support

* Product website / purchase – [https://passlink.com](https://passlink.com)
* Technical support – [support@passlink.com](mailto:support@passlink.com)
* Security disclosures – security@passlink.com (PGP fingerprint: 0xDEADBEEF…)
* GitHub issues – open a ticket in the Issues tab for bugs, feature requests, or installation help.

---
