<# 
.SYNOPSIS
This cmdlet is used to build the NovaKey service binary on Windows

.DESCRIPTION
Ensures the required tools are usable (go, git, mingw‑w64). Uses go to compile to binary with required parameters
    Requirements:
        1) git   – https://git-scm.com/install/windows
        2) Go    – https://go.dev/doc/install
        3) mingw‑w64 – https://code.visualstudio.com/docs/cpp/config-mingw

.PARAMETER Clean
Define this parameter to delete previous builds and compile the new binary from scratch

.PARAMETER FileName
If you wish to name your binary something specific it can be done so here

.EXAMPLE
PS> .\build-windows-amd64.ps1
# This example builds the binary file NovaKey.exe (Found in dist directory)

.EXAMPLE
PS> .\build-windows-amd64.ps1 -FileName NovaKey-Service.exe
# This example builds the binary file NovaKey-Service.exe (Found in dist directory)

.EXAMPLE
PS> .\build-windows-amd64.ps1 -Clean
# This example builds the binary file NovaKey.exe and deletes any preexisting compiled files. (Found in dist directory)

.EXAMPLE
PS> .\build-windows-amd64.ps1 -Clean -FileName NovaKey-Service.exe
# This example builds the binary file NovaKey-Service.exe and deletes any preexisting compiled files. (Found in dist directory)

.LINK
https://novakey.app/
https://osbornepro.com/

.NOTES
Author: Robert H. Osborne (OsbornePro)
Last Modified: 12/07/2025
Contact: security@novakey.app
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory=$False)]  # End Parameter
    [Switch]$Clean,

    [Parameter(Mandatory=$False)]  # End Parameter
    [String]$FileName = "NovaKey.exe"
)  # End param

Try {

    $InformationPreference = "Continue"

    # --------------------------------------------------------------------
    # Resolve the project root (the folder that contains this script)
    # --------------------------------------------------------------------
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
    Set-Location -Path $ProjectRoot

    # --------------------------------------------------------------------
    # Verify required tools are on the PATH
    # --------------------------------------------------------------------
    ForEach ($Tool in @("git","go")) {

        If (-not (Get-Command -Name $Tool -ErrorAction SilentlyContinue)) {
            Throw "[x] $(Get-Date -Format 'MM-dd-yyyy HH:mm:ss') $Tool is not installed or not on PATH. Exiting."
        }  # End If

    }  # End ForEach

    # --------------------------------------------------------------------
    # Determine the version string
    # --------------------------------------------------------------------
    Try {
        $Version = git describe --tags --abbrev=0 2>$Null | ForEach-Object -Process { $_.Trim() }
    } Catch {
        $Version = $Null
    }  # End Try Catch

    If (-not $Version) {
        $Version = "dev"
    }  # End If

    # --------------------------------------------------------------------
    # Build the ldflags string that embeds version & build date into the binary
    # --------------------------------------------------------------------
    $LdFlags = @(
        "-s"
        "-w"
        "-X", "main.version=$Version"
        "-X", "main.buildDate=$(Get-Date -Format o)"
    ) -join " "

    Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy HH:mm:ss') Building NovaKey $Version for Windows AMD64"

    # --------------------------------------------------------------------
    # Clean step (optional)
    # --------------------------------------------------------------------
    If ($Clean.IsPresent) {
        Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy HH:mm:ss') Cleaning previous build artifacts"
        Remove-Item -Recurse -Force -Path "dist" -ErrorAction SilentlyContinue
    }  # End If

    # --------------------------------------------------------------------
    # Ensure output directory exists
    # --------------------------------------------------------------------
    $DistDir = Join-Path -Path $ProjectRoot -ChildPath "dist"
    New-Item -ItemType Directory -Force -Path $DistDir | Out-Null

    # --------------------------------------------------------------------
    # Set environment variables for cross‑compilation
    # --------------------------------------------------------------------
    $env:GOOS       = "windows"
    $env:GOARCH     = "amd64"
    $env:CGO_ENABLED = "1"   # required for robotgo / cgo dependencies

    # --------------------------------------------------------------------
    # Ensure the output filename ends with .exe (helps when the user omits it)
    # --------------------------------------------------------------------
    If ($FileName -notmatch '\.exe$') { $FileName += ".exe" }

    # --------------------------------------------------------------------
    # Run the Go build – note the path to the actual main package
    # --------------------------------------------------------------------
    Try {

        go build -trimpath -ldflags $LdFlags -o (Join-Path -Path $DistDir -ChildPath $FileName) ./cmd/novakey
        If ($LASTEXITCODE -ne 0) {
            Throw "[x] $(Get-Date -Format 'MM-dd-yyyy HH:mm:ss') Go build failed with exit code $LASTEXITCODE"
        }  # End If

    } Catch {

        Throw "[x] $(Get-Date -Format 'MM-dd-yyyy HH:mm:ss') $_"

    }  # End Try Catch

} Finally {

    If (Test-Path -Path "$DistDir\$FileName") {

        Write-Information -MessageData "[-] $(Get-Date -Format 'MM-dd-yyyy HH:mm:ss') Windows AMD64 build complete! Binary is at $DistDir\$FileName"

    } Else {

        Write-Warning -Message "[!] $(Get-Date -Format 'MM-dd-yyyy HH:mm:ss') Failed to build Windows AMD64 binary at $DistDir\$FileName"

    }  # End If Else

    $InformationPreference = "SilentlyContinue"

}  # End Try Finally
