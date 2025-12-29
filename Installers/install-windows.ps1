#Requires -Version 5.1
$ErrorActionPreference = "Stop"
$InformationPreference = "Continue"

# NovaKey Windows installer (per-user, Scheduled Task)
# - Installs binary + config under %LOCALAPPDATA%\NovaKey
# - Creates a Scheduled Task that runs at user logon (interactive)
# - Starts task immediately
# - Bootstraps pairing: If devices.json is absent, waits for novakey-pair.png and opens it for the user
# - Adds optional firewall rule (admin only)

$TaskName     = "NovaKey"
$TaskDesc     = "NovaKey secure secret transfer service (per-user)"
$ExeName      = "novakey-windows-amd64.exe"

$RepoRoot     = (Resolve-Path (Join-Path -Path $PSScriptRoot "..")).Path
$SourceExe    = Join-Path -Path $RepoRoot -ChildPath "dist\$ExeName"
$SourceYaml   = Join-Path -Path $RepoRoot -ChildPath "server_config.yaml"
$SourceDevices= Join-Path -Path $RepoRoot -ChildPath "devices.json"   # optional; only copied If present

$InstallDir   = Join-Path -Path $env:LOCALAPPDATA -ChildPath "NovaKey"
$TargetExe    = Join-Path -Path $InstallDir -ChildPath $ExeName
$ConfigPath   = Join-Path -Path $InstallDir -ChildPath "server_config.yaml"

$DevicesPath  = Join-Path -Path $InstallDir -ChildPath "devices.json"
$ServerKeys   = Join-Path -Path $InstallDir -ChildPath "server_keys.json"
$PairPng      = Join-Path -Path $InstallDir -ChildPath "novakey-pair.png"

$FirewallRule = "NovaKey TCP Listener (Per-User)"
$ListenPort   = 60768
Function Get-EffectiveLogonIdentity {
    # Returns: @{ Name = "AzureAD\User" or "DOMAIN\User" or "MACHINE\User"; Sid = "S-1-..." }
    $Name = (& whoami 2>$null).Trim()
    If (-not $Name) {
        $Name = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
    }
    $Sid = $Null
    Try {
        $SidLine = (& whoami /user 2>$null | Select-String -Pattern 'S-\d-\d+-.+').Line
        If ($SidLine) {
            $Sid = ($SidLine -split '\s+') | Where-Object -FilterScript { $_ -like 'S-*' } | Select-Object -First 1
        }
    } Catch {}
    Return @{ Name = $Name; Sid = $Sid }
}

Write-Output -InputObject "[*] Installing NovaKey (Windows) as a per-user Scheduled Task"

If (-not (Test-Path -Path $SourceExe))  { Throw "[x] Missing binary: $SourceExe" }
If (-not (Test-Path -Path $SourceYaml)) { Throw "[x] Missing config: $SourceYaml" }

$CurrentUserFull = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name  # DOMAIN\User

Write-Information -MessageData "[*] RepoRoot      : $RepoRoot"
Write-Information -MessageData "[*] InstallDir    : $InstallDir"
Write-Information -MessageData "[*] Binary        : $TargetExe"
Write-Information -MessageData "[*] Config(runtime): $ConfigPath"
Write-Information -MessageData "[*] User          : $CurrentUserFull"

New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null

Copy-Item -Path $SourceExe  -Destination $TargetExe  -Force
Copy-Item -Path $SourceYaml -Destination $ConfigPath -Force

# devices.json: only copy If present in repo; otherwise ensure it's absent to trigger pairing bootstrap.
If (Test-Path -Path $SourceDevices) {
    Copy-Item -Path $SourceDevices -Destination $DevicesPath -Force
} Else {
    If (Test-Path -Path $DevicesPath) { Remove-Item -Force $DevicesPath }
}

# server_keys.json should auto-generate; If it exists from a prior run, keep it.
# If you want "fresh install" behavior, uncomment the next line:
# If (Test-Path -Path $ServerKeys) { Remove-Item -Force $ServerKeys }
# Lock down InstallDir to the current user (and SYSTEM, Administrators)
# Lock down InstallDir to the current user (and SYSTEM, Administrators)
$Ident = Get-EffectiveLogonIdentity
$EffectiveUser = $Ident.Name
$EffectiveSid  = $Ident.Sid

