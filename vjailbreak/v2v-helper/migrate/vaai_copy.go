// Copyright © 2025 The vjailbreak authors

package migrate

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	cindervolumes "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/pkg/errors"
	"github.com/platform9/vjailbreak/pkg/vpwned/sdk/storage"
	esxissh "github.com/platform9/vjailbreak/v2v-helper/esxi-ssh"
	"github.com/platform9/vjailbreak/v2v-helper/pkg/k8sutils"
	"github.com/platform9/vjailbreak/v2v-helper/pkg/utils"
	"github.com/platform9/vjailbreak/v2v-helper/vcenter"
	"github.com/platform9/vjailbreak/v2v-helper/vm"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
)

// sanitizeVolumeName converts a volume name to meet storage array naming requirements:
// - 1-63 characters long
// - Alphanumeric, '_', and '-' only
// - Must begin and end with a letter or number
// - Must include at least one letter, '_', or '-'
func sanitizeVolumeName(name string) string {
	// Replace spaces and other invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	sanitized := reg.ReplaceAllString(name, "-")

	// Remove leading/trailing hyphens or underscores
	sanitized = strings.Trim(sanitized, "-_")

	// Ensure it starts and ends with alphanumeric
	sanitized = regexp.MustCompile(`^[^a-zA-Z0-9]+|[^a-zA-Z0-9]+$`).ReplaceAllString(sanitized, "")

	// Truncate to 63 characters if needed
	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
		// Re-trim trailing non-alphanumeric after truncation
		sanitized = regexp.MustCompile(`[^a-zA-Z0-9]+$`).ReplaceAllString(sanitized, "")
	}

	// If empty or too short, provide a default
	if len(sanitized) == 0 {
		sanitized = "disk-1"
	}

	return sanitized
}

