<#
.SYNOPSIS
Builds NovaKey, nvclient, nvpair. Default: binaries only.
Optional: -Package to build installer/package artifacts.


.DESCRIPTION
Build the NovaKey daemon binary that creates a listener on devices


.PARAMETER Target
Define which OS to create the build for. Mac builds will not work on windows. This is here for possible future compatability

.PARAMETER Clean
Define whether to delete previous builds in the dist/ directory binaries are saved too

.PARAMETER Package
Define whether you wish to package the build


.EXAMPLE
PS> .\build.sh -Target Windows
# This example builds the NovaKey binary for Windows

.EXAMPLE
PS> .\build.sh -Target Windows
# This example builds the NovaKey binary for Windows and removes any binaries existing in the dist/ directory


.LINK
https://novakey.app/
https://osbornepro.com/


.NOTES
Author: Robert H. Osborne
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

ForEach ($Tool in @("git", "go")) {
  If (-not (Get-Command -Name $Tool -ErrorAction SilentlyContinue)) {
    Throw "[x] $Tool is required but not found in PATH"
  }
}

Try { $Version = (git describe --tags --abbrev=0 2>$null).Trim() } Catch { }
If (-not $Version) { $Version = "dev" }

$BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$LdFlags = "-s -w -X main.version=$Version -X main.buildDate=$BuildDate"

Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy HH:mm:ss') Building NovaKey $Version for $Target"
If ($Clean.IsPresent) {
  Write-Information -MessageData "[-] Cleaning dist/"
  Remove-Item -Recurse -Force -Path dist -ErrorAction SilentlyContinue
}

$DistRoot    = Join-Path -Path $ProjectRoot -ChildPath "dist"
$DistWindows = Join-Path -Path $DistRoot -ChildPath "windows"
$DistLinux   = Join-Path -Path $DistRoot -ChildPath "linux"
$DistMac     = Join-Path -Path $DistRoot -ChildPath "macos"
New-Item -ItemType Directory -Force -Path $DistWindows,$DistLinux,$DistMac | Out-Null

Switch ($Target.ToLower()) {

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
  } "linux" {
    $env:CGO_ENABLED = "0"
    $env:GOOS = "linux"

    ForEach ($Arch in @("amd64","arm64")) {
      $env:GOARCH = $Arch

      Write-Information -MessageData "[-] go build novakey (linux/$Arch)"
      go build -trimpath -ldflags $LdFlags -o (Join-Path $DistLinux "novakey-linux-$Arch.elf") "./cmd/novakey"

      Write-Information -MessageData "[-] go build nvpair (linux/$Arch)"
      go build -trimpath -ldflags $LdFlags -o (Join-Path $DistLinux "nvpair-linux-$Arch.elf") "./cmd/nvpair"

      Write-Information -MessageData "[-] go build nvclient (linux/$Arch)"
      go build -trimpath -ldflags $LdFlags -o (Join-Path $DistLinux "nvclient-linux-$Arch.elf") "./cmd/nvclient"
    }

    Write-Information -MessageData "[✓] Linux binaries built (dist/linux/)"
  } "darwin" {
    Write-Warning -Message @"
macOS builds must be performed on macOS.

Run on a Mac:
  ./build.sh -t darwin
Then package:
  ./installers/macos/pkg/build-pkg.sh $Version arm64
  ./installers/macos/pkg/build-pkg.sh $Version amd64
"@
    Return
  }
}

# ---------------- Package (ONLY when requested) ----------------
If ($Package.IsPresent) {
  Write-Information -MessageData "[-] Packaging enabled (-Package)"

  If ($Target.ToLower() -eq "windows") {
    $BuildInstaller = Join-Path $ProjectRoot "installers\windows\build-installer.ps1"
    If (-not (Test-Path -Path $BuildInstaller)) {
      Throw "Missing: $BuildInstaller"
    }
    powershell -ExecutionPolicy Bypass -File $BuildInstaller
    Write-Information -MessageData "[-] Windows installer built (installers/windows/out/)"
  }

  If ($Target.ToLower() -eq "linux") {
    Write-Warning -Message "Linux packaging should be run on Linux."
    Write-Warning -Message "On Linux:"
    Write-Warning -Message "  cp dist/linux/novakey-linux-amd64.elf dist/linux/novakey"
    Write-Warning -Message "  chmod +x dist/linux/novakey"
    Write-Warning -Message "  ./installers/linux/nfpm/build-packages.sh $Version"
  }

  If ($Target.ToLower() -eq "darwin") {
    Write-Warning -Message "macOS packaging must be run on macOS: installers/macos/pkg/build-pkg.sh"
  }
}

