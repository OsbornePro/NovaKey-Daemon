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
$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location -Path $ProjectRoot

Write-Verbose -Message "Verify required tools can be used"
ForEach ($Tool in "git", "go") {

    If (-not (Get-Command -Name $Tool -ErrorAction SilentlyContinue)) {
        Throw "[x] $Tool is required but not found in PATH"
    }  # End If

}  # End ForEach

# Get version tag
Try { $Version = (git describe --tags --abbrev=0 2>$null).Trim() } Catch { }
If (-not $Version) { $Version = "dev" }

$LdFlags = "-s -w -X main.version=$Version -X main.buildDate=$(Get-Date -Format o)"
Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy HH:mm:ss') Building NovaKey $Version for $Target"

If ($Clean.IsPresent) {

    Write-Information "[-] $(Get-Date -Format 'MM-dd-yyyy HH:mm:ss') Cleaning previous build artifacts"
    Remove-Item -Recurse -Force -Path dist -ErrorAction SilentlyContinue

}  # End If

$DistDir = Join-Path -Path $ProjectRoot -ChildPath "dist"
New-Item -ItemType Directory -Force -Path $DistDir | Out-Null

$env:GOARCH = "amd64"
$env:GOOS    = $Target
$env:CGO_ENABLED = "0"

Switch ($Target) {

    "windows" {
        $env:CGO_ENABLED = "0"
        $DefaultName = "NovaKey.exe"
    } "linux" {
        $DefaultName = "NovaKey"
    } "darwin" {
        $env:GOARCH = "all"     # Go 1.20+ magic for universal darwin binaries
        $DefaultName = "NovaKey"
    }  # End Switch options

}  # End Switch

# Final filename
If (-not $FileName) { $FileName = $DefaultName }
If ($Target -eq "windows" -and $FileName -notmatch '\.exe$') {
    $FileName += ".exe"
}  # End If

$OutputPath = Join-Path -Path $DistDir -ChildPath $FileName

# Build!
Write-Information "[-] $(Get-Date -Format 'MM-dd-yyyy HH:mm:ss') Running: go build -ldflags='$LdFlags' -o $OutputPath ./cmd/novakey"
go build -trimpath -ldflags $LdFlags -o $OutputPath ./cmd/novakey

If ($LASTEXITCODE -ne 0) {
    Throw "[x] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') Go build failed with exit code $LASTEXITCODE"
}  # End If

Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') SUCCESS! Binary created:`n   $OutputPath`n"
Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') Platform: $Target $(If($Target -eq 'darwin'){'(Universal Intel + Apple Silicon)'} Else {''})" 
