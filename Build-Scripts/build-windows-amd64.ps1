# Build-Scripts\build-windows-amd64.ps1
# REQUIREMENTS:
# 1.) git
# 2.) golang
Set-Location -Path ($PSScriptRoot | Split-Path)   # go to project root

# Get latest git tag or default to "dev"
$Version = (git describe --tags --abbrev=0 2>$Null) ?? "dev"
$LdFlags = "-s -w -X main.version=$($Version) -X main.buildDate=$(Get-Date -Format o)"

Write-Output -InputObject "Building NovaKey $Version for Windows AMD64"

# Clean previous build if requested
If ($Args -contains "clean") {
    Remove-Item -Recurse -Force -Path dist -ErrorAction SilentlyContinue
}  # End If

# Ensure output directory exists
New-Item -ItemType Directory -Force -Path dist | Out-Null

# Build Windows AMD64 executable
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "1"   # needed for robotgo
go build -trimpath -ldflags $ldflags -o dist/NovaKey.exe .

Write-Output -InputObject "Windows AMD64 build complete! Binary is in ./dist/NovaKey.exe"
