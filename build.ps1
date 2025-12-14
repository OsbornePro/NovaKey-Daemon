<#
.SYNOPSIS
This cmdlet is a cross-platform build script for NovaKey, nvclient, and nvpair (Windows, Linux, macOS)


.DESCRIPTION
Builds NovaKey, nvclient, and nvpair for Windows, Linux, or macOS (darwin) from a single PowerShell script.


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
        [Parameter(
            Mandatory=$False
        )]  # End Parameter
        [ValidateSet("windows", "linux", "darwin", IgnoreCase=$true)]
        [String]$Target = "windows",

        [Parameter(
            Mandatory=$False
        )]  # End Parameter
        [Switch]$Clean,

        [Parameter(
            Mandatory=$False
        )]  # End Parameter
        [String]$FileName
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
        If ($OutName.Length -eq 0) { $OutName = "novakey-windows-amd64.exe" }
        If ($OutName -notmatch '\.exe$') { $OutName += ".exe" }
        $Output = Join-Path -Path $DistDir -ChildPath $OutName
        ForEach ($Arch in @("amd64")) { #, "arm64")) {

            $env:GOOS = "darwin"
            $env:GOARCH = $Arch
            $env:CGO_ENABLED = "1"
            Write-Information -MessageData "[-] $OutName go build (windows/$Arch)"
            go build -trimpath -ldflags $LdFlags -o $Output "./cmd/novakey"

            Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') nvpair go build (windows/$Arch)"
            go build -o ".\dist\nvpair-windows-$Arch.exe" ".\cmd\nvpair"

            Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') nvclient go build (windows/$Arch)"
            go build -o ".\dist\nvclient-windows-$Arch.exe" ".\cmd\nvclient"

        }  # End ForEach   

    } "linux" {

        $env:CGO_ENABLED = 0
        $env:GOOS = "linux"
        $env:GOARCH = "amd64"
        $OutName = $FileName
        If ($OutName.Length -eq 0) { $OutName = "novakey-linux" }
        ForEach ($Arch in @("amd64")) { #, "arm64")) {

            $env:GOOS = "darwin"
            $env:GOARCH = $Arch
            $env:CGO_ENABLED = "1"
            $Output = Join-Path -Path $DistDir -ChildPath "$OutName-$Arch"
            Write-Information -MessageData "[-] $OutName go build (linux/$Arch)"
            go build -trimpath -ldflags $LdFlags -o $Output "./cmd/novakey"

            Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') nvpair go build (linux/$Arch)"
            go build -o ".\dist\nvpair-linux-$Arch" ".\cmd\nvpair"

            Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') nvclient go build (linux/$Arch)"
            go build -o ".\dist\nvclient-linux-$Arch" ".\cmd\nvclient"

        }  # End ForEach

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
        $OutName = $FileName
        If ($OutName.Length -eq 0) { $OutName = "novakey-darwin" }
        ForEach ($Arch in @("amd64")) { #, "arm64")) {

            $env:GOOS = "darwin"
            $env:GOARCH = $Arch
            $env:CGO_ENABLED = "1"
            $Output = Join-Path -Path $DistDir -ChildPath "$OutName-$Arch"
            Write-Information -MessageData "[-] $OutName go build (darwin/$Arch)"
            go build -trimpath -ldflags $LdFlags -o $Output ./cmd/novakey

            Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') nvpair go build (darwin/$Arch)"
            go build -o ".\dist\nvpair-darwin-$Arch" ".\cmd\nvpair"

            Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') nvclient go build (darwin/$Arch)"
            go build -o ".\dist\nvclient-darwin-$Arch" ".\cmd\nvclient"

        }  # End ForEach

        Write-Information "[-] To create a universal binary on macOS:"
        Write-Information "    lipo -create -output NovaKey NovaKey-darwin-amd64 NovaKey-darwin-arm64"
#>
    }  # End Switch Options

}  # End Switch
