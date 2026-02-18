// Copyright © 2024 The vjailbreak authors

package esxissh

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// ListDatastores returns all datastores on the ESXi host
// Note: Using vim-cmd instead of esxcli as it's more reliable (esxcli can hang on some hosts)
// This is primarily for testing/debugging - production code gets VMDK paths from vSphere API
func (c *Client) ListDatastores() ([]DatastoreInfo, error) {
	if c.sshClient == nil {
		return nil, fmt.Errorf("not connected to ESXi host")
	}

	// Use vim-cmd which is faster and more reliable than esxcli storage filesystem list
	output, err := c.ExecuteCommand("vim-cmd hostsvc/datastore/listsummary")
	if err != nil {
		return nil, fmt.Errorf("failed to list datastores: %w", err)
	}

	// Parse vim-cmd output format:
	// (vim.Datastore.Summary) {
	//    name = "datastore1",
	//    url = "/vmfs/volumes/...",
	//    capacity = 1099511627776,
	//    freeSpace = 549755813888,
	//    type = "VMFS",
	// }
	datastores := []DatastoreInfo{}
	lines := strings.Split(output, "\n")

	var currentDS *DatastoreInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "(vim.Datastore.Summary)") && strings.Contains(line, "{") {
			// New datastore entry - save previous one if exists
			// Only process if it's a real datastore entry (contains opening brace)
			if currentDS != nil {
				datastores = append(datastores, *currentDS)
			}
			currentDS = &DatastoreInfo{}
		} else if currentDS != nil {
			// Parse datastore properties
			if strings.HasPrefix(line, "name = ") {
				currentDS.Name = strings.Trim(strings.TrimPrefix(line, "name = "), "\",")
			} else if strings.HasPrefix(line, "url = ") {
				currentDS.Path = strings.Trim(strings.TrimPrefix(line, "url = "), "\",")
			} else if strings.HasPrefix(line, "capacity = ") {
				capacityStr := strings.Trim(strings.TrimPrefix(line, "capacity = "), ",")
				if capacity, err := strconv.ParseInt(capacityStr, 10, 64); err == nil {
					currentDS.Capacity = capacity
				}
			} else if strings.HasPrefix(line, "freeSpace = ") {
				freeStr := strings.Trim(strings.TrimPrefix(line, "freeSpace = "), ",")
				if free, err := strconv.ParseInt(freeStr, 10, 64); err == nil {
					currentDS.FreeSpace = free
				}
			} else if strings.HasPrefix(line, "type = ") {
				currentDS.Type = strings.Trim(strings.TrimPrefix(line, "type = "), "\",")
			}
		}
	}

	// Add last datastore if exists
	if currentDS != nil {
		datastores = append(datastores, *currentDS)
	}

	return datastores, nil
}

// GetDatastoreInfo returns information about a specific datastore
func (c *Client) GetDatastoreInfo(datastoreName string) (*DatastoreInfo, error) {
	datastores, err := c.ListDatastores()
	if err != nil {
		return nil, err
	}

	for _, ds := range datastores {
		if ds.Name == datastoreName {
			return &ds, nil
		}
	}

	return nil, fmt.Errorf("datastore %s not found", datastoreName)
}

// ListVMs returns all VMs on the ESXi host
func (c *Client) ListVMs() ([]VMInfo, error) {
	if c.sshClient == nil {
		return nil, fmt.Errorf("not connected to ESXi host")
	}

	// Use vim-cmd to list VMs
	output, err := c.ExecuteCommand("vim-cmd vmsvc/getallvms")
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	vms := []VMInfo{}
	lines := strings.Split(output, "\n")

	// Skip header line
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		// Parse line more carefully - fields are: Vmid  Name  File  Guest OS  Version
		// The File field is like: [datastore] vm-folder/vm-name.vmx
		// We need to extract from the full line, not just split by whitespace

		// Extract VM ID (first field)
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		vmID := fields[0]

		// Find the VMX path - it's everything between the VM name and the Guest OS
		// Look for the pattern [datastore] path/to/file.vmx
		// Updated regex to handle paths with spaces
		vmxPattern := regexp.MustCompile(`\[([^\]]+)\]\s+(.+\.vmx)`)
		matches := vmxPattern.FindStringSubmatch(line)

		if len(matches) < 3 {
			// Couldn't parse VMX path, skip this VM
			continue
		}

		datastore := matches[1]
		vmxFile := matches[2]
		vmxPath := fmt.Sprintf("[%s] %s", datastore, vmxFile)

		// Extract VM name - it's between the ID and the VMX path
		nameStart := strings.Index(line, vmID) + len(vmID)
		nameEnd := strings.Index(line, "["+datastore+"]")
		vmName := strings.TrimSpace(line[nameStart:nameEnd])

		vm := VMInfo{
			ID:        vmID,
			Name:      vmName,
			VMXPath:   vmxPath,
			Datastore: datastore,
			Path:      filepath.Dir(vmxFile),
		}

		vms = append(vms, vm)
	}

	return vms, nil
}