// StorageAcceleratedCopyCopyDisks performs StorageAcceleratedCopy XCOPY-based disk copy for all VM disks
// This offloads the copy operation to the storage array, which is much faster than NBD
func (migobj *Migrate) StorageAcceleratedCopyCopyDisks(ctx context.Context, vminfo vm.VMInfo) ([]storage.Volume, error) {
	migobj.logMessage("Starting StorageAcceleratedCopy XCOPY-based disk copy")

	// Validate prerequisites
	if migobj.StorageProvider == nil {
		return []storage.Volume{}, fmt.Errorf("storage provider not initialized for StorageAcceleratedCopy copy")
	}

	// Get ESXi host information
	host, err := migobj.getESXiHost(ctx)
	if err != nil {
		return []storage.Volume{}, errors.Wrap(err, "failed to get ESXi host")
	}

	hostIP, err := migobj.getHostIPAddress(ctx, host)
	if err != nil {
		return []storage.Volume{}, errors.Wrap(err, "failed to get ESXi host IP")
	}

	migobj.logMessage(fmt.Sprintf("ESXi host: %s (IP: %s)", host.Name(), hostIP))

	// Connect to ESXi via SSH
	esxiClient := esxissh.NewClient()
	defer esxiClient.Disconnect()

	// TODO: For now hardcode "root", give option to pass user via configmap
	migobj.logMessage("Connecting to ESXi host via SSH")
	if err := esxiClient.Connect(ctx, hostIP, "root", migobj.ESXiSSHPrivateKey); err != nil {
		return []storage.Volume{}, errors.Wrap(err, "failed to connect to ESXi via SSH")
	}

	// Test the connection
	migobj.logMessage("Testing ESXi connection")
	if err := esxiClient.TestConnection(); err != nil {
		return []storage.Volume{}, errors.Wrap(err, "failed to test ESXi connection")
	}

	migobj.logMessage("Connected to ESXi host via SSH")

	// Verify VM is powered off before attempting StorageAcceleratedCopy copy
	// The VM should already be powered off by the migration flow before calling this function
	if vminfo.State != "poweredOff" {
		migobj.logMessage(fmt.Sprintf("VM %s is not powered off (state: %s). VM must be powered off before storage copy can proceed", vminfo.Name, vminfo.State))
		migobj.logMessage("Powering off VM")
		if err := migobj.VMops.VMPowerOff(); err != nil {
			return []storage.Volume{}, errors.Wrap(err, "failed to power off VM")
		}
		migobj.logMessage("VM powered off successfully")
	}

	migobj.logMessage(fmt.Sprintf("VM %s is powered off, proceeding with StorageAcceleratedCopy copy", vminfo.Name))

	// Wait for ESXi to release file locks on VMDK files after VM power off
	// This is necessary to avoid "Failed to lock the file" errors during vmkfstools clone
	migobj.logMessage("Waiting 5 seconds for ESXi to release disk file locks...")
	time.Sleep(5 * time.Second)

	volumes := []storage.Volume{}

	// Process each disk
	for idx, vmdisk := range vminfo.VMDisks {
		migobj.logMessage(fmt.Sprintf("Processing disk %d/%d: %s", idx+1, len(vminfo.VMDisks), vmdisk.Name))

		// Perform StorageAcceleratedCopy copy for this disk
		clonedVolume, err := migobj.copyDiskViaStorageAcceleratedCopy(ctx, esxiClient, idx, &vminfo, hostIP)
		if err != nil {
			return []storage.Volume{}, errors.Wrapf(err, "failed to copy disk %s via StorageAcceleratedCopy", vmdisk.Name)
		}

		// Update the disk with the OpenStack volume info from the cloned volume
		vminfo.VMDisks[idx].OpenstackVol = &cindervolumes.Volume{
			ID:   clonedVolume.OpenstackVol.ID,
			Name: clonedVolume.Name,
			Size: int(clonedVolume.Size / (1024 * 1024 * 1024)), // Convert bytes to GB
		}
		migobj.logMessage(fmt.Sprintf("Updated disk %s with Cinder volume ID: %s", vmdisk.Name, clonedVolume.OpenstackVol.ID))

		// Attach the Cinder volume to get the device path
		devicePath, err := migobj.AttachVolume(ctx, vminfo.VMDisks[idx])
		if err != nil {
			return []storage.Volume{}, errors.Wrapf(err, "failed to attach volume for disk %s", vmdisk.Name)
		}
		vminfo.VMDisks[idx].Path = devicePath
		volumes = append(volumes, clonedVolume)

		migobj.logMessage(fmt.Sprintf("Successfully copied disk %s via StorageAcceleratedCopy XCOPY", vmdisk.Name))
	}

	migobj.logMessage("StorageAcceleratedCopy XCOPY-based disk copy completed successfully")
	return volumes, nil
}

