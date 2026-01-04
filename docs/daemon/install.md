# Installing NovaKey Daemon

The **NovaKey Daemon** is a background service that runs on your computer and securely receives secrets from the NovaKey app, then types them into the active application.

As of **v1.0**, NovaKey provides **native installers** for all supported platforms.
These installers are the **recommended and supported installation method**.

> ⚠️ **Important: Pairing is security-sensitive**
>
> During first install, the NovaKey Daemon performs a **one-time secure bootstrap** and enters pairing mode.
> A **time-limited QR code** is displayed for pairing your NovaKey app.
>
> If pairing is interrupted, cancelled, or secure storage initialization fails
> (for example due to keyring, DPAPI, or hardware-backed authentication constraints),
> the daemon may fall back to a local device store or require a **full reinstall to restart pairing**.
>
> For best results:
> - Have the NovaKey app ready before installing
> - Complete pairing when the QR code is displayed

Missing the QR code does not compromise security, but may require restarting or reinstalling the daemon to re-enter pairing mode.

---

## Supported Platforms

* **Windows 11**
* **macOS 14+**
* **Linux** (systemd user services, glibc)

---

## Recommended Installation (Installers)

### Download Locations

Installers are available from multiple sources:

* **Windows Primary AMD64:**
  `https://downloads.novakey.app/Installers/NovaKey-Setup.exe`
* **RHEL Based Linux Primary AMD64:**
  `https://downloads.novakey.app/Installers/novakey-1.0.0-1.amd64.rpm`
* **RHEL Based Linux Primary ARM64:**
  `https://downloads.novakey.app/Installers/novakey-1.0.0-1.aarch64.rpm`
* **Debian Based Linux Primary AMD64:**
  `https://downloads.novakey.app/Installers/novakey-1.0.0.amd64.deb`
* **Debian Based Linux Primary ARM64:**
  `https://downloads.novakey.app/Installers/novakey-1.0.0.arm64.deb`
* **MacOS Primary AMD64:**
  `https://downloads.novakey.app/Installers/NovaKey-1.0.0-amd64.pkg`
* **MacOS Primary ARM64:**
  `https://downloads.novakey.app/Installers/NovaKey-1.0.0-arm64.pkg`
