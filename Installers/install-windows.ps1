#Requires -Version 3.0
#Requires -RunAsAdministrator
$ErrorActionPreference = "Stop"
$InformationPreference = "Continue"

$ServiceName    = "NovaKey"
$DisplayName    = "NovaKey Service"
$Description    = "NovaKey secure secret transfer service"

$ExeName        = "novakey-windows-amd64.exe"
$SourceExe      = Join-Path -Path $PSScriptRoot -ChildPath $ExeName
$InstallDir     = "$($env:ProgramFiles)\NovaKey"
$TargetExe      = Join-Path -Path $InstallDir -ChildPath $ExeName
$LogDir         = Join-Path -Path $InstallDir -ChildPath "logs"

$FirewallRule   = "NovaKey TCP Listener"
$ListenPort     = 60768

Write-Output -InputObject "[*] Installing NovaKey (Windows)"

# ------------------------------------------------------------
# Preconditions
# ------------------------------------------------------------
If (-NOT (Test-Path -Path $SourceExe)) {
    Throw "[x] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') novakey-service.exe not found in installer directory"
}

# ------------------------------------------------------------
# Create install directories
# ------------------------------------------------------------
Write-Information -MessageData "[*] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') Creating install directories"
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
New-Item -ItemType Directory -Path $LogDir -Force | Out-Null

# ------------------------------------------------------------
# Install binary
# ------------------------------------------------------------
Write-Information -MessageData "[*] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') Installing service binary"
Copy-Item -Path $SourceExe -Destination $TargetExe -Force

# ------------------------------------------------------------
# Remove existing service if present
# ------------------------------------------------------------
If (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
    Write-Information -MessageData "[*] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') Existing service found – removing"
    Stop-Service -Name $ServiceName -Force
    sc.exe delete $ServiceName | Out-Null
    Start-Sleep -Seconds 2
}

# ------------------------------------------------------------
# Create service with virtual service account (least privilege)
# ------------------------------------------------------------
Write-Information -MessageData "[*] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') Creating Windows service"

sc.exe create $ServiceName `
    binPath= "`"$TargetExe`"" `
    start= auto `
    DisplayName= "`"$DisplayName`"" `
    obj= "NT SERVICE\$ServiceName" | Out-Null

sc.exe description $ServiceName "$Description" | Out-Null

# ------------------------------------------------------------
# Set filesystem ACLs (service SID only)
# ------------------------------------------------------------
Write-Information -MessageData "[*] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') Setting directory permissions"

$Acl = Get-Acl -Path $InstallDir
$Acl.SetAccessRuleProtection($True, $False)

$RuleService = New-Object -TypeName System.Security.AccessControl.FileSystemAccessRule(
    "NT SERVICE\$ServiceName",
    "Modify",
    "ContainerInherit,ObjectInherit",
    "None",
    "Allow"
)

$RuleAdmins = New-Object -TypeName System.Security.AccessControl.FileSystemAccessRule(
    "Administrators",
    "FullControl",
    "ContainerInherit,ObjectInherit",
    "None",
    "Allow"
)

$RuleUsers = New-Object -TypeName System.Security.AccessControl.FileSystemAccessRule(
    "Users",
    "ReadAndExecute",
    "ContainerInherit,ObjectInherit",
    "None",
    "Allow"
)

$Acl.SetAccessRule($ruleService)
$Acl.AddAccessRule($ruleAdmins)
$Acl.AddAccessRule($ruleUsers)
Set-Acl -Path $InstallDir -AclObject $Acl

# ------------------------------------------------------------
# Firewall rule (IPv4 TCP)
# ------------------------------------------------------------
Write-Information -MessageData "[*] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') Configuring firewall rule"

If (-NOT (Get-NetFirewallRule -DisplayName $FirewallRule -ErrorAction SilentlyContinue)) {
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
Write-Information -MessageData "[*] $(Get-Date -Format 'MM-dd-yyyy hh:mm:ss') Starting service"
Start-Service -Name $ServiceName

Write-Output -InputObject "[✓] NovaKey installed successfully"
Write-Output -InputObject "    Service Name : $ServiceName"
Write-Output -InputObject "    Install Dir  : $InstallDir"
Write-Output -InputObject "    Port         : $ListenPort (IPv4)"
Write-Output -InputObject " "