// copyDiskViaStorageAcceleratedCopy copies a single disk using StorageAcceleratedCopy XCOPY
func (migobj *Migrate) copyDiskViaStorageAcceleratedCopy(ctx context.Context, esxiClient *esxissh.Client,
	idx int, vminfo *vm.VMInfo, hostIP string,
) (storage.Volume, error) {
	startTime := time.Now()

	vmDisk := vminfo.VMDisks[idx]

	defer func() {
		migobj.logMessage(fmt.Sprintf("StorageAcceleratedCopy XCOPY completed in %s (total: %s) for disk %s",
			time.Since(startTime).Round(time.Second), time.Since(startTime).Round(time.Second), vmDisk.Name))
		migobj.StorageProvider.Disconnect()
	}()
	// Step 1: Initialize storage provider
	migobj.InitializeStorageProvider(ctx)

	// Step 2: Get ESXi host IQN for volume mapping
	hostIQN, err := esxiClient.GetHostIQN()
	if err != nil {
		return storage.Volume{}, errors.Wrap(err, "failed to get ESXi host IQN")
	}
	migobj.logMessage(fmt.Sprintf("ESXi host IQN: %s", hostIQN))

	// Step 3: Map host IQN to initiator group
	initiatorGroup := fmt.Sprintf("vjailbreak-xcopy")
	migobj.logMessage(fmt.Sprintf("Creating/updating initiator group: %s", initiatorGroup))
	mappingContext, err := migobj.StorageProvider.CreateOrUpdateInitiatorGroup(initiatorGroup, []string{hostIQN})
	if err != nil {
		return storage.Volume{}, errors.Wrapf(err, "failed to create initiator group %s", initiatorGroup)
	}

	// Step 4: Create target volume with sanitized name
	// Use vmDisk.Size (VMware disk size in bytes) - Pure API expects size in bytes
	diskSizeBytes := vmDisk.Size
	// Ensure size is a multiple of 512 (sector alignment)
	if diskSizeBytes%512 != 0 {
		diskSizeBytes = ((diskSizeBytes / 512) + 1) * 512
	}
	sanitizedName := sanitizeVolumeName(vminfo.Name + "-" + vmDisk.Name)
	migobj.logMessage(fmt.Sprintf("Creating target volume %s (sanitized from: %s) with size %d bytes (%d GB)",
		sanitizedName, vmDisk.Name, diskSizeBytes, diskSizeBytes/(1024*1024*1024)))
	targetVolume, err := migobj.StorageProvider.CreateVolume(sanitizedName, diskSizeBytes)
	if err != nil {
		return storage.Volume{}, errors.Wrapf(err, "failed to create target volume %s", sanitizedName)
	}

	// Step 5: Cinder manage the volume FIRST
	// This renames the volume on Pure to volume-<cinder-id>-cinder
	migobj.logMessage(fmt.Sprintf("Cinder managing the volume %s", targetVolume.Name))
	cinderVolumeId, err := migobj.manageVolumeToCinder(ctx, targetVolume.Name, vmDisk)
	if err != nil {
		return storage.Volume{}, errors.Wrapf(err, "failed to Cinder manage volume %s", targetVolume.Name)
	}
	vminfo.VMDisks[idx].OpenstackVol = &cindervolumes.Volume{
		ID:   cinderVolumeId,
		Name: targetVolume.Name,
		Size: int(targetVolume.Size / (1024 * 1024 * 1024)), // Convert bytes to GB
	}

	// After Cinder manage, the volume name changes based on the backend driver:
	// - Pure: volume-<cinder-id>-cinder
	// - NetApp: /vol/<volume_path>/volume-<cinder-id> (includes the full LUN path)
	// We use wildcard search with volume-<cinder-id> prefix which matches both patterns
	cinderVolumeName := fmt.Sprintf("volume-%s", cinderVolumeId)
	migobj.logMessage(fmt.Sprintf("Volume renamed by Cinder to pattern: *%s*", cinderVolumeName))

	// Step 6: Map target volume to ESXi host using the NEW Cinder volume name
	migobj.logMessage(fmt.Sprintf("Mapping target volume %s to ESXi host", cinderVolumeName))
	targetVol := storage.Volume{
		Name: cinderVolumeName, // Use the Cinder-renamed volume name
		NAA:  targetVolume.NAA,
		Size: targetVolume.Size,
		OpenstackVol: storage.OpenstackVolume{
			ID: cinderVolumeId,
		},
	}
	_, err = migobj.StorageProvider.MapVolumeToGroup(initiatorGroup, targetVol, mappingContext)
	if err != nil {
		// Volume might already be mapped, log warning and continue
		migobj.logMessage(fmt.Sprintf("Warning: Failed to map target volume (may already be mapped): %v", err))
	}

	// Cleanup function to unmap after copy
	defer func() {
		migobj.logMessage("Cleaning up volume mappings")
		if err := migobj.StorageProvider.UnmapVolumeFromGroup(initiatorGroup, targetVol, mappingContext); err != nil {
			migobj.logMessage(fmt.Sprintf("Warning: Failed to unmap target volume: %v", err))
		}
	}()

	// Step 6: Rescan ESXi storage and wait for target volume to appear
	migobj.logMessage(fmt.Sprintf("Waiting for target volume %s to appear on ESXi", targetVolume.NAA))
	deviceTimeout := 2 * time.Minute
	if err := esxiClient.RescanStorageForDevice(targetVolume.NAA, deviceTimeout); err != nil {
		return storage.Volume{}, errors.Wrapf(err, "target device %s not visible on ESXi", targetVolume.NAA)
	}

	targetDevicePath := fmt.Sprintf("/vmfs/devices/disks/%s", targetVolume.NAA)
	migobj.logMessage(fmt.Sprintf("Target device is visible: %s", targetDevicePath))

	// Wait for device to be fully ready after rescan
	// ESXi needs time to fully initialize the device after it appears
	migobj.logMessage("Waiting 10 seconds for device to be fully initialized...")
	time.Sleep(10 * time.Second)

	// Step 7: Perform StorageAcceleratedCopy XCOPY clone directly to raw device (RDM format)
	// This clones directly to the raw device without needing a datastore
	migobj.logMessage(fmt.Sprintf("Starting StorageAcceleratedCopy XCOPY clone: %s -> %s (RDM)", vmDisk.Path, targetDevicePath))

	cloneStart := time.Now()
	task, err := esxiClient.StartVmkfstoolsRDMClone(vmDisk.Path, targetDevicePath)
	if err != nil {
		return storage.Volume{}, errors.Wrapf(err, "failed to start StorageAcceleratedCopy RDM clone for disk %s", vmDisk.Name)
	}

	// Step 8: Monitor clone progress
	tracker := esxissh.NewCloneTracker(esxiClient, task, idx, migobj)
	tracker.SetPollInterval(2 * time.Second)

	err = tracker.WaitForCompletion(ctx)
	if err != nil {
		return storage.Volume{}, errors.Wrapf(err, "Copy failed for disk %s", vmDisk.Name)
	}

	cloneDuration := time.Since(cloneStart)
	totalDuration := time.Since(startTime)

	migobj.logMessage(fmt.Sprintf("Copy completed in %s (total: %s) for disk %s",
		cloneDuration.Round(time.Second), totalDuration.Round(time.Second), vmDisk.Name))

	// Update the target volume with Cinder info
	targetVolume.OpenstackVol = storage.OpenstackVolume{
		ID: cinderVolumeId,
	}

	return targetVolume, nil
}

