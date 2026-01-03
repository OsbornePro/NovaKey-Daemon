# How To Build Installers

### macOS

```bash
./build-installers-macos.sh 1.0.0
# after add Developer ID Installer + set notarytool creds:
# Get ID
security find-identity -v -p basic | grep "Developer ID Installer"
# Then sign
./sign-notarize-macos-pkgs.sh 1.0.0 novakey-notary
```

### Linux

```bash
./build-installers-linux.sh 1.0.0
```

### Windows (on Windows)

```powershell
.\build-installers-windows.ps1 -Version 1.0.0
```

