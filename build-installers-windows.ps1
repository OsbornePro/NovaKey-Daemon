param(
  [String]$Version = "1.0.0"
)  # End param

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSCommandPath
Set-Location -Path $Root

# Sanity check inputs
$NK = Join-Path -Path $Root -ChildPath "dist\windows\novakey.exe"
If (-NOT (Test-Path -Path $NK)) {
  Throw "Missing $($NK). Build binaries first (build.ps1 / build.sh)."
}  # End If

Write-Output -InputObject "[-] Building Windows installer (version label is set in novakey.iss)"
powershell -ExecutionPolicy Bypass -File ".\installers\windows\build-installer.ps1"

Write-Output -InputObject "[-] Built: installers\windows\out\NovaKey-Setup.exe"

