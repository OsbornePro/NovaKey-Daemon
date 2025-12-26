# Installing NovaKey-Daemon

## Supported platforms
- Windows 10/11
- macOS 14+
- Linux (systemd + glibc, root access for install scripts)

## Automatic installation (recommended)

### 1) Build or download a binary
Windows:
```powershell
.\build.ps1 -Target Windows
# dist\novakey-windows-amd64.exe
```

Linux:
```bash
./build.sh -t linux
# dist/novakey-linux-amd64
```

macOS:
```bash
./build.sh -t darwin
# dist/novakey-darwin-amd64
```

Note: macOS builds often need to happen on macOS for correct signing/entitlements behavior.

### 2) Run the installer

Windows:
```powershell
Set-ExecutionPolicy RemoteSigned
Unblock-File .\Installers\install-windows.ps1
.\Installers\install-windows.ps1
```

Linux:
```bash
sudo bash Installers/install-linux.sh
```

macOS:
```bash
sudo bash Installers/install-macos.sh
```

### Verify installation

On Linux (*example*):
```bash
systemctl status novakey --user
```

You should see the service running.

### First run and pairing QR

When there are no paired devices (*missing/empty device store*), the daemon generates a pairing QR (*often novakey-pair.png*) at startup.