// ValidateStorageAcceleratedCopyPrerequisites validates that all prerequisites for StorageAcceleratedCopy copy are met
func (migobj *Migrate) ValidateStorageAcceleratedCopyPrerequisites(ctx context.Context) error {
	migobj.logMessage("Validating StorageAcceleratedCopy prerequisites")

	// Check storage provider
	if migobj.StorageProvider == nil {
		return fmt.Errorf("storage provider not initialized")
	}

	// Load ESXi SSH key from secret if not already loaded
	if len(migobj.ESXiSSHPrivateKey) == 0 {
		if err := migobj.LoadESXiSSHKey(ctx); err != nil {
			return errors.Wrap(err, "failed to load ESXi SSH private key")
		}
	}

	// Validate storage provider credentials
	if err := migobj.StorageProvider.ValidateCredentials(ctx); err != nil {
		return errors.Wrap(err, "storage provider credential validation failed")
	}

	migobj.logMessage("StorageAcceleratedCopy prerequisites validated successfully")
	return nil
}

// getESXiHost returns the ESXi host object for the VM
func (migobj *Migrate) getESXiHost(ctx context.Context) (*object.HostSystem, error) {
	vm := migobj.VMops.GetVMObj()
	var vmProps mo.VirtualMachine
	err := vm.Properties(ctx, vm.Reference(), []string{"runtime.host"}, &vmProps)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get VM properties")
	}

	if vmProps.Runtime.Host == nil {
		return nil, fmt.Errorf("VM has no host")
	}

	// Access VCenterClient through the VMOps interface
	// VMops is of type vm.VMOperations interface, we need the concrete type
	// to access GetVCenterClient() which is not part of the interface
	type vcenterClientGetter interface {
		GetVCenterClient() *vcenter.VCenterClient
	}

	vcGetter, ok := migobj.VMops.(vcenterClientGetter)
	if !ok {
		return nil, fmt.Errorf("VMops does not implement GetVCenterClient()")
	}

	host := object.NewHostSystem(vcGetter.GetVCenterClient().VCClient, *vmProps.Runtime.Host)
	return host, nil
}