* **GitHub Releases:**
  [https://github.com/OsbornePro/NovaKey-Daemon/releases](https://github.com/OsbornePro/NovaKey-Daemon/releases)

Choose the file that matches your operating system and CPU architecture.

---

## Windows Installation

### 1) Download

Download:

```
NovaKey-Setup.exe
```

### 2) Run the installer

* Double-click **NovaKey-Setup.exe**
* Follow the on-screen prompts

The installer:

* Installs NovaKey into your user profile
* Creates a **per-user Scheduled Task**
* Starts the daemon automatically at login

No administrator privileges are required.

### 3) Permissions (first run)

Windows may prompt for:

* Firewall access (allow local network access)

---

## macOS Installation

### 1) Download

Download **one** of the following, depending on your Mac:

* Apple Silicon (M1/M2/M3):

  ```
  NovaKey-<version>-arm64.pkg
  ```
* Intel:

  ```
  NovaKey-<version>-amd64.pkg
  ```

### 2) Run the installer

* Double-click the `.pkg`
* Follow the installer prompts

The installer:

* Installs NovaKey into your user profile
* Registers a **LaunchAgent** that runs at login
* Starts the daemon automatically

### 3) Required macOS permissions

macOS will require explicit approval for typing automation.

After installation, open:

```
System Settings → Privacy & Security
```

Enable **NovaKey** under:

* **Accessibility**
* **Input Monitoring**

The daemon may not function correctly until both are enabled.

---

## Linux Installation

> ⚠️ **Linux security key / keyring warning**
>
> On Linux systems using hardware-backed authentication (for example **YubiKey**, smart cards,
> or PAM configurations that require external confirmation), NovaKey may be unable to unlock
> the system keyring during NovaKey-Daemon startup.
>
> If secure storage initialization fails or is cancelled multiple times,
> the daemon may not be able to complete pairing unless
> `require_sealed_device_store: false` is set in `server_config.yaml`.
> This setting requires the novakey service be restarted to apply.
>
> Restarting alone may not be sufficient if pairing state was already partially created.
>
> When restarting is not enough, remove the NovaKey **user configuration and data directories**
> and run the installer again:
>
> - `~/.config/novakey`
> - `~/.local/share/novakey`
>
> These directories are recreated automatically during install.
>
> - Fall back to a local device store (`devices.json`), **or**
> - Enter a state where pairing cannot be restarted automatically
>
> In this case, a **full uninstall and reinstall** of the daemon may be required to restart
> the secure pairing process.
>
> For best results on Linux:
> - Be prepared to complete pairing when the QR code is displayed
> - If signing in without a password via YubiKey or hardware token, cancel the keyring password prompt that comes up and utilize `require_sealed_device_store: false`

Reinstalling clears the daemon’s pairing state and forces a fresh secure bootstrap.

### 1) Download

Choose the correct package for your distribution and architecture:

* Debian / Ubuntu:

  ```
  novakey_<version>_amd64.deb
  novakey_<version>_arm64.deb
  ```
* Fedora / RHEL / openSUSE:

  ```
  novakey-<version>-1.amd64.rpm
  novakey-<version>-1.aarch64.rpm
  ```

### 2) Install

**Debian / Ubuntu**

```bash
sudo apt install ./novakey_<version>_amd64.deb
systemctl --user enable novakey --now
```

**Fedora / RHEL**

```bash
sudo dnf install ./novakey-<version>-1.amd64.rpm
systemctl --user enable novakey --now
```

### 3) Service behavior

* Installed as a **systemd user service**
* Starts automatically when you log in
* No system-wide daemon required

---

## Verifying Installation

### Windows

```powershell
Get-ScheduledTask -TaskName NovaKey
```

Optional (check listener):

```powershell
Get-NetTcpConnection -State Listen -LocalPort 60768
```

---

### macOS

```bash
launchctl list | grep novakey
```

---

### Linux

```bash
systemctl --user status novakey
ss -tunlp | grep 60768
```

---

## First Run & Device Pairing

On first startup, the NovaKey Daemon performs secure initialization and attempts to
bind itself to the platform’s native secure storage.

If no devices are paired during this phase:

- The daemon enters **pairing mode**
- A **time-limited QR code** (`novakey-pair.png`) is generated
- The QR is displayed automatically (Windows/macOS) or logged (Linux)

If pairing does not complete successfully and secure storage cannot be unlocked,
the daemon may fall back to a local device store (`devices.json`) or require a reinstall
to restart the pairing process.

This behavior is intentional and is designed to prevent indefinite pairing attempts.

If pairing does not complete as expected, see:
`docs/daemon/troubleshooting.md`

---

## Uninstalling NovaKey

### Windows

* Use **Apps & Features**
* During uninstall, you will be asked whether to **preserve pairing keys**

### macOS

* Remove via the installed package
* Or manually delete the LaunchAgent if needed

### Linux

```bash
sudo apt remove novakey
# or
sudo dnf remove novakey
```

---

## Legacy Installation (Deprecated)

> ⚠️ **Deprecated — use installers instead**

The original shell and PowerShell install scripts are still available for advanced or automated environments but are no longer recommended for end users.

Legacy scripts live under:

```
installers/legacy/
```

They may be removed in a future release.

---

## Why pairing is strict

NovaKey treats device pairing as a high-trust operation.
To prevent downgrade, replay, or brute-force pairing attempts:

- Pairing tokens are time-limited
- Secure storage failures are not retried indefinitely
- Manual recovery may require uninstalling and reinstalling the daemon

This ensures that pairing always reflects the user’s current security posture.


## Notes

* The daemon always runs **per-user**
* Secrets are stored securely using platform-native mechanisms
* No secrets are transmitted unless explicitly initiated by the NovaKey app

