# Firstboot-Scheduler.ps1
# The scheduler maintains a state table for each script and starts executing each script serially, 
# if the scheduler is interrupted by a reboot(during the installation of virtio drivers) the scheduler schedules itself on the next boot and continues from where it left off. 
# If a script fails after multiple reboots the scheduler stops. The script run by scheduler is wrapped in an exponential backoff mechanism, 
# which retries the script before failing completely and rescheduling the scheduler at next boot

$ScriptRoot = "C:\firstboot"
$LogFile = Join-Path $ScriptRoot "Firstboot-Scheduler.log"
$TaskName = "FirstbootSchedulerPostReboot"
$StateFilePath = Join-Path $ScriptRoot "Firstboot-Scheduler.state"
$SchedulerScriptPath = Join-Path $ScriptRoot "Firstboot-Scheduler.ps1"
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

    Write-Log "Detected 32-bit PowerShell on 64-bit OS → Relaunching in 64-bit PowerShell..." -Level "WARNING"

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

function Script-Runner {
    param(
        [string]$ScriptPath,
        [int]$MaxRetries = 3
    )
    $retryCount = 0
    $WaitTime = 60
    $retryCap = 300
    $result = @{
        ExitCode = 0
        Output = ""
        Success = $false
        Error = $null
    }
    
    try {
        Write-Log "Executing script: $ScriptPath"
        
        if (-not (Test-Path $ScriptPath)) {
            $result.ExitCode = 1
            $result.Error = "Script file not found: $ScriptPath"
            Write-Log $result.Error -Level "ERROR"
            return $result
        }
        while ($retryCount -lt $MaxRetries) {
        # Capture all output including errors
        $output = & $ScriptPath 2>&1
        $result.ExitCode = $LASTEXITCODE
        $result.Output = $output | Out-String
        
        if ($result.ExitCode -ne 0) {
            $result.Error = "attempt ($($retryCount+1)): Script failed with exit code $($result.ExitCode)"
            Write-Log $result.Error -Level "ERROR"
            Write-Log "Output: $($result.Output)" -Level "ERROR"
            Start-Sleep -Seconds $WaitTime
            $WaitTime *= 2
            if ($WaitTime -gt $retryCap) { $WaitTime = $retryCap }
            $retryCount++
        } else {
            $result.Success = $true
            Write-Log "Script executed successfully"
            Write-Log "Output: $($result.Output)"
            break
        }
        
        }    
    } catch {
        $result.ExitCode = 1
        $result.Error = "Exception occurred: $_"
        $result.Output = $_.Exception.Message
        Write-Log $result.Error -Level "ERROR"
    }
    
    return $result
}