// getHostIPAddress returns the management IP address of an ESXi host
func (migobj *Migrate) getHostIPAddress(ctx context.Context, host *object.HostSystem) (string, error) {
	var hostProps mo.HostSystem
	err := host.Properties(ctx, host.Reference(), []string{"config.network"}, &hostProps)
	if err != nil {
		return "", errors.Wrap(err, "failed to get host properties")
	}

	if hostProps.Config == nil || hostProps.Config.Network == nil {
		return "", fmt.Errorf("host has no network configuration")
	}

	// Get management IP from vNICs
	for _, vnic := range hostProps.Config.Network.Vnic {
		if vnic.Spec.Ip != nil && vnic.Spec.Ip.IpAddress != "" {
			// Prefer management network
			if strings.Contains(strings.ToLower(vnic.Device), "vmk0") {
				return vnic.Spec.Ip.IpAddress, nil
			}
		}
	}

	// Fallback to first available IP
	if len(hostProps.Config.Network.Vnic) > 0 && hostProps.Config.Network.Vnic[0].Spec.Ip != nil {
		return hostProps.Config.Network.Vnic[0].Spec.Ip.IpAddress, nil
	}

	return "", fmt.Errorf("no management IP found for host")
}

// manageVolumeToCinder manages an existing storage array volume into Cinder
// Uses the ManageExistingVolume function which matches the tested RDM controller pattern
func (migobj *Migrate) manageVolumeToCinder(ctx context.Context, volumeName string, vmDisk vm.VMDisk) (string, error) {
	migobj.logMessage(fmt.Sprintf("Managing volume %s into Cinder", volumeName))

	// Get array creds mapping to find the correct ArrayCreds for this datastore
	arrayCredsMapping, err := k8sutils.GetArrayCredsMapping(ctx, migobj.K8sClient, migobj.ArrayCredsMapping)
	if err != nil {
		return "", errors.Wrap(err, "failed to get array creds mapping")
	}

	dataStoreName := vmDisk.Datastore
	arrayCredsName := ""
	for _, mapping := range arrayCredsMapping.Spec.Mappings {
		if mapping.Source == dataStoreName {
			arrayCredsName = mapping.Target
			break
		}
	}

	if arrayCredsName == "" {
		return "", fmt.Errorf("no array creds found for datastore %s", dataStoreName)
	}

	arrayCreds, err := k8sutils.GetArrayCreds(ctx, migobj.K8sClient, arrayCredsName)
	if err != nil {
		return "", errors.Wrap(err, "failed to get array creds")
	}

	// Build the Cinder host string - prefer autodiscovery
	backendName := arrayCreds.Spec.OpenStackMapping.CinderBackendName
	cinderHost := arrayCreds.Spec.OpenStackMapping.CinderHost

	// If CinderHost is not explicitly set, use autodiscovery to find the full host string
	if cinderHost == "" {
		if backendName == "" {
			return "", fmt.Errorf("neither CinderHost nor CinderBackendName specified in ArrayCreds")
		}

		// Autodiscover the full host string (uuid@backend) from Cinder services API
		migobj.logMessage(fmt.Sprintf("CinderHost not set, autodiscovering from backend name: %s", backendName))
		discoveredHost, err := migobj.autodiscoverCinderHost(ctx, backendName)
		if err != nil {
			return "", errors.Wrapf(err, "failed to autodiscover Cinder host for backend %s", backendName)
		}
		cinderHost = discoveredHost
		migobj.logMessage(fmt.Sprintf("Autodiscovered Cinder host: %s", cinderHost))
	} else {
		migobj.logMessage(fmt.Sprintf("Using configured Cinder host: %s", cinderHost))
	}

	volumeType := arrayCreds.Spec.OpenStackMapping.VolumeType

	// Volume reference for storage array - use source-name
	volumeRef := map[string]interface{}{
		"source-name": volumeName,
	}

	migobj.logMessage(fmt.Sprintf("Importing volume to Cinder: host=%s, type=%s, ref=%v", cinderHost, volumeType, volumeRef))

	// Use ManageExistingVolume which uses the manageable_volumes endpoint
	managedVolume, err := migobj.Openstackclients.ManageExistingVolume(volumeName, volumeRef, cinderHost, volumeType)
	if err != nil {
		return "", errors.Wrapf(err, "failed to import volume %s to Cinder", volumeName)
	}

	// Wait for volume to become available
	migobj.logMessage(fmt.Sprintf("Waiting for volume %s to become available", managedVolume.ID))
	if err := migobj.Openstackclients.WaitForVolume(ctx, managedVolume.ID); err != nil {
		return "", errors.Wrapf(err, "failed to wait for volume %s to become available", managedVolume.ID)
	}

	migobj.logMessage(fmt.Sprintf("Volume %s managed successfully with Cinder ID: %s", volumeName, managedVolume.ID))

	return managedVolume.ID, nil
}