// GetVMInfo returns detailed information about a specific VM
func (c *Client) GetVMInfo(vmName string) (*VMInfo, error) {
	vms, err := c.ListVMs()
	if err != nil {
		return nil, err
	}

	for _, vm := range vms {
		if vm.Name == vmName {
			// Get disk information for this VM
			disks, err := c.GetVMDisks(vm.VMXPath)
			if err != nil {
				return nil, fmt.Errorf("failed to get disks for VM %s: %w", vmName, err)
			}
			vm.Disks = disks
			return &vm, nil
		}
	}

	return nil, fmt.Errorf("VM %s not found", vmName)
}

// GetVMDisks returns all disks for a VM given its VMX path
func (c *Client) GetVMDisks(vmxPath string) ([]DiskInfo, error) {
	if c.sshClient == nil {
		return nil, fmt.Errorf("not connected to ESXi host")
	}

	// Extract datastore and path
	var datastore, vmDir string
	if matches := regexp.MustCompile(`\[([^\]]+)\]\s+(.+)`).FindStringSubmatch(vmxPath); len(matches) == 3 {
		datastore = matches[1]
		vmDir = filepath.Dir(matches[2])
	} else {
		return nil, fmt.Errorf("invalid VMX path format: %s", vmxPath)
	}

	// List files in VM directory to find VMDK files
	vmPath := fmt.Sprintf("/vmfs/volumes/%s/%s", datastore, vmDir)
	output, err := c.ExecuteCommand(fmt.Sprintf("ls -lh %s/*.vmdk 2>/dev/null || true", vmPath))
	if err != nil {
		return nil, fmt.Errorf("failed to list VM disks: %w", err)
	}

	disks := []DiskInfo{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Skip descriptor files (flat.vmdk are the actual data files)
		if strings.Contains(line, "-flat.vmdk") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		// Extract filename (last field) - this is the full path from ls output
		diskPath := fields[len(fields)-1]
		filename := filepath.Base(diskPath)

		// Get detailed disk info
		diskInfo, err := c.GetDiskInfo(diskPath)
		if err != nil {
			// If we can't get detailed info, create basic info
			diskInfo = &DiskInfo{
				Path:      diskPath,
				Name:      filename,
				Datastore: datastore,
			}
		}

		disks = append(disks, *diskInfo)
	}

	return disks, nil
}

// GetDiskInfo returns detailed information about a specific VMDK
func (c *Client) GetDiskInfo(diskPath string) (*DiskInfo, error) {
	if c.sshClient == nil {
		return nil, fmt.Errorf("not connected to ESXi host")
	}

	diskInfo := &DiskInfo{
		Path: diskPath,
		Name: filepath.Base(diskPath),
	}

	// Read VMDK descriptor to find the actual disk file and type
	descriptor, err := c.ExecuteCommand(fmt.Sprintf("cat %s 2>/dev/null || true", diskPath))
	if err == nil && descriptor != "" {
		// Parse descriptor for createType
		lowerDesc := strings.ToLower(descriptor)
		if strings.Contains(lowerDesc, "createtype") {
			if strings.Contains(lowerDesc, "monolithicsparse") || strings.Contains(lowerDesc, "thin") {
				diskInfo.ProvisionType = "thin"
			} else if strings.Contains(lowerDesc, "monolithicflat") {
				diskInfo.ProvisionType = "thick"
			} else if strings.Contains(lowerDesc, "eagerzeroedthick") {
				diskInfo.ProvisionType = "eagerzeroedthick"
			}
		}

		// Find the -flat.vmdk file reference to get actual size
		// Look for lines like: RW 8388608 VMFS "disk-flat.vmdk"
		for _, line := range strings.Split(descriptor, "\n") {
			if strings.Contains(line, "-flat.vmdk") || strings.Contains(line, "VMFS") {
				// Extract the extent size (in sectors)
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if sectors, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
						diskInfo.SizeBytes = sectors * 512 // 512 bytes per sector
						break
					}
				}
			}
		}
	}

	// If we still don't have a size, try to get the -flat.vmdk file size directly
	if diskInfo.SizeBytes == 0 {
		// Try to find the corresponding -flat.vmdk file
		flatPath := strings.Replace(diskPath, ".vmdk", "-flat.vmdk", 1)
		output, err := c.ExecuteCommand(fmt.Sprintf("ls -l %s 2>/dev/null || true", flatPath))
		if err == nil && output != "" {
			// Parse ls output: -rw-------    1 root     root     107374182400 Dec  6 10:30 disk-flat.vmdk
			fields := strings.Fields(output)
			if len(fields) >= 5 {
				if size, err := strconv.ParseInt(fields[4], 10, 64); err == nil {
					diskInfo.SizeBytes = size
				}
			}
		}
	}

	return diskInfo, nil
}

