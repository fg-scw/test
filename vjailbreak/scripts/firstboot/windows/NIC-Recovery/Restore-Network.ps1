# Restore-Network.ps1
$ErrorActionPreference = "Stop"
$LogFile = "C:\NIC-Recovery\Restore-Network.log"

function Write-Log {
    param(
        [string]$Message,
        [ValidateSet("INFO", "WARNING", "ERROR")]
        [string]$Level = "INFO"
    )
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logEntry = "$timestamp - $Level - $Message"
    
    try {
        $logEntry | Out-File -FilePath $LogFile -Append -Encoding utf8
    } catch {
        # If we can't write to the log file, write to the console as a fallback
        [Console]::Error.WriteLine("Failed to write to log file: $_")
    }
    
    # Also write to the appropriate console stream
    switch ($Level) {
        "ERROR" { [Console]::Error.WriteLine($logEntry) }
        default { [Console]::Out.WriteLine($logEntry) }
    }
}

try {
    Write-Log "=== Starting Network Configuration Restore ==="
    Write-Log "Waiting 15 seconds for network interfaces to initialize..."
    Start-Sleep -Seconds 15

    if (-not (Test-Path "C:\NIC-Recovery\netconfig.json")) {
        throw "Network configuration file not found at C:\NIC-Recovery\netconfig.json"
    }

    $configs = Get-Content "C:\NIC-Recovery\netconfig.json" -ErrorAction Stop | ConvertFrom-Json
    Write-Log "Found $(($configs | Measure-Object).Count) network interface(s) to configure"

    foreach ($cfg in $configs) {
        $alias = $cfg.InterfaceAlias
        $mac = $cfg.MACAddress
        $ipAddr = $cfg.IPAddress
        
        Write-Log "Configuring $alias (MAC: $mac, IP: $ipAddr)"

        # Normalize MAC for comparison (remove : and -)
        $macClean = $mac -replace '[:-]',''

        try {
            $nic = Get-NetIPAddress -IPAddress $ipAddr -ErrorAction Stop | Get-NetAdapter -ErrorAction Stop
            if (-not $nic) {
                throw "No network adapter found with IP: $ipAddr"
            }

            # In case multiple nics match (very rare), take first
            $nic = $nic | Select-Object -First 1

            Write-Log " - Current adapter name: $($nic.Name), Status: $($nic.Status), Link Speed: $($nic.LinkSpeed)"

            # Disable DHCP
            # Write-Log " - Disabling DHCP"
            # Set-NetIPInterface -InterfaceAlias $nic.Name -Dhcp Disabled -Confirm:$false -ErrorAction Stop

            # Remove existing IPv4 addresses
            $oldIPs = Get-NetIPAddress -InterfaceAlias $nic.Name -AddressFamily IPv4 -ErrorAction SilentlyContinue
            # if ($oldIPs) {
                # Write-Log " - Removing $($oldIPs.Count) existing IPv4 address(es)"
                # $oldIPs | Remove-NetIPAddress -Confirm:$false -ErrorAction Stop
            # }

            # Rename adapter if needed
            if ($nic.Name -ne $alias) {
                Write-Log " - Renaming adapter from '$($nic.Name)' to '$alias'"
                Rename-NetAdapter -Name $nic.Name -NewName $alias -ErrorAction Stop -Confirm:$false
                $nic = Get-NetAdapter -Name $alias -ErrorAction Stop
            }

            # Prepare static IP parameters
            # $params = @{
                # InterfaceAlias = $nic.Name
                # IPAddress      = $ipAddr
                # PrefixLength   = $cfg.PrefixLength
            # }

            # if ($cfg.Gateway) {
                # $params.DefaultGateway = $cfg.Gateway
                # Write-Log " - Setting IP: $ipAddr/$($cfg.PrefixLength) with gateway: $($cfg.Gateway)"
            # } else {
                # Write-Log " - Setting IP: $ipAddr/$($cfg.PrefixLength) (no gateway)"
            # }

            # Configure IP address
            # New-NetIPAddress @params -ErrorAction Stop

            # Configure DNS if specified
            # if ($cfg.DNSServers -and $cfg.DNSServers.Count -gt 0) {
                # $dnsServers = $cfg.DNSServers -join ", "
                # Write-Log " - Setting DNS servers: $dnsServers"
                # Set-DnsClientServerAddress -InterfaceAlias $nic.Name -ServerAddresses $cfg.DNSServers -ErrorAction Stop
            # }

            # Verify the configuration
            # $newIP = Get-NetIPAddress -InterfaceAlias $nic.Name -AddressFamily IPv4 -ErrorAction SilentlyContinue
            # if ($newIP) {
                # Write-Log " - Successfully configured $alias. Current IP: $($newIP.IPAddress)/$($newIP.PrefixLength)" -Level "INFO"
            # } else {
                # Write-Log " - Configuration applied but verification failed for $alias" -Level "WARNING"
            # }

        } catch {
            $errorMsg = "Failed to configure $alias (IP: $ipAddr): $_"
            Write-Log $errorMsg -Level "ERROR"
            Write-Log "Stack Trace: $($_.ScriptStackTrace)" -Level "ERROR"
            continue
        }
    }

    Write-Log "=== Network Configuration Restore Completed Successfully ==="
    exit 0

} catch {
    $errorMsg = "Fatal error during network configuration: $_"
    Write-Log $errorMsg -Level "ERROR"
    Write-Log "Stack Trace: $($_.ScriptStackTrace)" -Level "ERROR"
    exit 1
}