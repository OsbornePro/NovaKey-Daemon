# Build-Scripts\Build-Windows.ps1
Set-Location ($PSScriptRoot | Split-Path)   # go to project root

# Get latest git tag or default to "dev"
$version = (git describe --tags --abbrev=0 2>$null) ?? "dev"
$ldflags = "-s -w -X main.version=$version -X main.buildDate=$(Get-Date -Format o)"

Write-Host "Building NovaKey $version for Windows AMD64" -ForegroundColor Cyan

# Clean previous build if requested
if ($args -contains "clean") {
    Remove-Item -Recurse -Force dist -ErrorAction SilentlyContinue
}

# Ensure output directory exists
New-Item -ItemType Directory -Force -Path dist | Out-Null

# Build Windows AMD64 executable
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "1"   # needed for robotgo
go build -trimpath -ldflags $ldflags -o dist/NovaKey.exe .

Write-Host "`nWindows AMD64 build complete! Binary is in ./dist/NovaKey.exe" -ForegroundColor Green

