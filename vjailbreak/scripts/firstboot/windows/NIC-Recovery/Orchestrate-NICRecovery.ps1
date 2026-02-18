# Orchestrate-NICRecovery.ps1
$ScriptRoot = "C:\NIC-Recovery"
$LogFile = Join-Path $ScriptRoot "NIC-Recovery.log"
$TaskName = "NICRecoveryPostReboot"
function Write-Log {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logEntry = "$timestamp - $Level - $Message"
    
    try {
        if (-not (Test-Path $ScriptRoot)) {
            New-Item -ItemType Directory -Path $ScriptRoot -Force | Out-Null
        }
        $logEntry | Out-File -FilePath $LogFile -Append -Encoding utf8
    } catch {
        Write-Host "Failed to write to log file: $_"
    }
    
    if ($Level -eq "ERROR") {
        Write-Host $logEntry -ForegroundColor Red
    } else {
        Write-Host $logEntry
    }
}
function Schedule-MyTask {
    param(
        [string]$TaskName,
        [string]$ScriptPath,
        [string]$Description
    )
   
    $taskName    = $TaskName
    $scriptPath  = $ScriptPath
    $description = $Description

    if (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue) {
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
        Start-Sleep -Seconds 2   # give it time to release locks
    }

    # Action: run your .ps1 file (bypass execution policy just in case)
    $action = New-ScheduledTaskAction `
        -Execute "powershell.exe" `
        -Argument "-NoProfile -ExecutionPolicy Bypass -File `"$scriptPath`""

    # Trigger: at startup (next boot), only once
    $trigger = New-ScheduledTaskTrigger -AtStartup

    # ─── Changed: use SYSTEM instead of Administrators group ───
    $principal = New-ScheduledTaskPrincipal `
        -UserId    "NT AUTHORITY\SYSTEM" `
        -LogonType ServiceAccount `
        -RunLevel  Highest

    # Settings: added StartWhenAvailable + explicit power settings
    $settings = New-ScheduledTaskSettingsSet `
        -AllowStartIfOnBatteries    `
        -DontStopIfGoingOnBatteries `
        -ExecutionTimeLimit          (New-TimeSpan -Minutes 30)

    # Create the task
    Register-ScheduledTask -TaskName $taskName `
        -Action      $action `
        -Trigger     $trigger `
        -Principal   $principal `
        -Settings    $settings `
        -Description $description `
        -Force

    Write-Log "Task '$taskName' created → will run your script once after next reboot."
}
function Ensure-64BitPowerShell {
    if (-not [Environment]::Is64BitOperatingSystem) {
        Write-Verbose "This is a 32-bit operating system → no 64-bit PowerShell available. Continuing as-is."
        return
    }

    if ([Environment]::Is64BitProcess) {
        Write-Verbose "Already running in 64-bit PowerShell process."
        return
    }

    Write-Log "Detected 32-bit PowerShell on 64-bit OS → Relaunching in 64-bit PowerShell..." -ForegroundColor Yellow

    $ps64Path = "$env:windir\sysnative\WindowsPowerShell\v1.0\powershell.exe"

    if (-not (Test-Path $ps64Path)) {
        Write-Warning "64-bit PowerShell not found at $ps64Path. Continuing in 32-bit (may cause issues)."
        return
    }

    $argList = @(
        '-NoProfile',
        '-ExecutionPolicy', 'Bypass'
    )

    if ($args.Count -gt 0) {
        $argList += $args
    }

    $argList += '-File'
    $argList += $PSCommandPath   # or $MyInvocation.MyCommand.Definition

    try {
        Start-Process -FilePath $ps64Path `
                      -ArgumentList $argList `
                      -Wait `
                      -NoNewWindow

        Write-Log "64-bit execution completed. Exiting 32-bit instance."
        exit 0
    }
    catch {
        Write-Log "Failed to relaunch in 64-bit PowerShell: $_" -Level "ERROR"
    }
}

function Remove-MyTask{
    param(
        [string]$TaskName
    )
    
    $taskName = $TaskName
    
    try {
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
        Write-Log "Task '$taskName' removed successfully."
    } catch {
        Write-Log "Failed to remove task '$taskName': $_" -Level "ERROR"
    }
}

try {
    Write-Log "=== Starting NIC Recovery Orchestration ==="
    # Start Network Setup Service
    Ensure-64BitPowerShell
    try {
        Write-Log "Starting Network Setup Service..."
        Start-Service NetSetupSvc -ErrorAction Stop
        Write-Log "Network Setup Service started successfully"
    } catch {
        Write-Log "Warning: Failed to start Network Setup Service - $_" -Level "WARNING"
    }
    

    # Run Recover-HiddenNICMapping.ps1
    if (Test-Path "$ScriptRoot\netconfig.json") {
        Write-Log "File already exists skipping step"
    } else {
        Write-Log "Network configuration not found, running Recover-HiddenNICMapping.ps1"
        try {
            Write-Log "Running Recover-HiddenNICMapping.ps1..."
            & "$ScriptRoot\Recover-HiddenNICMapping.ps1" *>> $LogFile
            if ($LASTEXITCODE -ne 0) { throw "Script failed with exit code $LASTEXITCODE" }
            Write-Log "Recover-HiddenNICMapping.ps1 completed successfully"
        } catch {
            Write-Log "Error in Recover-HiddenNICMapping.ps1 - $_" -Level "ERROR"
            throw $_
        }
    }
    
    # Check if pnputil.exe is recognized
    if ((Get-Command pnputil.exe -ErrorAction SilentlyContinue) -or (Test-Path "C:\Windows\System32\pnputil.exe")) {
        Write-Log "pnputil.exe is recognized"
        # Run Cleanup-GhostNICs.ps1
        try {
            Write-Log "Running Cleanup-GhostNICs.ps1..."
            & "$ScriptRoot\Cleanup-GhostNICs.ps1" *>> $LogFile
            if ($LASTEXITCODE -ne 0) { throw "Script failed with exit code $LASTEXITCODE" }
            Write-Log "Cleanup-GhostNICs.ps1 completed successfully"
        } catch {
            Write-Log "Error in Cleanup-GhostNICs.ps1 - $_" -Level "ERROR"
            throw $_
        }
        # Check for network configuration and restore if needed
        if (Test-Path "$ScriptRoot\netconfig.json") {
            try {
                Write-Log "Network configuration found, running Restore-Network.ps1..."
                & "$ScriptRoot\Restore-Network.ps1" *>> $LogFile
                if ($LASTEXITCODE -ne 0) { throw "Script failed with exit code $LASTEXITCODE" }
                Write-Log "Network configuration restored successfully"
            } catch {
                Write-Log "Error restoring network configuration - $_" -Level "ERROR"
                throw $_
            }
        } else {
            Write-Log "No network configuration found (netconfig.json not present)"
        }

        Write-Log "=== NIC Recovery Orchestration completed successfully ==="
        Remove-MyTask -TaskName $TaskName
        exit 0
    } else {
        Write-Log "Pnputil not found rescheduling NIC recovery post reboot" -Level "WARNING"
        Schedule-MyTask -TaskName $TaskName -ScriptPath "$ScriptRoot\Orchestrate-NICRecovery.ps1" -Description "Runs once after next reboot then deletes itself"
        exit 0
    }


} catch {
    Write-Log "FATAL ERROR: $_" -Level "ERROR"
    Write-Log "Stack Trace: $($_.ScriptStackTrace)" -Level "ERROR"
    exit 1
}