// getCinderBackendForDatastore returns the Cinder backend host for a given datastore
func (migobj *Migrate) getCinderBackendForDatastore(datastoreName string) string {
	// TODO: This should look up the ArrayCredsMapping to find the correct backend
	// For now, return a placeholder that needs to be configured
	// Format is typically: hostname@backend#pool

	// This mapping should come from the ArrayCredsMapping CR
	// which maps datastores to ArrayCreds, which has the OpenStackMapping
	migobj.logMessage(fmt.Sprintf("Looking up Cinder backend for datastore: %s", datastoreName))

	// Placeholder - this needs to be implemented to read from ArrayCredsMapping
	// The actual value should come from ArrayCreds.Spec.OpenStackMapping.CinderBackendPool
	return fmt.Sprintf("hostname@%s#pool", migobj.VendorType)
}

// autodiscoverCinderHost queries the Cinder services API to find the actual host string for a backend
// Returns the full host format: "uuid@backend" or "hostname@backend"
func (migobj *Migrate) autodiscoverCinderHost(ctx context.Context, backendName string) (string, error) {
	migobj.logMessage(fmt.Sprintf("Autodiscovering Cinder host for backend: %s", backendName))

	// Get Cinder volume services via the interface method
	servicesInterface, err := migobj.Openstackclients.GetCinderVolumeServices(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to get Cinder volume services")
	}

	// Type assert to the concrete struct slice from utils package
	serviceList, ok := servicesInterface.([]utils.CinderVolumeService)
	if !ok {
		return "", fmt.Errorf("unexpected type from GetCinderVolumeServices: %T", servicesInterface)
	}

	migobj.logMessage(fmt.Sprintf("Found %d Cinder volume services", len(serviceList)))

	// Find the cinder-volume service for our backend
	for _, svc := range serviceList {
		// Only consider enabled and up services
		if svc.Status != "enabled" || svc.State != "up" {
			migobj.logMessage(fmt.Sprintf("Skipping service: host=%s, status=%s, state=%s", svc.Host, svc.Status, svc.State))
			continue
		}

		// Extract backend name from host (e.g., "55f61998-7b56-4f64-8527-2fdfaba63dcd@netapp" -> "netapp")
		parts := strings.Split(svc.Host, "@")
		if len(parts) == 2 && parts[1] == backendName {
			migobj.logMessage(fmt.Sprintf("Found Cinder host for backend %s: %s", backendName, svc.Host))
			return svc.Host, nil
		}
	}

	return "", fmt.Errorf("no active Cinder volume service found for backend: %s", backendName)
}
