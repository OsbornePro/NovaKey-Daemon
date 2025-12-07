#Requires -RunAsAdministrator
$ErrorActionPreference = "Stop"

$ServiceName    = "NovaKey"
$DisplayName    = "NovaKey Secure Typing Service"
$Description    = "NovaKey secure local password transfer service"

$ExeName        = "novakey-service.exe"
$SourceExe      = Join-Path $PSScriptRoot $ExeName
$InstallDir     = "C:\Program Files\NovaKey"
$TargetExe      = Join-Path $InstallDir $ExeName
$LogDir         = Join-Path $InstallDir "logs"

$FirewallRule   = "NovaKey TCP Listener"
$ListenPort     = 60768

Write-Host "[*] Installing NovaKey (Windows)"

# ------------------------------------------------------------
# Preconditions
# ------------------------------------------------------------
if (-not (Test-Path $SourceExe)) {
    Write-Error "novakey-service.exe not found in installer directory"
}

# ------------------------------------------------------------
# Create install directories
# ------------------------------------------------------------
Write-Host "[*] Creating install directories"
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
New-Item -ItemType Directory -Path $LogDir -Force | Out-Null

# ------------------------------------------------------------
# Install binary
# ------------------------------------------------------------
Write-Host "[*] Installing service binary"
Copy-Item $SourceExe $TargetExe -Force

# ------------------------------------------------------------
# Remove existing service if present
# ------------------------------------------------------------
if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
    Write-Host "[*] Existing service found – removing"
    Stop-Service $ServiceName -Force
    sc.exe delete $ServiceName | Out-Null
    Start-Sleep -Seconds 2
}

# ------------------------------------------------------------
# Create service with virtual service account (least privilege)
# ------------------------------------------------------------
Write-Host "[*] Creating Windows service"

sc.exe create $ServiceName `
    binPath= "`"$TargetExe`"" `
    start= auto `
    DisplayName= "`"$DisplayName`"" `
    obj= "NT SERVICE\$ServiceName" | Out-Null

sc.exe description $ServiceName "$Description" | Out-Null

# ------------------------------------------------------------
# Set filesystem ACLs (service SID only)
# ------------------------------------------------------------
Write-Host "[*] Setting directory permissions"

$acl = Get-Acl $InstallDir
$acl.SetAccessRuleProtection($true, $false)

$ruleService = New-Object System.Security.AccessControl.FileSystemAccessRule(
    "NT SERVICE\$ServiceName",
    "Modify",
    "ContainerInherit,ObjectInherit",
    "None",
    "Allow"
)

$ruleAdmins = New-Object System.Security.AccessControl.FileSystemAccessRule(
    "Administrators",
    "FullControl",
    "ContainerInherit,ObjectInherit",
    "None",
    "Allow"
)

$ruleUsers = New-Object System.Security.AccessControl.FileSystemAccessRule(
    "Users",
    "ReadAndExecute",
    "ContainerInherit,ObjectInherit",
    "None",
    "Allow"
)

$acl.SetAccessRule($ruleService)
$acl.AddAccessRule($ruleAdmins)
$acl.AddAccessRule($ruleUsers)
Set-Acl $InstallDir $acl

# ------------------------------------------------------------
# Firewall rule (IPv4 TCP)
# ------------------------------------------------------------
Write-Host "[*] Configuring firewall rule"

if (-not (Get-NetFirewallRule -DisplayName $FirewallRule -ErrorAction SilentlyContinue)) {
    New-NetFirewallRule `
        -DisplayName $FirewallRule `
        -Direction Inbound `
        -Protocol TCP `
        -LocalPort $ListenPort `
        -Action Allow `
        -Profile Any `
        | Out-Null
}

# ------------------------------------------------------------
# Start service
# ------------------------------------------------------------
Write-Host "[*] Starting service"
Start-Service $ServiceName

Write-Host
Write-Host "[✓] NovaKey installed successfully"
Write-Host "    Service Name : $ServiceName"
Write-Host "    Install Dir  : $InstallDir"
Write-Host "    Port         : $ListenPort (IPv4)"
Write-Host
