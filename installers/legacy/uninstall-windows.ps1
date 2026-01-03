#Requires -Version 5.1
$ErrorActionPreference = "Stop"
$InformationPreference = "Continue"

# NovaKey Windows uninstaller (per-user, Scheduled Task)
# - Stops + unregisters the Scheduled Task
# - Removes firewall rules (best-effort; admin only)
# - Deletes %LOCALAPPDATA%\NovaKey (and the installed exe/config)

$TaskName      = "NovaKey"
$ExeName       = "novakey-windows-amd64.exe"
$InstallDir    = Join-Path -Path $env:LOCALAPPDATA -ChildPath "NovaKey"
$TargetExe     = Join-Path -Path $InstallDir -ChildPath $ExeName

$FirewallRules = @(
  "NovaKey TCP Listener (Per-User)",
  "NovaKey TCP Pairing (Per-User)"
)

Function Log($msg) { Write-Information -MessageData ("[*] " + $msg) }

Log "Uninstalling NovaKey (Windows) per-user"
Log "User       : $([System.Security.Principal.WindowsIdentity]::GetCurrent().Name)"
Log "InstallDir : $InstallDir"
Log "Task       : $TaskName"

# --- Scheduled Task: stop + unregister (idempotent) ---
Try {
  $Task = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
  If ($Null -ne $Task) {
    Log "Stopping Scheduled Task (if running)"
    Try { Stop-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue | Out-Null } Catch {}

    Log "Unregistering Scheduled Task"
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false -ErrorAction Stop
  } Else {
    Log "Scheduled Task not found: $TaskName"
  }
} Catch {
  Write-Warning -Message "Failed to remove Scheduled Task '$TaskName': $($_.Exception.Message)"
}

# --- Firewall rules: best-effort, admin only ---
$IsAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()
).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

If (Get-Command Get-NetFirewallRule -ErrorAction SilentlyContinue) {
  If ($IsAdmin) {
    ForEach ($Rule in $FirewallRules) {
      Try {
        $R = Get-NetFirewallRule -DisplayName $Rule -ErrorAction SilentlyContinue
        If ($Null -ne $R) {
          Log "Removing firewall rule: $Rule"
          Remove-NetFirewallRule -DisplayName $Rule -ErrorAction SilentlyContinue | Out-Null
        } Else {
          Log "Firewall rule not found: $Rule"
        }
      } Catch {
        Write-Warning -Message "Failed to remove firewall rule '$rule': $($_.Exception.Message)"
      }
    }
  } Else {
    Log "Skipping firewall rule removal (not running as Administrator)"
  }
} Else {
  Log "NetFirewall cmdlets not available; skipping firewall removal"
}

# --- Remove files/folders ---
If (Test-Path -Path $InstallDir) {
  Log "Removing install directory: $InstallDir"
  Try {
    # Clear attributes just in case anything got marked read-only
    Get-ChildItem -LiteralPath $InstallDir -Recurse -Force -ErrorAction SilentlyContinue |
      ForEach-Object -Process { Try { $_.Attributes = 'Normal' } Catch {} }

    Remove-Item -LiteralPath $InstallDir -Recurse -Force -ErrorAction Stop
  } Catch {
    Write-Warning -Message "Failed to remove InstallDir '$InstallDir': $($_.Exception.Message)"
    Write-Warning -Message "If files are in use, log out/in and run again."
  }
} Else {
  Log "Install directory not found: $InstallDir"
}

Write-Output -InputObject " "
Write-Output -InputObject "[-] NovaKey uninstalled (per-user) (best-effort)"
Write-Output -InputObject "  Task removed : $TaskName"
Write-Output -InputObject "    InstallDir   : $InstallDir"
Write-Output -InputObject "    Binary       : $TargetExe"
Write-Output -InputObject ""

