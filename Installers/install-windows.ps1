#Requires -Version 5.1
$ErrorActionPreference = "Stop"
$InformationPreference = "Continue"

# NovaKey Windows installer (per-user, Scheduled Task)
# - Installs binary + config under %LOCALAPPDATA%\NovaKey
# - Creates a Scheduled Task that runs at user logon (and can be started immediately)
# - Runs as the current user (so UI automation / focused typing can work)
# - Adds an optional firewall rule (requires admin; if not admin, it will skip)
# - Does NOT create devices.json if missing (pairing bootstrap should handle it)

$TaskName     = "NovaKey"
$TaskDesc     = "NovaKey secure secret transfer service (per-user)"
$ExeName      = "novakey-windows-amd64.exe"

$RepoRoot     = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$SourceExe    = Join-Path $RepoRoot "dist\$ExeName"
$SourceYaml   = Join-Path $RepoRoot "server_config.yaml"
$SourceDevices= Join-Path $RepoRoot "devices.json"   # optional

$InstallDir   = Join-Path $env:LOCALAPPDATA "NovaKey"
$TargetExe    = Join-Path $InstallDir $ExeName
$LogDir       = Join-Path $InstallDir "logs"
$ConfigPath   = Join-Path $InstallDir "server_config.yaml"

$FirewallRule = "NovaKey TCP Listener (Per-User)"
$ListenPort   = 60768

Write-Output "[*] Installing NovaKey (Windows) as a per-user Scheduled Task"

# Preconditions
if (-not (Test-Path $SourceExe))   { throw "[!] Missing binary: $SourceExe" }
if (-not (Test-Path $SourceYaml))  { throw "[!] Missing config: $SourceYaml" }

Write-Information "[*] RepoRoot    : $RepoRoot"
Write-Information "[*] InstallDir  : $InstallDir"
Write-Information "[*] Binary      : $TargetExe"
Write-Information "[*] Config      : $ConfigPath"

# Create install dirs
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
New-Item -ItemType Directory -Path $LogDir -Force | Out-Null

# Install binary + config
Copy-Item -Path $SourceExe -Destination $TargetExe -Force
Copy-Item -Path $SourceYaml -Destination $ConfigPath -Force

# devices.json: install if present; otherwise do NOT create it
if (Test-Path $SourceDevices) {
    Copy-Item -Path $SourceDevices -Destination (Join-Path $InstallDir "devices.json") -Force
} else {
    $maybe = Join-Path $InstallDir "devices.json"
    if (Test-Path $maybe) { Remove-Item -Force $maybe }
}

# Build task action:
# Working directory is InstallDir so relative paths in YAML resolve (devices.json, server_keys.json, ./logs)
$Action  = New-ScheduledTaskAction -Execute $TargetExe -Argument "--config `"$ConfigPath`"" -WorkingDirectory $InstallDir
$Trigger = New-ScheduledTaskTrigger -AtLogOn
$Principal = New-ScheduledTaskPrincipal -UserId "$env:USERNAME" -LogonType Interactive -RunLevel LeastPrivilege
$Settings  = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit (New-TimeSpan -Hours 0) -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1)

# Remove existing task if present
try {
    if (Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue) {
        Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
    }
} catch {}

Write-Information "[*] Creating Scheduled Task: $TaskName"
Register-ScheduledTask -TaskName $TaskName -Description $TaskDesc -Action $Action -Trigger $Trigger -Principal $Principal -Settings $Settings | Out-Null

# Try to start immediately
Write-Information "[*] Starting task now"
Start-ScheduledTask -TaskName $TaskName

# Firewall rule (best-effort; requires admin)
$IsAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()
).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if ($IsAdmin) {
    Write-Information "[*] Configuring firewall rule (admin)"
    if (-not (Get-NetFirewallRule -DisplayName $FirewallRule -ErrorAction SilentlyContinue)) {
        New-NetFirewallRule `
            -DisplayName $FirewallRule `
            -Direction Inbound `
            -Protocol TCP `
            -LocalPort $ListenPort `
            -Action Allow `
            -Profile Any | Out-Null
    }
} else {
    Write-Information "[*] Skipping firewall rule (not running as Administrator)"
}

Write-Output ""
Write-Output "[✓] NovaKey installed (per-user)"
Write-Output "    User       : $env:USERNAME"
Write-Output "    InstallDir : $InstallDir"
Write-Output "    Binary     : $TargetExe"
Write-Output "    Config     : $ConfigPath"
Write-Output "    Logs       : $LogDir"
Write-Output "    Task       : $TaskName"
Write-Output ""
Write-Output "To manage:"
Write-Output "  schtasks /Query /TN $TaskName /V /FO LIST"
Write-Output "  schtasks /Run /TN $TaskName"
Write-Output "  schtasks /End /TN $TaskName"

