param([String]$Version = "1.0.0")
$ErrorActionPreference = "Stop"

Push-Location "$PSScriptRoot\helper"
go build -o "out\novakey-installer-helper.exe" .
Pop-Location

$ISCC = "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe"
If (-NOT (Test-Path -Path $ISCC)) {
  Throw "ISCC.exe not found. Install Inno Setup 6, or update path in build-installer.ps1"
}

& $ISCC "/DMyAppVersion=$Version" "$PSScriptRoot\novakey.iss"
Write-Output "Built installer at installers/windows/out/NovaKey-Setup.exe"