Write-Information -MessageData "[*] ACL identity  : $EffectiveUser"
If ($EffectiveSid) { Write-Information -MessageData "[*] ACL SID       : $EffectiveSid" }
icacls $InstallDir /inheritance:r | Out-Null
icacls $InstallDir /grant:r "$($EffectiveUser):(OI)(CI)F" "SYSTEM:(OI)(CI)F" "Administrators:(OI)(CI)F" | Out-Null
If ($EffectiveUser -like "AzureAD\*") {
    If ($EffectiveSid) {
        icacls $InstallDir /grant "*$($EffectiveSid):(OI)(CI)F" | Out-Null
    } Else {
        Write-Information -MessageData "[!] AzureAD user detected but SID not found via whoami /user"
    }
}
icacls $InstallDir /remove "Users" "Authenticated Users" "Everyone" 2>$null | Out-Null
icacls $InstallDir /T /C | Out-Null

# Task action: WorkingDirectory MUST be InstallDir so relative paths resolve (devices.json, server_keys.json, ./logs, arm_token.txt)
$Action    = New-ScheduledTaskAction -Execute $TargetExe -Argument "--config `"$ConfigPath`"" -WorkingDirectory $InstallDir
$Trigger   = New-ScheduledTaskTrigger -AtLogOn
$Principal = New-ScheduledTaskPrincipal -UserId $CurrentUserFull -LogonType Interactive -RunLevel Limited
$Settings  = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -ExecutionTimeLimit ([TimeSpan]::Zero) `
    -RestartCount 3 `
    -RestartInterval (New-TimeSpan -Minutes 1)

Try {
    If (Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue) {
        Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
    }
} Catch {}

Write-Information -MessageData "[*] Creating Scheduled Task: $TaskName"
Register-ScheduledTask -TaskName $TaskName -Description $TaskDesc -Action $Action -Trigger $Trigger -Principal $Principal -Settings $Settings | Out-Null

Write-Information -MessageData "[*] Starting task now"
Start-ScheduledTask -TaskName $TaskName

# Pairing bootstrap (interactive open)
# If devices.json does not exist, the daemon should generate novakey-pair.png; open it automatically.
If (-not (Test-Path -Path $DevicesPath)) {
    Write-Information -MessageData "[*] Pairing bootstrap: waiting for $PairPng"
    $Sw = [Diagnostics.Stopwatch]::StartNew()
    $Opened = $false

    While ($Sw.Elapsed.TotalSeconds -lt 20) {
        If (Test-Path -Path $PairPng) {
            Try {
                Start-Process -FilePath $PairPng | Out-Null
                $Opened = $true
            } Catch {}
            Break
        }
        Start-Sleep -Milliseconds 250
    }

    If ($Opened) {
        Write-Information -MessageData "[*] Opened QR PNG for pairing: $PairPng"
    } Else {
        Write-Information -MessageData "[*] QR PNG not detected yet. If pairing mode is expected, check: $PairPng"
    }
}

# Firewall rule (best-effort; requires admin)
$IsAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()
).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

If ($IsAdmin) {
    Write-Information -MessageData "[*] Configuring firewall rules (admin)"
    If (-not (Get-NetFirewallRule -DisplayName $FirewallRule -ErrorAction SilentlyContinue)) {
        New-NetFirewallRule `
            -DisplayName $FirewallRule `
            -Direction Inbound `
            -Protocol TCP `
            -LocalPort $ListenPort `
            -Action Allow `
            -Profile Any | Out-Null
    }
} Else {
    Write-Information -MessageData "[*] Skipping firewall rule (not running as Administrator)"
}

Write-Output -InputObject ""
Write-Output -InputObject "[-] NovaKey installed (per-user)"
Write-Output -InputObject "    User        : $CurrentUserFull"
Write-Output -InputObject "    InstallDir  : $InstallDir"
Write-Output -InputObject "    Binary      : $TargetExe"
Write-Output -InputObject "    Config      : $ConfigPath"
Write-Output -InputObject "    Devices     : $DevicesPath"
Write-Output -InputObject "    Pair PNG    : $PairPng"
Write-Output -InputObject "    Task        : $TaskName"
Write-Output -InputObject ""
Write-Output -InputObject "To manage:"
Write-Output -InputObject "  schtasks /Query /TN $TaskName /V /FO LIST"
Write-Output -InputObject "  schtasks /Run   /TN $TaskName"
Write-Output -InputObject "  schtasks /End   /TN $TaskName"
