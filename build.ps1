<#
.SYNOPSIS
This cmdlet is a cross-platform build script for NovaKey (Windows, Linux, macOS)


.DESCRIPTION
Builds NovaKey for Windows, Linux, or macOS (darwin) from a single PowerShell script.


.PARAMETER Target
windows | linux | darwin

.PARAMETER Clean
Delete previous builds before compiling

.PARAMETER FileName
Custom output filename (default: NovaKey.exe on Windows, NovaKey on others)


.EXAMPLE
PS> .\build.ps1 -Target windows
# Builds Windows AMD64

.EXAMPLE
PS> .\build.ps1 -Target linux
# Builds Linux AMD64

.EXAMPLE
PS> .\build.ps1 -Target darwin
# Builds universal macOS binary

.EXAMPLE
PS> .\build.ps1 -Target linux -Clean
# Builds Linux AMD64 and deletes dist directory and its contents


.NOTES
Author: Robert H. Osborne (OsbornePro)
Last Modified: 12/07/2025
Contact: security@novakey.app


.LINK
https://novakey.app/
https://osbornepro.com/
#>
[CmdletBinding()]
    param(
        [Parameter(Mandatory=$false)]
        [ValidateSet("windows", "linux", "darwin", IgnoreCase=$true)]
        [string]$Target = "windows",

        [Parameter(Mandatory=$false)]
        [switch]$Clean,

        [Parameter(Mandatory=$false)]
        [string]$FileName
    )  # End param

$ErrorActionPreference = "Stop"
$InformationPreference = "Continue"
$ProjectRoot = $PSScriptRoot
Set-Location -Path $ProjectRoot

Write-Verbose -Message "Verify required tools can be used"
ForEach ($Tool in "git", "go") {

    If (-not (Get-Command -Name $Tool -ErrorAction SilentlyContinue)) {
        Throw "[x] $Tool is required but not found in PATH"
    }  # End If

}  # End ForEach

# Get version tag
Try { $Version = (git describe --tags --abbrev=0 2>$Null).Trim() } Catch { }
If (-not $Version) { $Version = "dev" }

$LdFlags = "-s -w -X main.version=$Version -X main.buildDate=$(Get-Date -Format o)"
Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy HH:mm:ss') Building NovaKey $Version for $Target"

If ($Clean.IsPresent) {

    Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy HH:mm:ss') Cleaning previous build artifacts"
    Remove-Item -Recurse -Force -Path dist -ErrorAction SilentlyContinue

}  # End If

$DistDir = Join-Path -Path $ProjectRoot -ChildPath "dist"
New-Item -ItemType Directory -Force -Path $DistDir | Out-Null

Switch ($Target) {

    "windows" {

        $env:CGO_ENABLED = 0
        $env:GOOS = "windows"
        $env:GOARCH = "amd64"
        $OutName = $FileName
        If (-not $OutName) { $OutName = "NovaKey.exe" }
        If ($OutName -notmatch '\.exe$') { $OutName += ".exe" }
        $Output = Join-Path -Path $DistDir -ChildPath $OutName

        Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') go build (windows/amd64)"
        go build -trimpath -ldflags $LdFlags -o $Output ./cmd/novakey

    } "linux" {

        $env:CGO_ENABLED = 0
        $env:GOOS = "linux"
        $env:GOARCH = "amd64"
        $OutName = $FileName
        If (-not $OutName) { $OutName = "NovaKey" }
        $Output = Join-Path -Path $DistDir -ChildPath $OutName

        Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') go build (linux/amd64)"
        go build -trimpath -ldflags $LdFlags -o $Output ./cmd/novakey

    } "darwin" {

        Write-Warning -Message @"
macOS builds must be performed on macOS.

Reason:
  NovaKey uses CGO + Apple Cocoa / Accessibility APIs,
  which may not cross-compile cleanly from this host. 

What to do:
  Run this command on a Mac with Xcode installed:
      ./build.sh -t darwin
      ./build.ps1 -Target darwin
"@
        Return
<#
# In case it ever becomes possible
        Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') Attempting build of macOS binaries"
        ForEach ($Arch in @("amd64", "arm64")) {

            $env:GOOS = "darwin"
            $env:GOARCH = $Arch
            $env:CGO_ENABLED = "1"

            $Output = Join-Path $DistDir "NovaKey-darwin-$Arch"
            Write-Information "[-] go build (darwin/$Arch)"
            go build -trimpath -ldflags $LdFlags -o $Output ./cmd/novakey

        }  # End ForEach

        Write-Information "[-] To create a universal binary on macOS:"
        Write-Information "    lipo -create -output NovaKey NovaKey-darwin-amd64 NovaKey-darwin-arm64"
#>
    }  # End Switch Options

}  # End Switch
