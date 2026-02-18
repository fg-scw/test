# Enhanced-Cleanup-v2.ps1
# Aim: Really remove ghost NICs + free up old connection names so Rename-NetAdapter works later

param(
    [switch]$WhatIf,
    [string]$LogFile = "C:\NIC-Recovery\Enhanced-Cleanup-v2.log"
)

function Write-Log {
    param([string]$Message)
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    try {
        $logDir = Split-Path -Path $LogFile -Parent
        if (-not (Test-Path $logDir)) { New-Item -ItemType Directory -Path $logDir -Force | Out-Null }
        "$timestamp - $Message" | Out-File -FilePath $LogFile -Append -Encoding utf8
    } catch {
        Write-Host "Failed to write log: $_"
    }
    Write-Host "$timestamp - $Message"
}

# Must be admin
if (-not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw "Script requires Administrator rights."
}

Write-Log "=== Starting Enhanced Network Cleanup v2 ==="

try {
    # ────────────────────────────────────────────────
    # 1. Rename conflicting visible adapters (your original logic - good)
    # ────────────────────────────────────────────────
    Write-Log "Renaming any existing conflicting interface names (vjb* or similar)..."
    $interfaces = Get-NetAdapter -ErrorAction SilentlyContinue |
                  Where-Object { $_.Name -match '^vjb|Ethernet|Local Area Connection|Wi-Fi' }

    foreach ($nic in $interfaces) {
        $newName = "tmp_" + [guid]::NewGuid().ToString().Substring(0,8)
        Write-Log "Renaming $($nic.Name) → $newName to avoid name conflicts"
        if (-not $WhatIf) {
            try {
                Rename-NetAdapter -Name $nic.Name -NewName $newName -ErrorAction Stop
                Write-Log "  → Success"
            } catch {
                Write-Log "  → Failed: $_"
            }
        }
    }

    # ────────────────────────────────────────────────
    # 2. Remove ghost / non-present devices using pnputil (most effective method)
    # ────────────────────────────────────────────────
    Write-Log "Removing ghost/non-present network devices via pnputil..."
    $ghosts = Get-PnpDevice -Class Net -ErrorAction SilentlyContinue |
              Where-Object { $_.Status -eq "Unknown" -or $_.Status -eq "Error" }

    if (-not $ghosts) {
        Write-Log "No ghost NICs found via Get-PnpDevice."
    } else {
        foreach ($dev in $ghosts) {
            $fname = if ($dev.FriendlyName) { $dev.FriendlyName } else { "Unknown" }
            $iid   = $dev.InstanceId
            Write-Log "Found ghost: $fname ($iid)"

            if (-not $WhatIf) {
                try {
                    $output = & pnputil.exe /remove-device "$iid" 2>&1
                    if ($LASTEXITCODE -eq 0) {
                        Write-Log "  → Removed successfully via pnputil"
                    } else {
                        Write-Log "  → pnputil failed: $output"
                    }
                } catch {
                    Write-Log "  → Exception during pnputil: $_"
                }
            } else {
                Write-Log "  [WhatIf] Would remove device $iid"
            }
        }
    }

    # ────────────────────────────────────────────────
    # 3. Clean stale network connection profiles / names
    #    (this is usually WHY rename fails even after ghosts are gone)
    # ────────────────────────────────────────────────
    Write-Log "Cleaning stale network profile names..."
    $oldProfiles = Get-ChildItem "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\NetworkList\Profiles" -ErrorAction SilentlyContinue |
                   Where-Object {
                       $p = Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue
                       $p.ProfileName -match "vjb|Ethernet|Local Area|Wi-Fi" -or
                       $p.Description -match "vjb|Ethernet|Local Area|Wi-Fi"
                   }

    foreach ($prof in $oldProfiles) {
        $name = (Get-ItemProperty $prof.PSPath).ProfileName
        Write-Log "Found stale profile: $name ($($prof.PSChildName))"

        if (-not $WhatIf) {
            try {
                Remove-Item $prof.PSPath -Recurse -Force -ErrorAction Stop
                Write-Log "  → Deleted profile key"
            } catch {
                Write-Log "  → Failed to delete profile: $_"
            }
        } else {
            Write-Log "  [WhatIf] Would delete profile $($prof.PSChildName)"
        }
    }

    # Optional: trigger hardware rescan (helps Windows forget ghosts faster)
    Write-Log "Triggering hardware rescan..."
    if (-not $WhatIf) {
        & "$env:SystemRoot\System32\pnputil.exe" /scan-devices | Out-Null
    }

    Write-Log "=== Cleanup finished ==="
    if ($WhatIf) { Write-Log "NOTE: WhatIf mode - no actual changes made" }
}
catch {
    Write-Log "Critical error: $_"
    Write-Log $_.ScriptStackTrace
    throw
}