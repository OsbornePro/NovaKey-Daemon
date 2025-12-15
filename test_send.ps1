<#
.SYNOPSIS
This cmdlet is used to impersonate what a phone app or other device would submit to have the daemon type a secret


.DESCRIPTION
Impersonate what a phone app or other device would submit to have the daemon type a secret


.PARAMETER Address
Define the address of the NovaKey-Daemon server

.PARAMETER DeviceID
Define the ID of the device

.PARAMETER Password
Define the password to type

.PARAMETER KeyHex
Define the device secret

.PARAMETER ServerKeysFile
Define file containing the servers kyber key values

.PARAMETER ArmAddress
Arm API address (local only)

.PARAMETER ArmTokenFile
Path to arm token file

.PARAMETER ArmMs
Arm duration in milliseconds

.PARAMETER TwoManEnabled
Handle when two man is enabled

.PARAMETER ApproveMagic
Define the approve magic secret

.EXAMPLE
PS> ."$env:USERPROFILE\Downloads\NovaKey-Daemon-main\NovaKey-Daemon-main\test_send.ps1"
# This example shows how to test using the defaults

PS> .\test_send.ps1 `
        -Address "127.0.0.1:60768" `
        -DeviceID "phone" `
        -Password "SuperStrongPassword123!" `
        -KeyHex "7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234" `
        -ServerKyberPubBase64 $B64String `
        -ArmAddress "127.0.0.1:60769" `
        -ArmTokenFile ".\arm_token.txt" `
        -ArmMs 20000
# This example shows how to test using defined parameters


.LINK
https://novakey.app/
https://osbornepro.com/


.NOTES
Author: Robert H. Osborne (OsbornePro)
Contact: security@novakey.app
Last Modified: 12/14/2025
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory=$False)]
    [String]$Address = "127.0.0.1:60768",

    [Parameter(Mandatory=$False)]
    [String]$DeviceID = "phone",

    [Parameter(Mandatory=$False)]
    [String]$Password = "SuperStrongPassword123!",

    [Parameter(Mandatory=$False)]
    [String]$KeyHex = "7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234",

    [Parameter(Mandatory=$False)]
    [String]$ServerKeysFile = ".\server_keys.json",

    [Parameter(Mandatory=$False)]
    [String]$ArmAddress = "127.0.0.1:60769",

    [Parameter(Mandatory=$False)]
    [String]$ArmTokenFile = ".\arm_token.txt",

    [Parameter(Mandatory=$False)]
    [Int]$ArmMs = 20000,

    [Parameter(Mandatory=$False)]
    [Bool]$TwoManEnabled = $True,

    [Parameter(Mandatory=$False)]
    [String]$ApproveMagic = "__NOVAKEY_APPROVE__",

    [Parameter(Mandatory=$False)]
    [switch]$UseLegacyApproveMagic
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ServerAddr = $Address
$NvClient = ".\dist\nvclient.exe"

Function Test-TcpPort {
    param(
        [Parameter(Mandatory=$True)][String]$ComputerName,
        [Parameter(Mandatory=$True)][Int]$Port,
        [Int]$TimeoutMs = 300
    )
    Try {
        $C = New-Object System.Net.Sockets.TcpClient
        $Iar = $C.BeginConnect($ComputerName, $Port, $Null, $Null)
        If (-not $Iar.AsyncWaitHandle.WaitOne($TimeoutMs, $False)) { $C.Close(); Return $False }
        $C.EndConnect($Iar) | Out-Null
        $C.Close()
        Return $True
    } Catch { Return $False }
}

Function Get-ServerKyberPubB64 {
    param([Parameter(Mandatory=$True)][String]$Path)

    If (!(Test-Path -Path $Path)) {
        Throw "server keys file not found: $Path"
    }

    $Raw = Get-Content -Raw -Path $Path
    $Obj = $Raw | ConvertFrom-Json

    If (-not $Obj.kyber768_public) {
        Throw "server_keys.json missing kyber768_public"
    }

    $B64 = ($Obj.kyber768_public.ToString()).Trim()
    $B64 = ($B64 -replace '\s+', '')

    Try { [Void][Convert]::FromBase64String($B64) }
    Catch { Throw "kyber768_public is not valid base64: $($_.Exception.Message)" }

    Return $B64
}

# Ensure nvclient exists
If (!(Test-Path -Path $NvClient)) {
    Write-Information -MessageData "[*] nvclient.exe not found; building it..."
    If (!(Test-Path -Path ".\dist")) { New-Item -ItemType Directory -Path ".\dist" | Out-Null }
    go build -o $NvClient .\cmd\nvclient
}
If (!(Test-Path -Path $NvClient)) { Throw "nvclient not found at $NvClient" }

$ServerKyberPubBase64 = Get-ServerKyberPubB64 -Path $ServerKeysFile

# --- ARM (optional) ---
Try {
    $ArmHost = $ArmAddress.Split(":")[0]
    $ArmPort = [int]$ArmAddress.Split(":")[1]

    If (Test-TcpPort -ComputerName $ArmHost -Port $ArmPort -TimeoutMs 300) {
        Write-Information -MessageData "[+] Arm API detected at $ArmAddress"

        If (!(Test-Path -Path $ArmTokenFile)) {
            Throw "Arm API is up but token file not found: $ArmTokenFile"
        }

        Write-Information -MessageData "[+] Arming for ${ArmMs}ms..."
        & $NvClient arm --addr $ArmAddress --token_file $ArmTokenFile --ms $ArmMs | Out-Host
        If ($LASTEXITCODE -ne 0) { Throw "Arming failed (exit code $LASTEXITCODE)" }
    } Else {
        Write-Warning -Message "[!] Arm API not detected at $ArmAddress (continuing without arming)"
    }
} Catch {
    Throw "Arm step skipped/failed: $($_.Exception.Message)"
}

# --- TWO-MAN approve (required if enabled) ---
If ($TwoManEnabled) {
    If ($UseLegacyApproveMagic) {
        Write-Information -MessageData "[+] Sending TWO-MAN legacy approve (msgType=1 payload magic)..."
        & $NvClient approve --legacy_magic --magic $ApproveMagic `
            -addr $ServerAddr `
            -device-id $DeviceID `
            -key-hex $KeyHex `
            -server-kyber-pub-b64 $ServerKyberPubBase64 | Out-Host
    } Else {
        Write-Information -MessageData "[+] Sending TWO-MAN approve (msgType=2 control frame)..."
        & $NvClient approve `
            -addr $ServerAddr `
            -device-id $DeviceID `
            -key-hex $KeyHex `
            -server-kyber-pub-b64 $ServerKyberPubBase64 | Out-Host
    }

    If ($LASTEXITCODE -ne 0) { Throw "Two-Man approve send failed (exit code $LASTEXITCODE)" }
}

Write-Warning -Message "Click into the browser address bar or somewhere to test the typing that will happen"
Start-Sleep -Seconds 3

Write-Information -MessageData "[+] Sending encrypted password frame to $ServerAddr..."
& $NvClient `
    -addr $ServerAddr `
    -device-id $DeviceID `
    -password $Password `
    -key-hex $KeyHex `
    -server-kyber-pub-b64 $ServerKyberPubBase64 | Out-Host

If ($LASTEXITCODE -ne 0) { Throw "Password send failed (exit code $LASTEXITCODE)" }

Write-Information -MessageData "[+] Done."

