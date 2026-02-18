package constants

import "time"

const (
	HotplugCPUKey       = "HOTPLUG_CPU"
	HotplugMemoryKey    = "HOTPLUG_MEMORY"
	HotplugCPUMaxKey    = "HOTPLUG_CPU_MAX"
	HotplugMemoryMaxKey = "HOTPLUG_MEMORY_MAX"
	RhelFirstBootScript = `#!/bin/bash
set -e
LOG_FILE="/var/log/network_fix.log"
echo "$(date '+%Y-%m-%d %H:%M:%S') - Starting network fix script" >> "$LOG_FILE"

if ! systemctl is-active NetworkManager >/dev/null 2>&1; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') - NetworkManager not active, attempting to start" >> "$LOG_FILE"
    systemctl start NetworkManager
    if ! systemctl is-active NetworkManager >/dev/null 2>&1; then
        echo "$(date '+%Y-%m-%d %H:%M:%S') - Failed to start NetworkManager, exiting" >> "$LOG_FILE"
        exit 1
    fi
fi
echo "$(date '+%Y-%m-%d %H:%M:%S') - NetworkManager is active" >> "$LOG_FILE"

nmcli con reload || {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - Warning: Reload failed, restarting NM" >> "$LOG_FILE"
    systemctl restart NetworkManager
    sleep 5
    nmcli con reload || echo "$(date '+%Y-%m-%d %H:%M:%S') - Error: Reload still failed" >> "$LOG_FILE"
}

OLD_CONNS=$(nmcli -t -f NAME,TYPE connection show | grep -v ':loopback' | cut -d: -f1)
if [ -z "$OLD_CONNS" ]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') - No existing connections found" >> "$LOG_FILE"
else
    echo "$(date '+%Y-%m-%d %H:%M:%S') - Found connections: $OLD_CONNS" >> "$LOG_FILE"
fi

for conn in $OLD_CONNS; do
    echo "$(date '+%Y-%m-%d %H:%M:%S') - Processing connection: $conn" >> "$LOG_FILE"
    nmcli con mod "$conn" ipv4.method auto ipv4.addresses "" ipv4.gateway "" 2>>"$LOG_FILE" || \
        echo "$(date '+%Y-%m-%d %H:%M:%S') - Error: Failed to modify IPv4 for $conn" >> "$LOG_FILE"
    nmcli con mod "$conn" ipv6.method auto ipv6.addresses "" ipv6.gateway "" 2>>"$LOG_FILE" || \
        echo "$(date '+%Y-%m-%d %H:%M:%S') - Error: Failed to modify IPv6 for $conn" >> "$LOG_FILE"
    nmcli con up "$conn" 2>>"$LOG_FILE" || \
        echo "$(date '+%Y-%m-%d %H:%M:%S') - Warning: Failed to activate $conn" >> "$LOG_FILE"
done

NEW_IFACES=$(ip link show | grep -o '^[0-9]\+: [a-zA-Z0-9]\+:' | cut -d ' ' -f2 | cut -d ':' -f1 | grep -v lo)
if [ -z "$NEW_IFACES" ]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') - No new interfaces detected" >> "$LOG_FILE"
else
    echo "$(date '+%Y-%m-%d %H:%M:%S') - Detected interfaces: $NEW_IFACES" >> "$LOG_FILE"
fi

for iface in $NEW_IFACES; do
    echo "$(date '+%Y-%m-%d %H:%M:%S') - Processing new interface: $iface" >> "$LOG_FILE"
    conn_name="$iface"
    if ! nmcli con show "$conn_name" >/dev/null 2>&1; then
        echo "$(date '+%Y-%m-%d %H:%M:%S') - Creating new connection for $iface" >> "$LOG_FILE"
        nmcli con add type ethernet con-name "$conn_name" ifname "$iface" ipv4.method auto ipv6.method auto 2>>"$LOG_FILE" || \
            echo "$(date '+%Y-%m-%d %H:%M:%S') - Error: Failed to create connection for $iface" >> "$LOG_FILE"
    fi
    nmcli con up "$conn_name" 2>>"$LOG_FILE" || \
        echo "$(date '+%Y-%m-%d %H:%M:%S') - Error: Failed to activate $iface" >> "$LOG_FILE"
done
echo "$(date '+%Y-%m-%d %H:%M:%S') - Network fix script completed" >> "$LOG_FILE"`

	MaxCPU = 9999999
	MaxRAM = 9999999

	// Number of intervals to wait for the volume to become available
	MaxIntervalCount = 60

	InspectOSCommand         = "inspect-os"
	LSBootCommand            = "ls /boot"
	XMLFileName              = "libxml.xml"
	MigrationSnapshotName    = "migration-snap"
	MaxHTTPRetryCount        = 5
	MaxVMActiveCheckCount    = 15
	VMActiveCheckInterval    = 20 * time.Second
	NamespaceMigrationSystem = "migration-system"
	TrueString               = "true"

	LogsDir = "/var/log/pf9"

	EventMessageConvertingDisk                    = "Converting disk"
	EventMessageWaitingForCutOverStart            = "Waiting for VM Cutover start time"
	EventMessageCopyingChangedBlocksWithIteration = "Copying changed blocks"
	EventMessageWaitingForDataCopyStart           = "Waiting for data copy start time"
	EventMessageDataCopyStart                     = "Data copy start time reached"
	EventMessageWaitingForAdminCutOver            = "Waiting for Admin Cutover conditions to be met"
	EventMessageMigrationSucessful                = "VM created successfully"
	EventMessageMigrationFailed                   = "Trying to perform cleanup"
	EventMessageCopyingDisk                       = "Copying disk"
	EventMessageFailed                            = "Failed to"
	EventDisconnect                               = "Disconnected network interfaces"

	// StorageAcceleratedCopy specific event messages
	EventMessageEsxiSSHConnect                       = "Connecting to ESXi"
	EventMessageEsxiSSHTest                          = "Testing ESXi connection"
	EventMessageEsxiConnected                        = "Connected to ESXi"
	EventMessageInitiatorGroup                       = "Creating/updating initiator group"
	EventMessageStorageAcceleratedCopyCreatingVolume = "Creating target volume"
	EventMessageStorageAcceleratedCopyCinderManage   = "Cinder managing the volume"
	EventMessageStorageAcceleratedCopyMappingVolume  = "Mapping target volume"
	EventMessageStorageAcceleratedCopyRescanStorage  = "Waiting for target volume"
	EventMessageStorageAcceleratedCopyTargetDevice   = "Target device is visible:"

	OSFamilyWindows = "windowsguest"
	OSFamilyLinux   = "linuxguest"

	PCDClusterNameNoCluster = "NO CLUSTER"

	// VCenterVMScanConcurrencyLimit is the limit for concurrency while scanning vCenter VMs
	VCenterVMScanConcurrencyLimit = 100

	// ConfigMap default values
	ChangedBlocksCopyIterationThreshold = 20
	PeriodicSyncInterval                = "1h"
	// VMActiveWaitIntervalSeconds is the interval to wait for vm to become active
	VMActiveWaitIntervalSeconds = 20

	// VMActiveWaitRetryLimit is the number of retries to wait for vm to become active
	VMActiveWaitRetryLimit = 15

	// VolumeAvailableWaitIntervalSeconds is the interval to wait for volume to become available
	VolumeAvailableWaitIntervalSeconds = 5

	// VolumeAvailableWaitRetryLimit is the number of retries to wait for volume to become available
	VolumeAvailableWaitRetryLimit = 15

	// DefaultMigrationMethod is the default migration method
	DefaultMigrationMethod = "cold"

	// VCenterScanConcurrencyLimit is the max number of vcenter scan pods
	VCenterScanConcurrencyLimit = 100

	// CleanupVolumesAfterConvertFailure is the default value for cleanup volumes after convert failure
	CleanupVolumesAfterConvertFailure = true

	// CleanupPortsAfterMigrationFailure is the default value for cleanup ports after migration failure
	CleanupPortsAfterMigrationFailure = false

	// PopulateVMwareMachineFlavors is the default value for populating VMwareMachine objects with OpenStack flavors
	PopulateVMwareMachineFlavors = true

	// ValidateRDMOwnerVMs is the default value for RDM owner VM validation
	ValidateRDMOwnerVMs = true

	// VjailbreakSettingsConfigMapName is the name of the vjailbreak settings configmap
	VjailbreakSettingsConfigMapName = "vjailbreak-settings"

	// VCenterLoginRetryLimit is the number of retries for vcenter login
	VCenterLoginRetryLimit = 1

	// OpenstackCredsRequeueAfterMinutes is the time to requeue after.
	OpenstackCredsRequeueAfterMinutes = 60

	// VMwareCredsRequeueAfterMinutes is the time to requeue after.
	VMwareCredsRequeueAfterMinutes = 60

	// PeriodicSyncMaxRetries is the max number of retries for CBT sync
	PeriodicSyncMaxRetries = 3

	// PeriodicSyncRetryCap is the max retry interval for CBT sync
	PeriodicSyncRetryCap = "3h"
	// ValidateRDMOwnerVMsKey is the key for enabling/disabling RDM owner VM validation
	ValidateRDMOwnerVMsKey = "VALIDATE_RDM_OWNER_VMS"

	// ESXiSSHSecretName is the name of the Kubernetes secret containing ESXi SSH private key
	ESXiSSHSecretName = "esxi-ssh-key"

	// AutoFstabUpdate is the default value for automatic fstab update
	AutoFstabUpdate = false
	// AutoFstabUpdateKey is the key for enabling/disabling automatic fstab update
	AutoFstabUpdateKey = "AUTO_FSTAB_UPDATE"

	// AutoPXEBootOnConversionDefault is the default value for automatic PXE boot during cluster conversion
	AutoPXEBootOnConversionDefault = false
	// AutoPXEBootOnConversionKey is the key for enabling/disabling automatic PXE boot during cluster conversion
	AutoPXEBootOnConversionKey = "AUTO_PXE_BOOT_ON_CONVERSION"

	// StorageCopyMethod is the default value for storage copy method
	StorageCopyMethod = "StorageAcceleratedCopy"

	// MaxPowerOffRetryLimit is the max number of retries for power off status check
	MaxPowerOffRetryLimit = 3

	// PowerOffRetryCap is the max retry interval for power off status check
	PowerOffRetryCap = 5 * time.Minute

	// V2VHelperPodCPURequest is the default CPU request for v2v-helper pod
	V2VHelperPodCPURequest = "1000m"
	// V2VHelperPodCPURequestKey is the key for v2v-helper pod CPU request
	V2VHelperPodCPURequestKey = "V2V_HELPER_POD_CPU_REQUEST"

	// V2VHelperPodMemoryRequest is the default memory request for v2v-helper pod
	V2VHelperPodMemoryRequest = "1Gi"
	// V2VHelperPodMemoryRequestKey is the key for v2v-helper pod memory request
	V2VHelperPodMemoryRequestKey = "V2V_HELPER_POD_MEMORY_REQUEST"

	// V2VHelperPodCPULimit is the default CPU limit for v2v-helper pod
	V2VHelperPodCPULimit = "2000m"
	// V2VHelperPodCPULimitKey is the key for v2v-helper pod CPU limit
	V2VHelperPodCPULimitKey = "V2V_HELPER_POD_CPU_LIMIT"

	// V2VHelperPodMemoryLimit is the default memory limit for v2v-helper pod
	V2VHelperPodMemoryLimit = "3Gi"
	// V2VHelperPodMemoryLimitKey is the key for v2v-helper pod memory limit
	V2VHelperPodMemoryLimitKey = "V2V_HELPER_POD_MEMORY_LIMIT"

	// V2VHelperPodEphemeralStorageRequest is the default ephemeral storage request for v2v-helper pod
	V2VHelperPodEphemeralStorageRequest = "3Gi"
	// V2VHelperPodEphemeralStorageRequestKey is the key for v2v-helper pod ephemeral storage request
	V2VHelperPodEphemeralStorageRequestKey = "V2V_HELPER_POD_EPHEMERAL_STORAGE_REQUEST"

	// V2VHelperPodEphemeralStorageLimit is the default ephemeral storage limit for v2v-helper pod
	V2VHelperPodEphemeralStorageLimit = "3Gi"
	// V2VHelperPodEphemeralStorageLimitKey is the key for v2v-helper pod ephemeral storage limit
	V2VHelperPodEphemeralStorageLimitKey = "V2V_HELPER_POD_EPHEMERAL_STORAGE_LIMIT"

	// NICRecoveryFirstBootScript is the first boot script for NIC recovery
	WindowsPersistFirstBootScript = `
	@echo off
setlocal EnableDelayedExpansion

:: ────────────────────────────────────────────────
::  Configuration
:: ────────────────────────────────────────────────
set "PS_SCRIPT=C:\NIC-Recovery\Orchestrate-NICRecovery.ps1"
set "LOGDIR=C:\NIC-Recovery"
set "LOGFILE=%LOGDIR%\NIC-Recovery-Orchestrate_%DATE:~-4%%DATE:~3,2%%DATE:~0,2%_%TIME:~0,2%%TIME:~3,2%.log"

:: Replace space in time with zero if hour < 10
set "LOGFILE=%LOGFILE: =0%"

:: ────────────────────────────────────────────────
::  Create log directory if missing
:: ────────────────────────────────────────────────
if not exist "%LOGDIR%\" (
    mkdir "%LOGDIR%" 2>nul
    if errorlevel 1 (
        echo ERROR: Cannot create log directory %LOGDIR%
        pause
        exit /b 1
    )
)

:: ────────────────────────────────────────────────
::  Header in log
:: ────────────────────────────────────────────────
echo [%DATE% %TIME%] ============================================== >> "%LOGFILE%"
echo [%DATE% %TIME%] Starting NIC Recovery Orchestration           >> "%LOGFILE%"
echo [%DATE% %TIME%] Script: %PS_SCRIPT%                           >> "%LOGFILE%"
echo [%DATE% %TIME%] Computer: %COMPUTERNAME%                      >> "%LOGFILE%"
echo [%DATE% %TIME%] User:     %USERNAME%                          >> "%LOGFILE%"
echo [%DATE% %TIME%] ============================================== >> "%LOGFILE%"

:: ────────────────────────────────────────────────
::  Check if PowerShell script exists
:: ────────────────────────────────────────────────
if not exist "%PS_SCRIPT%" (
    echo [%DATE% %TIME%] ERROR: PowerShell script not found at:     >> "%LOGFILE%"
    echo [%DATE% %TIME%]        %PS_SCRIPT%                         >> "%LOGFILE%"
    echo.
    echo ERROR: Script not found: %PS_SCRIPT%
    echo        Check path and try again.
    echo.
    pause
    exit /b 1
)

:: ────────────────────────────────────────────────
::  Self-elevate to Administrator if not already
:: ────────────────────────────────────────────────
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [%DATE% %TIME%] Requesting administrator rights...         >> "%LOGFILE%"
    echo.
    echo Requesting admin rights ─ please accept the UAC prompt...
    echo.

    powershell -NoProfile -ExecutionPolicy Bypass -Command ^
        "Start-Process cmd -ArgumentList '/c %~f0' -Verb RunAs" 2>nul

    exit /b
)

:: ────────────────────────────────────────────────
::  Now we are elevated ─ run the real PowerShell script
:: ────────────────────────────────────────────────
echo [%DATE% %TIME%] Running PowerShell script as Administrator...  >> "%LOGFILE%"
echo.                                                            >> "%LOGFILE%"

powershell.exe -NoProfile -ExecutionPolicy Bypass ^
    -Command "& '%PS_SCRIPT%' *>> '%LOGFILE%' 2>&1"

set PS_EXITCODE=%errorlevel%

echo.                                                            >> "%LOGFILE%"
echo [%DATE% %TIME%] PowerShell script finished.                  >> "%LOGFILE%"
echo [%DATE% %TIME%] Exit code: !PS_EXITCODE!                     >> "%LOGFILE%"

if !PS_EXITCODE! equ 0 (
    echo [%DATE% %TIME%] Result: SUCCESS                              >> "%LOGFILE%"
    echo.
    echo NIC Recovery orchestration completed.
    echo Log saved to:
    echo   %LOGFILE%
) else (
    echo [%DATE% %TIME%] Result: FAILED (exit code !PS_EXITCODE!)     >> "%LOGFILE%"
    echo.
    echo NIC Recovery script FAILED (exit code !PS_EXITCODE!).
    echo Check the log for details:
    echo   %LOGFILE%
)

echo.
echo [%DATE% %TIME%] Finished. Press any key to exit...           >> "%LOGFILE%"
pause >nul
exit /b !PS_EXITCODE!
	`
)
