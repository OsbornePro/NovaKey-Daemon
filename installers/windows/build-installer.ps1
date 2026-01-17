param(
  [String]$ISCCPath = "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe",
  [String]$Version = "1.0.0"
)  # End param

$ErrorActionPreference = "Stop"
Push-Location -Path "$PSScriptRoot\helper"
go build -o "out\novakey-installer-helper.exe" .
Pop-Location
If (-NOT (Test-Path -Path $ISCCPath)) {
  Throw "ISCC.exe not found. Install Inno Setup 6, or update path in build-installer.ps1"
}  # End If

& $ISCC "/DMyAppVersion=$($Version)" "$($PSScriptRoot)\novakey.iss"
Write-Output -InputObject "Built installer at installers/windows/out/NovaKey-Setup.exe"
