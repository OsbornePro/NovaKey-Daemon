# Installing NovaKey-Daemon

## Supported platforms
- Windows 11
- macOS 14+
- Linux (systemd + glibc, root access for install scripts)

## Automatic installation (*recommended*)

### Uninstall Scripts
If you have used the install script to install your NovaKey-Daemon instance you can utilize the uninstall scripts to remove it or perform a re-freshed installation.
The uninstall scripts are execute the same as the install scripts are below. 
The difference being you would run `./Installers/uninstall-macos.sh` to uninstall instead of `./Installers/install-macos.sh` to install.

### 0) Build or download a binary

Easiest to use the pre-compiled binaries in the GitHub repository.
However, if you wish to compile the binary yourself you can use the included build scripts.

Windows:
```powershell
Set-ExecutionPolicy RemoteSigned
Unblock-File .\build.ps1
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

### 1) Download and Run the installer

Windows:
```powershell
# Download from the GitHub repository
Invoke-WebRequest -Uri https://github.com/OsbornePro/NovaKey-Daemon/archive/refs/heads/main.zip -OutFile $env:USERPROFILE\Downloads\NovaKey-Daemon-main.zip

# Extract the zip file contents
Expand-Archive -Path $env:USERPROFILE\Downloads\NovaKey-Daemon-main.zip -DestinationPath $env:USERPROFILE\Downloads\ -Force

# Ensure you can execute scripts in your session
Set-ExecutionPolicy RemoteSigned

# Unblock script downloaded from the internet
Unblock-File .\Installers\install-windows.ps1

# Run the installer
.\Installers\install-windows.ps1
```

Linux:
```bash
# Download
cd /tmp
git clone https://github.com/OsbornePro/NovaKey-Daemon.git

# Install
cd /tmp/NovaKey-Daemon
sudo bash Installers/install-linux.sh
```

macOS:
```bash
# Download
cd /tmp
git clone https://github.com/OsbornePro/NovaKey-Daemon.git

# Install
cd /tmp/NovaKey-Daemon
sudo bash Installers/install-macos.sh
```

### Verify installation

On Windows (*example*):
```powershell
Get-NetTcpConnection -State Listen -LocalPort 60768
```
You should see the listening port

On Linux (*example*):
```bash
systemctl status novakey --user
ss -tunlp | grep 60768
```
You should see the service running and the listening port

On Linux (*example*):
```bash
systemctl status novakey --user
netstat -at | grep 60768 # May take a little for command to complete
```
You should see the listening port

### First run and pairing QR

When there are no paired devices (*missing/empty device store*), the daemon generates a pairing QR (*novakey-pair.png*) at startup.


