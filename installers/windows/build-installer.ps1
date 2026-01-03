$ErrorActionPreference = "Stop"

# Build helper
Push-Location "$PSScriptRoot\helper"
go build -o "out\novakey-installer-helper.exe" .
Pop-Location

# Compile Inno Setup (adjust path if needed)
$ISCC = "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe"
If (-NOT (Test-Path -Path $ISCC)) {
  Throw "ISCC.exe not found. Install Inno Setup 6, or update path in build-installer.ps1"
}

& $ISCC "$PSScriptRoot\novakey.iss"
Write-Output -InputObject "Built installer at installers/windows/out/NovaKey-Setup.exe"

