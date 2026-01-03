<#
.SYNOPSIS
Builds NovaKey, nvclient, nvpair. Default: binaries only.
Optional: -Package to build installer/package artifacts.

.NOTES
- Windows packaging uses installers/windows/build-installer.ps1
- Linux/macOS packaging should be done on those OSes (this script warns accordingly)
#>
[CmdletBinding()]
param(
  [ValidateSet("windows", "linux", "darwin", IgnoreCase=$true)]
  [String]$Target = "windows",

  [Switch]$Clean,

  [Switch]$Package
)

$ErrorActionPreference = "Stop"
$InformationPreference = "Continue"

$ProjectRoot = $PSScriptRoot
Set-Location -Path $ProjectRoot

foreach ($Tool in @("git", "go")) {
  if (-not (Get-Command -Name $Tool -ErrorAction SilentlyContinue)) {
    throw "[x] $Tool is required but not found in PATH"
  }
}

try { $Version = (git describe --tags --abbrev=0 2>$null).Trim() } catch { }
if (-not $Version) { $Version = "dev" }

$BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$LdFlags = "-s -w -X main.version=$Version -X main.buildDate=$BuildDate"

Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy HH:mm:ss') Building NovaKey $Version for $Target"

if ($Clean.IsPresent) {
  Write-Information -MessageData "[-] Cleaning dist/"
  Remove-Item -Recurse -Force -Path dist -ErrorAction SilentlyContinue
}

$DistRoot    = Join-Path $ProjectRoot "dist"
$DistWindows = Join-Path $DistRoot "windows"
$DistLinux   = Join-Path $DistRoot "linux"
$DistMac     = Join-Path $DistRoot "macos"
New-Item -ItemType Directory -Force -Path $DistWindows,$DistLinux,$DistMac | Out-Null

switch ($Target.ToLower()) {

  "windows" {
    $env:CGO_ENABLED = "0"
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"

    $GuiLdFlags = "$LdFlags -H=windowsgui"

    Write-Information -MessageData "[-] go build novakey (windows/amd64)"
    go build -trimpath -ldflags $GuiLdFlags -o (Join-Path $DistWindows "novakey.exe") "./cmd/novakey"

    Write-Information -MessageData "[-] go build nvpair (windows/amd64)"
    go build -trimpath -ldflags $LdFlags -o (Join-Path $DistWindows "nvpair-windows-amd64.exe") "./cmd/nvpair"

    Write-Information -MessageData "[-] go build nvclient (windows/amd64)"
    go build -trimpath -ldflags $LdFlags -o (Join-Path $DistWindows "nvclient-windows-amd64.exe") "./cmd/nvclient"

    Write-Information -MessageData "[✓] Windows binaries built (dist/windows/)"
  }

  "linux" {
    $env:CGO_ENABLED = "0"
    $env:GOOS = "linux"

    foreach ($Arch in @("amd64","arm64")) {
      $env:GOARCH = $Arch

      Write-Information -MessageData "[-] go build novakey (linux/$Arch)"
      go build -trimpath -ldflags $LdFlags -o (Join-Path $DistLinux "novakey-linux-$Arch.elf") "./cmd/novakey"

      Write-Information -MessageData "[-] go build nvpair (linux/$Arch)"
      go build -trimpath -ldflags $LdFlags -o (Join-Path $DistLinux "nvpair-linux-$Arch.elf") "./cmd/nvpair"

      Write-Information -MessageData "[-] go build nvclient (linux/$Arch)"
      go build -trimpath -ldflags $LdFlags -o (Join-Path $DistLinux "nvclient-linux-$Arch.elf") "./cmd/nvclient"
    }

    Write-Information -MessageData "[✓] Linux binaries built (dist/linux/)"
  }

  "darwin" {
    Write-Warning @"
macOS builds must be performed on macOS.

Run on a Mac:
  ./build.sh -t darwin
Then package:
  ./installers/macos/pkg/build-pkg.sh $Version arm64
  ./installers/macos/pkg/build-pkg.sh $Version amd64
"@
    return
  }
}

# ---------------- Package (ONLY when requested) ----------------
if ($Package.IsPresent) {
  Write-Information -MessageData "[-] Packaging enabled (-Package)"

  if ($Target.ToLower() -eq "windows") {
    $BuildInstaller = Join-Path $ProjectRoot "installers\windows\build-installer.ps1"
    if (-not (Test-Path $BuildInstaller)) {
      throw "Missing: $BuildInstaller"
    }
    powershell -ExecutionPolicy Bypass -File $BuildInstaller
    Write-Information -MessageData "[✓] Windows installer built (installers/windows/out/)"
  }

  if ($Target.ToLower() -eq "linux") {
    Write-Warning "Linux packaging should be run on Linux."
    Write-Warning "On Linux:"
    Write-Warning "  cp dist/linux/novakey-linux-amd64.elf dist/linux/novakey"
    Write-Warning "  chmod +x dist/linux/novakey"
    Write-Warning "  ./installers/linux/nfpm/build-packages.sh $Version"
  }

  if ($Target.ToLower() -eq "darwin") {
    Write-Warning "macOS packaging must be run on macOS: installers/macos/pkg/build-pkg.sh"
  }
}