// ListStorageDevices returns all storage devices/LUNs visible to the ESXi host
func (c *Client) ListStorageDevices() ([]StorageDeviceInfo, error) {
	if c.sshClient == nil {
		return nil, fmt.Errorf("not connected to ESXi host")
	}

	// Use esxcli to list storage devices
	output, err := c.ExecuteCommand("esxcli storage core device list")
	if err != nil {
		return nil, fmt.Errorf("failed to list storage devices: %w", err)
	}

	devices := []StorageDeviceInfo{}
	lines := strings.Split(output, "\n")

	var currentDevice *StorageDeviceInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			// Empty line marks end of device entry
			if currentDevice != nil && currentDevice.DeviceID != "" {
				devices = append(devices, *currentDevice)
				currentDevice = nil
			}
			continue
		}

		// New device starts with a device ID line
		if !strings.Contains(line, ":") && (strings.HasPrefix(line, "naa.") || strings.HasPrefix(line, "t10.") || strings.HasPrefix(line, "mpx.")) {
			// Save previous device if exists
			if currentDevice != nil && currentDevice.DeviceID != "" {
				devices = append(devices, *currentDevice)
			}
			currentDevice = &StorageDeviceInfo{
				DeviceID: line,
			}
			continue
		}

		// Parse device properties
		if currentDevice != nil && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "Display Name":
				currentDevice.DisplayName = value
			case "Size":
				// Parse size - ESXi returns just a number in MB (e.g., "2048")
				if size, err := strconv.ParseInt(value, 10, 64); err == nil {
					// Size is in MB
					currentDevice.Size = size * 1024 * 1024
				}
			case "Device Type":
				currentDevice.DeviceType = value
			case "Vendor":
				currentDevice.Vendor = value
			case "Model":
				currentDevice.Model = value
			case "Is Local":
				currentDevice.IsLocal = (value == "true")
			case "Is SSD":
				currentDevice.IsSSD = (value == "true")
			case "Devfs Path":
				currentDevice.DevfsPath = value
			}
		}
	}

	// Add last device if exists
	if currentDevice != nil && currentDevice.DeviceID != "" {
		devices = append(devices, *currentDevice)
	}

	return devices, nil
}