function Push-Script {
    param(
        [string]$ScriptName
    )
    # Check if file exists if it does then check if the scriptName is already there or not the format in the file is Scriptname|Number get these two values 
    try {
        [int]$scriptRunTimes = -2
        if (Test-Path $StateFilePath) {
            $existingScripts = Get-Content -Path $StateFilePath 
            foreach ($existingScript in $existingScripts) {
                $entryName, [int]$entryNumber = $existingScript -split '\|'
                if ($entryName -eq $ScriptName) {
                    Write-Log "Script '$ScriptName' already exists in the state file with number '$entryNumber'."
                    if ($entryNumber -lt 3){
                        $scriptRunTimes = $entryNumber
                    }else{
                        Write-Log "Script '$ScriptName' has reached its maximum run times (3)."
                        throw "Script '$ScriptName' has reached its maximum run times (3)."
                    }
                    break
                }
            }
            if($scriptRunTimes -eq -2){
                throw "Script '$ScriptName' does not exist in the state file."
            }else{
                $originalRunTimes = $scriptRunTimes
                $scriptRunTimes = $scriptRunTimes + 1
                $updated = $existingScripts -replace "$ScriptName\|$originalRunTimes", "$ScriptName|$scriptRunTimes"
                Set-Content -Path $StateFilePath -Value $updated
            }
        }else{
            throw "State file does not exist."
        }
    }catch{
        Write-Log "Failed to push script: $_" -Level "ERROR"
        throw $_
    }
}
function Pop-Script{
    param(
        [string]$ScriptName
    )
    try {
        if (Test-Path $StateFilePath) {
            (Get-Content $StateFilePath) -notmatch "^\s*$([regex]::Escape($ScriptName))\s*\|" | Set-Content $StateFilePath
        } else {
            throw "State file does not exist."
        }
    } catch {
        Write-Log "Failed to pop script: $_" -Level "ERROR"
        throw $_
    }
}
function Init-Table{
    $scriptsJsonPath = Join-Path $ScriptRoot "scripts.json"
    if (Test-Path $scriptsJsonPath) {
        Write-Log "Found scripts.json at: $scriptsJsonPath"
        New-Item -Path $StateFilePath -ItemType File -Force | Out-Null
        try {
            $scriptsContent = Get-Content -Path $scriptsJsonPath -Raw -ErrorAction Stop
            $scriptsArray = $scriptsContent | ConvertFrom-Json -ErrorAction Stop
            
            Write-Log "Successfully parsed scripts.json, found $(($scriptsArray | Measure-Object).Count) script(s)"
            
            foreach ($script in $scriptsArray) {
                Add-Content -Path $StateFilePath -Value "$script|-1"
            }
        } catch {
            Write-Log "Failed to parse scripts.json: $_" -Level "ERROR"
        }
    }
}
function Get-Script{
    try {
    if (Test-Path $StateFilePath){
        $stateEntries = Get-Content -Path $StateFilePath
        #validate
        foreach ($entry in $stateEntries){
            $entryName, [int]$entryNumber = $entry -split '\|'
            if ($entryName -ne "" -and $entryNumber -ge -1 -and $entryNumber -lt 3){
                Write-Log "Script Name: $entryName, Run Times: $entryNumber"
            } else {
                throw "Invalid script entry: $entry"
            }
        }
        foreach ($entry in $stateEntries) {
            $entryName, [int]$entryNumber = $entry -split '\|'
            if ($entryNumber -eq -1){
                return $entryName
            } 
            if ($entryNumber -ge 0 -and $entryNumber -lt 3){
                return $entryName
            }
        }
    return ""
    }else{
        throw "State file does not exist."
    }
    }catch{
        Write-Log "Failed to get script: $_" -Level "ERROR"
        throw $_
    }
}
try {
    Write-Log "=== Starting Firstboot Scheduler ==="
    Write-Log "Script Root: $ScriptRoot"
    Write-Log "Log File: $LogFile"
    Write-Log "PowerShell Version: $($PSVersionTable.PSVersion)"
    Write-Log "OS: $([Environment]::OSVersion.VersionString)"
    Write-Log "64-bit Process: $([Environment]::Is64BitProcess)"
    Write-Log "64-bit OS: $([Environment]::Is64BitOperatingSystem)"
    
    # Ensure 64-bit PowerShell
    Ensure-64BitPowerShell
    
    # Read and parse scripts.json as PowerShell object
    $scriptsJsonPath = Join-Path $ScriptRoot "scripts.json"
    
    if (Test-Path $scriptsJsonPath) {
        Write-Log "Found scripts.json at: $scriptsJsonPath"
        Schedule-MyTask -TaskName $TaskName -ScriptPath $SchedulerScriptPath -Description "Firstboot Scheduler" 
        try {
            Init-Table
            while ($true) {
                $script = Get-Script
                Write-Log "Selected script: $script"
                if ($script -ne "" -and $script -ne "False"){
                    Push-Script -ScriptName $script
                    $result = Script-Runner -ScriptPath (Join-Path $ScriptRoot $script)
                    if ($result.ExitCode -ne 0) {
                        Write-Log "Script '$script' failed with exit code $($result.ExitCode)" -Level "ERROR"
                        Write-Log "Output: $($result.Output)" -Level "ERROR"
                        break
                    } else {
                        Write-Log "Script '$script' executed successfully"
                        Write-Log "Output: $($result.Output)"
                        Pop-Script -ScriptName $script
                    }
                }else{
                    Write-Log "No scripts to run, exiting..."
                    Remove-MyTask -TaskName $TaskName
                    break
                }
            }

        } catch {
            Write-Log "Failed to parse scripts.json: $_" -Level "ERROR"
        }
    } else {
        Write-Log "scripts.json not found at: $scriptsJsonPath" -Level "WARNING"
    }
    
    
    Write-Log "=== Firstboot Scheduler completed successfully ==="
    exit 0
    
} catch {
    Write-Log "FATAL ERROR: $_" -Level "ERROR"
    Write-Log "Stack Trace: $($_.ScriptStackTrace)" -Level "ERROR"
    exit 1
}