// GetDatastoreBackingDevice returns the NAA device ID that backs a datastore
func (c *Client) GetDatastoreBackingDevice(datastoreName string) (string, error) {
	if c.sshClient == nil {
		return "", fmt.Errorf("not connected to ESXi host")
	}

	// Use esxcli to get datastore extent information
	output, err := c.ExecuteCommand(fmt.Sprintf("esxcli storage vmfs extent list | grep -A2 %s", datastoreName))
	if err != nil {
		return "", fmt.Errorf("failed to get datastore extent: %w", err)
	}

	// Parse output to extract NAA device
	// Format: datastore_name  uuid  partition  device_name  partition_num
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "naa.") {
			fields := strings.Fields(line)
			// Find the field that starts with "naa."
			for _, field := range fields {
				if strings.HasPrefix(field, "naa.") {
					return field, nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not find NAA device for datastore %s", datastoreName)
}

// GetVMDKBackingNAA returns the NAA device that backs a VMDK file
func (c *Client) GetVMDKBackingNAA(vmdkPath string) (string, error) {
	// Extract datastore name from VMDK path
	// Format: /vmfs/volumes/datastore-name/vm-folder/disk.vmdk or [datastore-name] vm-folder/disk.vmdk
	var datastoreName string

	if strings.HasPrefix(vmdkPath, "/vmfs/volumes/") {
		// Path format: /vmfs/volumes/datastore-name/...
		parts := strings.Split(strings.TrimPrefix(vmdkPath, "/vmfs/volumes/"), "/")
		if len(parts) > 0 {
			datastoreName = parts[0]
		}
	} else if strings.HasPrefix(vmdkPath, "[") {
		// Path format: [datastore-name] vm-folder/disk.vmdk
		endBracket := strings.Index(vmdkPath, "]")
		if endBracket > 0 {
			datastoreName = vmdkPath[1:endBracket]
		}
	}

	if datastoreName == "" {
		return "", fmt.Errorf("could not extract datastore name from VMDK path: %s", vmdkPath)
	}

	return c.GetDatastoreBackingDevice(datastoreName)
}

// PowerOffVM powers off a VM by its ID
func (c *Client) PowerOffVM(vmID string) error {
	if c.sshClient == nil {
		return fmt.Errorf("not connected to ESXi host")
	}

	// Check current power state first
	powerState, err := c.ExecuteCommand(fmt.Sprintf("vim-cmd vmsvc/power.getstate %s", vmID))
	if err != nil {
		return fmt.Errorf("failed to get power state: %w", err)
	}

	if strings.Contains(powerState, "Powered off") {
		return nil // Already powered off
	}

	// Power off the VM
	_, err = c.ExecuteCommand(fmt.Sprintf("vim-cmd vmsvc/power.off %s", vmID))
	if err != nil {
		return fmt.Errorf("failed to power off VM: %w", err)
	}

	return nil
}

// PowerOnVM powers on a VM by its ID
func (c *Client) PowerOnVM(vmID string) error {
	if c.sshClient == nil {
		return fmt.Errorf("not connected to ESXi host")
	}

	// Check current power state first
	powerState, err := c.ExecuteCommand(fmt.Sprintf("vim-cmd vmsvc/power.getstate %s", vmID))
	if err != nil {
		return fmt.Errorf("failed to get power state: %w", err)
	}

	if strings.Contains(powerState, "Powered on") {
		return nil // Already powered on
	}

	// Power on the VM
	_, err = c.ExecuteCommand(fmt.Sprintf("vim-cmd vmsvc/power.on %s", vmID))
	if err != nil {
		return fmt.Errorf("failed to power on VM: %w", err)
	}

	return nil
}

// GetVMPowerState returns the power state of a VM
func (c *Client) GetVMPowerState(vmID string) (string, error) {
	if c.sshClient == nil {
		return "", fmt.Errorf("not connected to ESXi host")
	}

	powerState, err := c.ExecuteCommand(fmt.Sprintf("vim-cmd vmsvc/power.getstate %s", vmID))
	if err != nil {
		return "", fmt.Errorf("failed to get power state: %w", err)
	}

	if strings.Contains(powerState, "Powered on") {
		return "on", nil
	} else if strings.Contains(powerState, "Powered off") {
		return "off", nil
	} else if strings.Contains(powerState, "Suspended") {
		return "suspended", nil
	}

	return "unknown", nil
}

// RescanStorage rescans ESXi storage adapters to detect new volumes
func (c *Client) RescanStorage() error {
	if c.sshClient == nil {
		return fmt.Errorf("not connected to ESXi host")
	}

	klog.Info("Rescanning ESXi storage adapters...")

	// Method 1: Rescan all HBAs using esxcli
	output, err := c.ExecuteCommand("esxcli storage core adapter rescan --all")
	if err != nil {
		// "Scan already in progress" is not a real error, just means another scan is running
		if strings.Contains(output, "already in progress") {
			klog.Info("Storage rescan already in progress, waiting...")
			time.Sleep(5 * time.Second)
		} else {
			klog.Warningf("esxcli rescan failed (output: %s): %v", output, err)
		}
	}
	klog.Info("Storage rescan completed")
	return nil
}

// RescanStorageForDevice rescans storage and waits for a specific device to appear
func (c *Client) RescanStorageForDevice(naaID string, timeout time.Duration) error {
	if c.sshClient == nil {
		return fmt.Errorf("not connected to ESXi host")
	}

	// Clean the naaID - remove any whitespace or hidden characters
	naaID = strings.TrimSpace(naaID)

	devicePath := fmt.Sprintf("/vmfs/devices/disks/%s", naaID)
	klog.Infof("Waiting for device %s to appear (timeout: %s)", devicePath, timeout)
	klog.Infof("Looking for NAA ID: %q (len=%d)", naaID, len(naaID))

	startTime := time.Now()
	pollInterval := 5 * time.Second
	rescanInterval := 15 * time.Second
	lastRescan := time.Time{}

	for time.Since(startTime) < timeout {
		// Check if device exists by listing all devices and checking in Go
		// This is more reliable than shell grep which can have issues with special chars
		listCmd := "ls /vmfs/devices/disks/"
		output, err := c.ExecuteCommand(listCmd)

		if err == nil {
			// Check each line for exact match
			for _, line := range strings.Split(output, "\n") {
				device := strings.TrimSpace(line)
				if device == naaID {
					klog.Infof("Device %s is now visible (took %s)", naaID, time.Since(startTime).Round(time.Second))
					return nil
				}
			}
		}

		// Trigger rescan periodically
		if time.Since(lastRescan) >= rescanInterval {
			klog.Infof("Device %s not yet visible, triggering rescan...", naaID)
			_ = c.RescanStorage()
			lastRescan = time.Now()
		}

		time.Sleep(pollInterval)
	}

	// Final check - list available devices for debugging
	allDisks, _ := c.ExecuteCommand("ls /vmfs/devices/disks/")
	naaDevices := []string{}
	for _, line := range strings.Split(allDisks, "\n") {
		device := strings.TrimSpace(line)
		if strings.HasPrefix(device, "naa.") {
			naaDevices = append(naaDevices, device)
			// Check for exact match
			if device == naaID {
				klog.Infof("Device %s found in final check!", naaID)
				return nil
			}
			// Also check if our naaID is contained (in case of prefix issues)
			if strings.Contains(device, naaID) || strings.Contains(naaID, device) {
				klog.Infof("Partial match found: looking for %q, found %q", naaID, device)
			}
		}
	}

	klog.Infof("Available NAA devices after timeout (%d found):", len(naaDevices))
	for _, d := range naaDevices {
		klog.Infof("  - %q (len=%d)", d, len(d))
	}

	return fmt.Errorf("device %s not visible after %s", naaID, timeout)
}
func (c *Client) CreateDatastore(datastoreName, naaID string) error {
	if c.sshClient == nil {
		return fmt.Errorf("not connected to ESXi host")
	}

	devicePath := fmt.Sprintf("/vmfs/devices/disks/%s", naaID)

	klog.Infof("Creating VMFS datastore %s on device %s", datastoreName, devicePath)

	// Create partition table and VMFS filesystem
	// Using vmkfstools with -C to create VMFS datastore
	cmd := fmt.Sprintf("vmkfstools -C vmfs6 -S %s %s", datastoreName, devicePath)
	_, err := c.ExecuteCommand(cmd)
	if err != nil {
		return fmt.Errorf("failed to create datastore: %w", err)
	}

	klog.Infof("Created datastore %s", datastoreName)
	return nil
}

// GetDatastorePath returns the full datastore path for a given datastore name
func (c *Client) GetDatastorePath(datastoreName string) (string, error) {
	if c.sshClient == nil {
		return "", fmt.Errorf("not connected to ESXi host")
	}

	// List datastores and find the matching one
	output, err := c.ExecuteCommand("esxcli storage filesystem list")
	if err != nil {
		return "", fmt.Errorf("failed to list filesystems: %w", err)
	}

	// Parse output to find datastore path
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, datastoreName) {
			// Extract mount point - typically /vmfs/volumes/<uuid> or /vmfs/volumes/<name>
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return fields[0], nil
			}
		}
	}

	return "", fmt.Errorf("datastore %s not found", datastoreName)
}

// GetHostIQN returns the iSCSI IQN of the ESXi host
func (c *Client) GetHostIQN() (string, error) {
	if c.sshClient == nil {
		return "", fmt.Errorf("not connected to ESXi host")
	}

	// Use storage core adapter list which provides UID field containing IQN/NQN/FC WWN
	results, err := c.RunEsxcliCommand("storage", []string{"core", "adapter", "list"})
	if err != nil {
		return "", fmt.Errorf("failed to get storage adapter list: %w", err)
	}

	klog.Infof("Found %d storage adapters", len(results))

	for _, adapter := range results {
		uid, hasUID := adapter["UID"]
		linkState, hasLink := adapter["LinkState"]
		driver, hasDriver := adapter["Driver"]

		if !hasUID || !hasLink || !hasDriver {
			continue
		}

		uid = strings.ToLower(strings.TrimSpace(uid))
		klog.Infof("Adapter: Driver=%s, LinkState=%s, UID=%s", driver, linkState, uid)

		// Check if the UID is iSCSI IQN (could also check for FC or NVMe-oF)
		isIQN := strings.HasPrefix(uid, "iqn.")

		if (linkState == "link-up" || linkState == "online") && isIQN {
			klog.Infof("Found ESXi IQN: %s", uid)
			return uid, nil
		}
	}

	return "", fmt.Errorf("no iSCSI IQN found on ESXi host")
}
