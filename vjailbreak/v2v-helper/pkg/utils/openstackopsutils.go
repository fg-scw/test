package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	openstackpkg "github.com/platform9/vjailbreak/pkg/common/openstack"
	"github.com/platform9/vjailbreak/v2v-helper/pkg/constants"
	"github.com/platform9/vjailbreak/v2v-helper/pkg/k8sutils"
	"github.com/platform9/vjailbreak/v2v-helper/vm"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gophercloud "github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/volumeattach"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
)

type OpenStackClients struct {
	BlockStorageClient *gophercloud.ServiceClient
	ComputeClient      *gophercloud.ServiceClient
	NetworkingClient   *gophercloud.ServiceClient
	K8sClient          client.Client
	AuthURL, Tenant    string
}

type OpenStackMetadata struct {
	UUID string `json:"uuid"`
}

var (
	cachedMetadata *OpenStackMetadata
	metadataMutex  sync.RWMutex
)

func GetCurrentInstanceUUID() (string, error) {

	// Step 1. Path with a read lock
	// First Check if the data is already cached. This read lock allows multiple
	// Goroutines to read the cached data concurrently.

	metadataMutex.RLock()
	if cachedMetadata != nil {
		// If cached, return it immediately.
		defer metadataMutex.RUnlock()
		return cachedMetadata.UUID, nil
	}

	// Release the read lock before we get the Write lock.
	metadataMutex.RUnlock()

	// Step 2. Path with write lock
	// Acquire a write lock to fetch and cache the metadata.
	metadataMutex.Lock()
	defer metadataMutex.Unlock()

	// Check again if another goroutine has already fetched the metadata.

	if cachedMetadata != nil {
		return cachedMetadata.UUID, nil
	}

	// Step 4: Fetch metadata from the OpenStack metadata service

	client := retryablehttp.NewClient()
	client.RetryMax = 5
	client.Logger = nil
	req, err := retryablehttp.NewRequest("GET", "http://169.254.169.254/openstack/latest/meta_data.json", http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %s", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get response: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %s", err)
	}

	var metadata OpenStackMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return "", fmt.Errorf("failed to unmarshal metadata: %s", err)
	}
	cachedMetadata = &metadata
	return metadata.UUID, nil
}

// create a new volume
func (osclient *OpenStackClients) CreateVolume(ctx context.Context, name string, size int64, ostype string, uefi bool, volumetype string, setRDMLabel bool) (*volumes.Volume, error) {
	blockStorageClient := osclient.BlockStorageClient

	PrintLog(fmt.Sprintf("OPENSTACK API: Creating volume with name %s with size %d, for OS type %s, UEFI %v, volume type %s, authurl %s, tenant %s", name, size, ostype, uefi, volumetype, osclient.AuthURL, osclient.Tenant))
	opts := volumes.CreateOpts{
		VolumeType: volumetype,
		Size:       int(math.Ceil(float64(size) / (1024 * 1024 * 1024))),
		Name:       name,
	}

	// Add 1GB to the size to account for the extra space
	opts.Size += 1

	volume, err := volumes.Create(ctx, blockStorageClient, opts, nil).Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to create volume: %s", err)
	}

	err = osclient.WaitForVolume(ctx, volume.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for volume: %s", err)
	}
	volume, err = volumes.Get(ctx, osclient.BlockStorageClient, volume.ID).Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to get volume: %s", err)
	}
	PrintLog(fmt.Sprintf("Volume created successfully. current status %s", volume.Status))

	if uefi {
		err = osclient.SetVolumeUEFI(ctx, volume)
		if err != nil {
			return nil, fmt.Errorf("failed to set volume uefi: %s", err)
		}
	}

	if strings.ToLower(ostype) == constants.OSFamilyWindows {
		err = osclient.SetVolumeImageMetadata(ctx, volume, setRDMLabel)
		if err != nil {
			return nil, fmt.Errorf("failed to set volume image metadata: %s", err)
		}
	}

	err = osclient.EnableQGA(ctx, volume)
	if err != nil {
		return nil, err
	}

	return volume, nil
}

func (osclient *OpenStackClients) DeleteVolume(ctx context.Context, volumeID string) error {
	PrintLog(fmt.Sprintf("OPENSTACK API: Deleting volume with ID %s, authurl %s, tenant %s", volumeID, osclient.AuthURL, osclient.Tenant))
	err := volumes.Delete(ctx, osclient.BlockStorageClient, volumeID, volumes.DeleteOpts{}).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to delete volume: %s", err)
	}
	return nil
}

func (osclient *OpenStackClients) WaitForVolume(ctx context.Context, volumeID string) error {
	// Get vjailbreak settings
	vjailbreakSettings, err := k8sutils.GetVjailbreakSettings(ctx, osclient.K8sClient)
	if err != nil {
		return errors.Wrap(err, "failed to get vjailbreak settings")
	}
	PrintLog(fmt.Sprintf("OPENSTACK API: Waiting for volume %s to become available, authurl %s, tenant %s", volumeID, osclient.AuthURL, osclient.Tenant))
	for i := 0; i < vjailbreakSettings.VolumeAvailableWaitRetryLimit; i++ {
		volume, err := volumes.Get(ctx, osclient.BlockStorageClient, volumeID).Extract()
		if err != nil {
			return fmt.Errorf("failed to get volume: %s", err)
		}
		if volume.Status == "error" {
			return fmt.Errorf("volume %s is in error state", volumeID)
		}
		instanceID, err := GetCurrentInstanceUUID()
		if err != nil {
			return fmt.Errorf("failed to get instance ID: %s", err)
		}

		// Check if the volume is available from nova side as well
		server, err := servers.Get(ctx, osclient.ComputeClient, instanceID).Extract()
		if err != nil {
			return fmt.Errorf("failed to get server: %s", err)
		}

		// get the attachments from the server
		found := false
		attachments := server.AttachedVolumes
		for _, attachment := range attachments {
			if attachment.ID == volumeID {
				found = true
				break
			}
		}
		// Check if volume is available and there are no attachments to the volume
		if volume.Status == "available" && len(volume.Attachments) == 0 && !found {
			return nil
		}
		fmt.Printf("Volume %s is still attached to server retrying %d times\n", volumeID, i)
		time.Sleep(time.Duration(vjailbreakSettings.VolumeAvailableWaitIntervalSeconds) * time.Second) // Wait for 5 seconds before checking again
	}
	return fmt.Errorf("volume did not become available within %d seconds", vjailbreakSettings.VolumeAvailableWaitRetryLimit*vjailbreakSettings.VolumeAvailableWaitIntervalSeconds)
}

func (osclient *OpenStackClients) AttachVolumeToVM(ctx context.Context, volumeID string) error {
	instanceID, err := GetCurrentInstanceUUID()
	if err != nil {
		return fmt.Errorf("failed to get instance ID: %s", err)
	}
	PrintLog(fmt.Sprintf("OPENSTACK API: Attaching volume %s to VM %s, authurl %s, tenant %s", volumeID, instanceID, osclient.AuthURL, osclient.Tenant))

	vjailbreakSettings, err := k8sutils.GetVjailbreakSettings(ctx, osclient.K8sClient)
	if err != nil {
		return errors.Wrap(err, "failed to get vjailbreak settings")
	}
	for i := 0; i < vjailbreakSettings.VolumeAvailableWaitRetryLimit; i++ {
		_, err = volumeattach.Create(ctx, osclient.ComputeClient, instanceID, volumeattach.CreateOpts{
			VolumeID:            volumeID,
			DeleteOnTermination: false,
		}).Extract()
		if err == nil || strings.Contains(err.Error(), "already attached") {
			err = nil
			break
		}
		time.Sleep(time.Duration(vjailbreakSettings.VolumeAvailableWaitIntervalSeconds) * time.Second) // Wait for 5 seconds before checking again
	}
	if err != nil {
		return fmt.Errorf("failed to attach volume to VM: %s", err)
	}

	PrintLog(fmt.Sprintf("OPENSTACK API: Waiting for volume attachment for volume %s to VM %s, authurl %s, tenant %s", volumeID, instanceID, osclient.AuthURL, osclient.Tenant))
	err = osclient.WaitForVolumeAttachment(ctx, volumeID)
	if err != nil {
		return fmt.Errorf("failed to wait for volume attachment: %s", err)
	}

	return nil
}

func (osclient *OpenStackClients) FindDevice(volumeID string) (string, error) {
	files, err := os.ReadDir("/dev/disk/by-id/")
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %s", err)
	}

	for _, file := range files {
		if strings.Contains(file.Name(), volumeID[:18]) {
			devicePath, err := filepath.EvalSymlinks(filepath.Join("/dev/disk/by-id/", file.Name()))
			if err != nil {
				return "", fmt.Errorf("failed to evaluate symlink: %s", err)
			}

			return devicePath, nil
		}
	}

	return "", nil
}

func (osclient *OpenStackClients) WaitForVolumeAttachment(ctx context.Context, volumeID string) error {
	instanceID, err := GetCurrentInstanceUUID()
	if err != nil {
		return fmt.Errorf("failed to get instance ID: %s", err)
	}
	PrintLog(fmt.Sprintf("OPENSTACK API: Waiting for volume attachment for volume %s to VM %s, authurl %s, tenant %s", volumeID, instanceID, osclient.AuthURL, osclient.Tenant))
	for i := 0; i < constants.MaxIntervalCount; i++ {
		devicePath, _ := osclient.FindDevice(volumeID)
		if devicePath != "" {
			return nil
		}
		time.Sleep(5 * time.Second) // Wait for 5 seconds before checking again
	}
	return fmt.Errorf("volume attachment not found within %d seconds", constants.MaxIntervalCount*5)
}

func (osclient *OpenStackClients) DetachVolumeFromVM(ctx context.Context, volumeID string) error {
	instanceID, err := GetCurrentInstanceUUID()
	if err != nil {
		return fmt.Errorf("failed to get instance ID: %s", err)
	}
	PrintLog(fmt.Sprintf("OPENSTACK API: Detaching volume %s from VM %s, authurl %s, tenant %s", volumeID, instanceID, osclient.AuthURL, osclient.Tenant))

	for i := 0; i < constants.MaxIntervalCount; i++ {
		err = volumeattach.Delete(ctx, osclient.ComputeClient, instanceID, volumeID).ExtractErr()
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second) // Wait for 5 seconds before checking again
	}
	if err != nil && !strings.Contains(err.Error(), "is not attached") {
		return fmt.Errorf("failed to detach volume from VM: %s", err)
	}

	return nil
}

func (osclient *OpenStackClients) EnableQGA(ctx context.Context, volume *volumes.Volume) error {
	PrintLog(fmt.Sprintf("OPENSTACK API: Enabling QGA for volume %s, authurl %s, tenant %s", volume.ID, osclient.AuthURL, osclient.Tenant))
	options := volumes.ImageMetadataOpts{
		Metadata: map[string]string{
			"hw_qemu_guest_agent": "yes",
			"hw_video_model":      "virtio",
			"hw_pointer_model":    "usbtablet",
		},
	}
	err := volumes.SetImageMetadata(ctx, osclient.BlockStorageClient, volume.ID, options).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to detach volume from VM: %s", err)
	}
	return nil
}

func (osclient *OpenStackClients) SetVolumeUEFI(ctx context.Context, volume *volumes.Volume) error {
	PrintLog(fmt.Sprintf("OPENSTACK API: Setting UEFI for volume %s, authurl %s, tenant %s", volume.ID, osclient.AuthURL, osclient.Tenant))
	options := volumes.ImageMetadataOpts{
		Metadata: map[string]string{
			"hw_firmware_type": "uefi",
		},
	}
	err := volumes.SetImageMetadata(ctx, osclient.BlockStorageClient, volume.ID, options).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to set volume image metadata hw_firmware_type to uefi: %s", err)
	}
	return nil
}

func (osclient *OpenStackClients) SetVolumeImageMetadata(ctx context.Context, volume *volumes.Volume, setRDMLabel bool) error {
	options := volumes.ImageMetadataOpts{
		Metadata: map[string]string{
			"hw_disk_bus": "virtio",
			"os_type":     "windows",
		},
	}
	if setRDMLabel {
		options.Metadata["hw_scsi_model"] = "virtio-scsi"
	}
	err := volumes.SetImageMetadata(ctx, osclient.BlockStorageClient, volume.ID, options).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to set volume image metadata for windows: %s", err)
	}
	return nil
}

func (osclient *OpenStackClients) SetVolumeBootable(ctx context.Context, volume *volumes.Volume) error {
	PrintLog(fmt.Sprintf("OPENSTACK API: Setting volume %s as bootable, authurl %s, tenant %s", volume.ID, osclient.AuthURL, osclient.Tenant))
	options := volumes.BootableOpts{
		Bootable: true,
	}
	err := volumes.SetBootable(ctx, osclient.BlockStorageClient, volume.ID, options).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to set volume as bootable: %s", err)
	}
	return nil
}

func (osclient *OpenStackClients) GetClosestFlavour(ctx context.Context, cpu int32, memory int32) (*flavors.Flavor, error) {
	PrintLog(fmt.Sprintf("OPENSTACK API: Getting closest flavor for %d vCPUs and %d MB RAM, authurl %s, tenant %s", cpu, memory, osclient.AuthURL, osclient.Tenant))
	allPages, err := flavors.ListDetail(osclient.ComputeClient, nil).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list flavors: %s", err)
	}

	allFlavors, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract all flavors: %s", err)
	}

	PrintLog(fmt.Sprintf("Current requirements: %d CPUs and %d MB of RAM", cpu, memory))

	// Use the shared flavor selection logic (without GPU filtering for v2v-helper fallback)
	// Note: v2v-helper doesn't track GPU counts, so we pass 0 for both
	bestFlavor, err := openstackpkg.GetClosestFlavour(int(cpu), int(memory), 0, 0, allFlavors, false)
	if err != nil {
		PrintLog("No suitable flavor found without GPU.")
		return nil, errors.Wrap(err, "failed to get closest flavor")
	}

	PrintLog(fmt.Sprintf("The best flavor is:\nName: %s, ID: %s, RAM: %dMB, VCPUs: %d, Disk: %dGB\n",
		bestFlavor.Name, bestFlavor.ID, bestFlavor.RAM, bestFlavor.VCPUs, bestFlavor.Disk))

	return bestFlavor, nil
}

func (osclient *OpenStackClients) GetFlavor(ctx context.Context, flavorId string) (*flavors.Flavor, error) {
	PrintLog(fmt.Sprintf("OPENSTACK API: Getting flavor %s, authurl %s, tenant %s", flavorId, osclient.AuthURL, osclient.Tenant))
	flavor, err := flavors.Get(ctx, osclient.ComputeClient, flavorId).Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to get flavor: %s", err)
	}
	return flavor, nil
}

func (osclient *OpenStackClients) GetNetwork(ctx context.Context, networkname string) (*networks.Network, error) {
	PrintLog(fmt.Sprintf("OPENSTACK API: Fetching network %s, authurl %s, tenant %s", networkname, osclient.AuthURL, osclient.Tenant))
	allPages, err := networks.List(osclient.NetworkingClient, nil).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %s", err)
	}

	allNetworks, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract all networks: %s", err)
	}

	for _, network := range allNetworks {
		if network.Name == networkname {
			return &network, nil
		}
	}
	return nil, fmt.Errorf("network not found")
}

func (osclient *OpenStackClients) GetPort(ctx context.Context, portID string) (*ports.Port, error) {
	PrintLog(fmt.Sprintf("OPENSTACK API: Fetching port %s, authurl %s, tenant %s", portID, osclient.AuthURL, osclient.Tenant))
	port, err := ports.Get(ctx, osclient.NetworkingClient, portID).Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to get port: %s", err)
	}
	return port, nil
}

func (osclient *OpenStackClients) DeletePort(ctx context.Context, portID string) error {
	PrintLog(fmt.Sprintf("OPENSTACK API: Deleting port %s, authurl %s, tenant %s", portID, osclient.AuthURL, osclient.Tenant))
	err := ports.Delete(ctx, osclient.NetworkingClient, portID).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to delete port %s: %s", portID, err)
	}
	PrintLog(fmt.Sprintf("Successfully deleted port %s", portID))
	return nil
}

func (osclient *OpenStackClients) GetSubnet(ctx context.Context, subnetList []string, ip string) (*subnets.Subnet, error) {
	parsedIp := net.ParseIP(ip)
	if parsedIp == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}
	for _, subnet := range subnetList {
		sn, err := subnets.Get(ctx, osclient.NetworkingClient, subnet).Extract()
		if err != nil {
			return nil, fmt.Errorf("failed to get subnet: %s", err)
		}
		_, ipNet, err := net.ParseCIDR(sn.CIDR)
		if err != nil {
			return nil, fmt.Errorf("failed to parse CIDR %q for subnet %s : %w", sn.CIDR, sn.ID, err)
		}
		if ipNet.Contains(parsedIp) {
			PrintLog(fmt.Sprintf("Subnet %s contains IP %s", sn.ID, ip))
			return sn, nil
		}
	}
	return nil, fmt.Errorf("IP %s is not in any of the subnets %v", ip, subnetList)
}

func (osclient *OpenStackClients) CheckIfPortExists(ctx context.Context, ipEntries []vm.IpEntry, mac string, network *networks.Network, gatewayIP map[string]string) (*ports.Port, error) {

	pages, err := ports.List(osclient.NetworkingClient, ports.ListOpts{
		NetworkID:  network.ID,
		MACAddress: mac,
	}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %s", err)
	}

	portList, err := ports.ExtractPorts(pages)
	if err != nil {
		return nil, err
	}
	for _, port := range portList {
		if port.MACAddress == mac {
			if port.DeviceID != "" {
				return nil, fmt.Errorf("precheck failed: port %s (MAC %s) is already in use by device %s", port.ID, mac, port.DeviceID)
			}
			if len(port.FixedIPs) > 0 {
				fixedIps := []string{}
				for _, fixedIp := range port.FixedIPs {
					fixedIps = append(fixedIps, fixedIp.IPAddress)
				}
				contain_all := true
				for _, ipIdx := range ipEntries {
					if !slices.Contains(fixedIps, ipIdx.IP) {
						contain_all = false
					}
					subnetId, err := osclient.GetSubnet(ctx, network.Subnets, ipIdx.IP)
					if err != nil {
						return nil, fmt.Errorf("subnet not found for IP %s", ipIdx.IP)
					}
					gatewayIP[mac] = subnetId.GatewayIP
				}
				if !contain_all {
					return nil, fmt.Errorf("port conflict: a port with MAC %s already exists but has IPs %v, while IPs %v were requested", mac, fixedIps, ipEntries)
				}
				// Check if port is already active - cannot reuse active ports
				if port.Status == "ACTIVE" {
					return nil, errors.New("port is already active, VM might already been migrated or this IP is used by another VM")
				}
				PrintLog(fmt.Sprintf("Port with MAC address %s already exists and is available, ID: %s", mac, port.ID))
				return &port, nil
			} else if len(port.FixedIPs) == 0 && len(ipEntries) == 0 {
				// Check if port is already active - cannot reuse active ports
				if port.Status == "ACTIVE" {
					return nil, errors.New("port is already active, VM might already been migrated or this IP is used by another VM")
				}
				PrintLog(fmt.Sprintf("Port with MAC address %s already exists, ID: %s", mac, port.ID))
				return &port, nil
			} else {
				return nil, fmt.Errorf("port conflict: a port with MAC %s already exists but has IP %s, while IP %v was requested", mac, port.FixedIPs, ipEntries)
			}
		}
	}

	return nil, nil

}

func (osclient *OpenStackClients) GetCreateOpts(ctx context.Context, network *networks.Network, mac string, ipEntries []vm.IpEntry, vmname string, securityGroups []string, gatewayIP map[string]string) (ports.CreateOpts, error) {

	createOpts := ports.CreateOpts{
		Name:           "port-" + vmname,
		NetworkID:      network.ID,
		MACAddress:     mac,
		SecurityGroups: &securityGroups,
	}
	if len(ipEntries) > 0 {
		fixedIPs := make([]ports.IP, 0)
		for _, ipEntry := range ipEntries {
			subnetId, err := osclient.GetSubnet(ctx, network.Subnets, ipEntry.IP)
			if err != nil {
				return createOpts, fmt.Errorf("subnet not found for IP %s", ipEntry.IP)
			} else {
				gatewayIP[mac] = subnetId.GatewayIP
				PrintLog(fmt.Sprintf("IP %s is in subnet %s", ipEntry.IP, subnetId.ID))
				fixedIPs = append(fixedIPs, ports.IP{
					SubnetID:  subnetId.ID,
					IPAddress: ipEntry.IP,
				})
			}
		}
		createOpts.FixedIPs = fixedIPs
	} else if len(ipEntries) == 0 {
		PrintLog("Empty port on vcentre detected for mac " + mac)
		subnetID, err := subnets.Get(ctx, osclient.NetworkingClient, network.Subnets[0]).Extract()
		if err != nil {
			return createOpts, fmt.Errorf("subnet not found for network %s", network.ID)
		}
		gatewayIP[mac] = subnetID.GatewayIP
	}
	return createOpts, nil
}

func (osclient *OpenStackClients) ValidateAndCreatePort(ctx context.Context, network *networks.Network, mac string, ipPerMac map[string][]vm.IpEntry, vmname string, securityGroups []string, fallbackToDHCP bool, gatewayIP map[string]string) (*ports.Port, error) {
	PrintLog(fmt.Sprintf("OPENSTACK API: Creating port for network %s, authurl %s, tenant %s with MAC address %s and IP addresses %v", network.ID, osclient.AuthURL, osclient.Tenant, mac, ipPerMac[mac]))
	Existingport, err := osclient.CheckIfPortExists(ctx, ipPerMac[mac], mac, network, gatewayIP)
	if err != nil {
		return nil, err
	}
	if Existingport != nil {
		return Existingport, nil
	}
	PrintLog(fmt.Sprintf("Port with MAC address %s does not exist, creating new port, trying with same IP address: %v", mac, ipPerMac[mac]))

	// Check if subnet is valid to avoid panic.
	if len(network.Subnets) == 0 {
		return nil, fmt.Errorf("no subnets found for network: %s", network.ID)
	}

	createOpts, err := osclient.GetCreateOpts(ctx, network, mac, ipPerMac[mac], vmname, securityGroups, gatewayIP)
	if err != nil {
		if !fallbackToDHCP {
			return nil, errors.Wrapf(err, "failed to create port with static IP %v, and fallback to DHCP is disabled", ipPerMac[mac])
		} else {
			PrintLog(fmt.Sprintf("Could Not Use IP: %v, using DHCP to create Port", ipPerMac[mac]))
			return osclient.CreatePortWithDHCP(ctx, network, ipPerMac, mac, gatewayIP, createOpts)
		}
	}
	port, err := osclient.createPortLowLevel(ctx, createOpts)
	if err != nil {
		if !fallbackToDHCP {
			return nil, errors.Wrapf(err, "failed to create port with static IP %v, and fallback to DHCP is disabled", ipPerMac[mac])
		}
		PrintLog(fmt.Sprintf("Could Not Use IP: %v, using DHCP to create Port", ipPerMac[mac]))
		createOpts.FixedIPs = nil
		return osclient.CreatePortWithDHCP(ctx, network, ipPerMac, mac, gatewayIP, createOpts)
	}
	return port, nil
}

func (osclient *OpenStackClients) CreatePortWithDHCP(ctx context.Context, network *networks.Network, ipPerMac map[string][]vm.IpEntry, mac string, gatewayIP map[string]string, createOpts ports.CreateOpts) (*ports.Port, error) {

	dhcpPort, dhcpErr := osclient.createPortLowLevel(ctx, createOpts)

	if dhcpErr != nil {
		return nil, errors.Wrap(dhcpErr, "failed to create port with DHCP after static IP failed")
	}
	ipPerMac[mac] = []vm.IpEntry{}
	for _, iAddr := range dhcpPort.FixedIPs {
		dhcpSubnetId, err := osclient.GetSubnet(ctx, network.Subnets, iAddr.IPAddress)
		if err != nil {
			return nil, fmt.Errorf("subnet not found for IP %s", iAddr.IPAddress)
		}
		ipPerMac[mac] = append(ipPerMac[mac], vm.IpEntry{
			IP:     iAddr.IPAddress,
			Prefix: 0,
		})
		gatewayIP[mac] = dhcpSubnetId.GatewayIP
	}
	logMsg := "Port created with DHCP instead of static IP"
	if len(ipPerMac[mac]) > 0 {
		logMsg = fmt.Sprintf("Port created with DHCP instead of static IP %v", ipPerMac[mac][0])
	}
	PrintLog(fmt.Sprintf("%s. Port ID: %s", logMsg, dhcpPort.ID))
	return dhcpPort, nil
}

func (osclient *OpenStackClients) CreatePort(ctx context.Context, networkid *networks.Network, mac string, ip []string, vmname string, securityGroups []string, fallbackToDHCP bool, gatewayIP map[string]string) (*ports.Port, error) {
	// Convert ip []string to []vm.IpEntry
	ipEntries := make([]vm.IpEntry, len(ip))
	for i, ipAddr := range ip {
		ipEntries[i] = vm.IpEntry{IP: ipAddr}
	}

	createOpts, err := osclient.GetCreateOpts(ctx, networkid, mac, ipEntries, vmname, securityGroups, gatewayIP)
	if err != nil {
		if !fallbackToDHCP {
			return nil, errors.Wrapf(err, "failed to create port with static IP %v, and fallback to DHCP is disabled", ip)
		}
		PrintLog(fmt.Sprintf("Could Not Use IP: %v, using DHCP to create Port", ip))
		// Create with DHCP by removing fixed IPs
		createOpts.FixedIPs = nil
	}
	return osclient.createPortLowLevel(ctx, createOpts)
}

func (osclient *OpenStackClients) createPortLowLevel(ctx context.Context, createOpts ports.CreateOpts) (*ports.Port, error) {
	return ports.Create(ctx, osclient.NetworkingClient, createOpts).Extract()
}

func (osclient *OpenStackClients) CreateVM(ctx context.Context, flavor *flavors.Flavor, networkIDs, portIDs []string, vminfo vm.VMInfo, availabilityZone string, securityGroups []string, serverGroupID string, vjailbreakSettings k8sutils.VjailbreakSettings, useFlavorless bool) (*servers.Server, error) {
	uuid := ""
	bootableDiskIndex := 0
	for idx, disk := range vminfo.VMDisks {
		if disk.Boot {
			uuid = disk.OpenstackVol.ID
			bootableDiskIndex = idx
			break
		}
	}
	if uuid == "" {
		return nil, fmt.Errorf("unable to determine boot volume for VM: %s", vminfo.Name)
	}
	PrintLog(fmt.Sprintf("OPENSTACK API: Creating VM %s, authurl %s, tenant %s with flavor %s in availability zone %s", vminfo.Name, osclient.AuthURL, osclient.Tenant, flavor.ID, availabilityZone))

	// Create the server
	openstacknws := []servers.Network{}
	for idx := range networkIDs {
		openstacknws = append(openstacknws, servers.Network{
			UUID: networkIDs[idx],
			Port: portIDs[idx],
		})
	}
	serverCreateOpts := servers.CreateOpts{
		Name:           vminfo.Name,
		FlavorRef:      flavor.ID,
		Networks:       openstacknws,
		SecurityGroups: securityGroups,
	}

	if useFlavorless {
		PrintLog(fmt.Sprintf("Using flavorless provisioning. Adding hotplug metadata: CPU=%d, Memory=%dMB", vminfo.CPU, vminfo.Memory))
		serverCreateOpts.Metadata = map[string]string{
			constants.HotplugCPUKey:       fmt.Sprintf("%d", vminfo.CPU),
			constants.HotplugMemoryKey:    fmt.Sprintf("%d", vminfo.Memory),
			constants.HotplugCPUMaxKey:    fmt.Sprintf("%d", vminfo.CPU),
			constants.HotplugMemoryMaxKey: fmt.Sprintf("%d", vminfo.Memory),
		}
	}

	if availabilityZone != "" && !strings.Contains(availabilityZone, constants.PCDClusterNameNoCluster) {
		// for PCD, this will be set to cluster name
		serverCreateOpts.AvailabilityZone = availabilityZone
	}
	if len((vminfo.RDMDisks)) > 0 {
		if serverCreateOpts.Metadata == nil {
			serverCreateOpts.Metadata = map[string]string{}
		}
		serverCreateOpts.Metadata["hw_scsi_reservations"] = "true"
	}

	// Set up boot block device with BootIndex 0
	bootBlockDevice := servers.BlockDevice{
		DeleteOnTermination: false,
		DestinationType:     servers.DestinationVolume,
		SourceType:          servers.SourceVolume,
		UUID:                uuid,
		BootIndex:           0,
	}
	serverCreateOpts.BlockDevice = []servers.BlockDevice{bootBlockDevice}

	// Prepare scheduler hints for server group if specified
	var schedulerHints servers.SchedulerHintOptsBuilder
	if serverGroupID != "" {
		schedulerHints = servers.SchedulerHintOpts{
			Group: serverGroupID,
		}
		PrintLog(fmt.Sprintf("Applying server group ID %s to VM %s via scheduler hints", serverGroupID, vminfo.Name))
	} else {
		PrintLog(fmt.Sprintf("No server group specified for VM %s - using default scheduling", vminfo.Name))
	}

	for _, disk := range vminfo.RDMDisks {
		// Set the Nova API version to 2.6
		osclient.ComputeClient.Microversion = "2.60"
		blockDevice := servers.BlockDevice{
			DeleteOnTermination: false,
			DestinationType:     servers.DestinationVolume,
			SourceType:          servers.SourceVolume,
			UUID:                disk.Status.CinderVolumeID,
			DeviceType:          "lun",
			DiskBus:             "scsi",
			BootIndex:           -1,
		}
		serverCreateOpts.BlockDevice = append(serverCreateOpts.BlockDevice, blockDevice)
	}

	// Wait for disks to become available
	for _, disk := range vminfo.VMDisks {
		err := osclient.WaitForVolume(ctx, disk.OpenstackVol.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to wait for volume to become available: %s", err)
		}
	}

	server, err := servers.Create(ctx, osclient.ComputeClient, serverCreateOpts, schedulerHints).Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %s", err)
	}

	// Wait for server to become active
	for i := 0; i < vjailbreakSettings.VMActiveWaitRetryLimit; i++ {
		result, err := servers.Get(ctx, osclient.ComputeClient, server.ID).Extract()
		if err != nil {
			return nil, fmt.Errorf("failed to get server status: %s", err)
		}
		if result.Status == "ACTIVE" {
			break
		}
		if result.Status == "ERROR" {
			return nil, fmt.Errorf("server %s went into ERROR state", server.ID)
		}
		time.Sleep(time.Duration(vjailbreakSettings.VMActiveWaitIntervalSeconds) * time.Second)
	}

	PrintLog(fmt.Sprintf("Server created with ID: %s, Attaching Additional Disks", server.ID))

	for _, disk := range append(vminfo.VMDisks[:bootableDiskIndex], vminfo.VMDisks[bootableDiskIndex+1:]...) {
		_, err := volumeattach.Create(ctx, osclient.ComputeClient, server.ID, volumeattach.CreateOpts{
			VolumeID:            disk.OpenstackVol.ID,
			DeleteOnTermination: false,
		}).Extract()
		if err != nil {
			return nil, fmt.Errorf("failed to attach volume to VM: %s", err)
		}
	}
	return server, nil
}

func (osclient *OpenStackClients) WaitUntilVMActive(ctx context.Context, vmID string) (bool, error) {
	result, err := servers.Get(ctx, osclient.ComputeClient, vmID).Extract()
	if err != nil {
		return false, fmt.Errorf("failed to get server: %s", err)
	}
	if result.Status != "ACTIVE" {
		return false, fmt.Errorf("server is not active")
	}
	return true, nil
}

// ManageExistingVolume manages an existing volume on the storage backend into Cinder
// Uses the manageable_volumes endpoint which is the standard Cinder manage API
func (osclient *OpenStackClients) ManageExistingVolume(name string, ref map[string]interface{}, host string, volumeType string) (*volumes.Volume, error) {
	PrintLog(fmt.Sprintf("OPENSTACK API: Managing existing volume %s on host %s with type %s", name, host, volumeType))

	// Build the manage request payload
	// This matches the format used by the tested RDM disk controller
	volumePayload := map[string]interface{}{
		"volume": map[string]interface{}{
			"host":        host,
			"ref":         ref,
			"name":        name,
			"volume_type": volumeType,
			"description": "Volume managed by vjailbreak StorageAcceleratedCopy copy",
			"bootable":    false,
		},
	}

	PrintLog(fmt.Sprintf("OPENSTACK API: Manage volume payload: %+v", volumePayload))

	var result map[string]interface{}
	response, err := osclient.BlockStorageClient.Post(
		context.Background(),
		osclient.BlockStorageClient.ServiceURL("manageable_volumes"),
		volumePayload,
		&result,
		&gophercloud.RequestOpts{
			OkCodes:     []int{202}, // Accepted
			MoreHeaders: map[string]string{"OpenStack-API-Version": "volume 3.8"},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to manage existing volume: %w", err)
	}

	if response != nil && response.Body != nil {
		defer response.Body.Close()
	}

	// Extract volume from response
	volumeMap, ok := result["volume"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to extract volume from response: %+v", result)
	}

	// Marshal and unmarshal to convert to volumes.Volume struct
	volumeJSON, err := json.Marshal(volumeMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal volume map: %w", err)
	}

	var volume volumes.Volume
	if err := json.Unmarshal(volumeJSON, &volume); err != nil {
		return nil, fmt.Errorf("failed to unmarshal volume: %w", err)
	}

	PrintLog(fmt.Sprintf("OPENSTACK API: Successfully managed volume %s with ID %s", name, volume.ID))

	return &volume, nil
}

func (osclient *OpenStackClients) GetSecurityGroupIDs(ctx context.Context, groupNames []string, projectName string) ([]string, error) {
	if len(groupNames) == 0 {
		return nil, nil
	}

	if projectName == "" {
		return nil, fmt.Errorf("projectName is required for security group lookup")
	}

	//check if string is UUID
	isUUID := func(s string) bool {
		re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
		return re.MatchString(s)
	}

	//build a map name -> ID
	identityClient, err := openstack.NewIdentityV3(osclient.NetworkingClient.ProviderClient, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to create identity client: %w", err)
	}

	listOpts := projects.ListOpts{Name: projectName}
	allPages, err := projects.List(identityClient, listOpts).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects with name %s: %w", projectName, err)
	}
	allProjects, err := projects.ExtractProjects(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract projects: %w", err)
	}
	if len(allProjects) == 0 {
		return nil, fmt.Errorf("no project found with name %s", projectName)
	}
	projectID := allProjects[0].ID

	allPages, err = groups.List(osclient.NetworkingClient, groups.ListOpts{
		TenantID: projectID,
	}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list security groups: %w", err)
	}
	allGroups, err := groups.ExtractGroups(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract security groups: %w", err)
	}

	nameToIDMap := make(map[string]string)
	for _, group := range allGroups {
		nameToIDMap[group.Name] = group.ID
	}

	var groupIDs []string
	for _, g := range groupNames {
		if isUUID(g) {
			groupIDs = append(groupIDs, g)
			continue
		}

		id, found := nameToIDMap[g]
		if !found {
			return nil, fmt.Errorf("security group with name '%s' not found in project '%s'", g, projectName)
		}
		groupIDs = append(groupIDs, id)
	}

	return groupIDs, nil
}

func (osclient *OpenStackClients) GetServerGroups(ctx context.Context, projectName string) ([]vjailbreakv1alpha1.ServerGroupInfo, error) {
	PrintLog(fmt.Sprintf("OPENSTACK API: Fetching server groups for project %s", projectName))

	allPages, err := servergroups.List(osclient.ComputeClient, servergroups.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list server groups: %w", err)
	}

	allGroups, err := servergroups.ExtractServerGroups(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract server groups: %w", err)
	}

	var result []vjailbreakv1alpha1.ServerGroupInfo
	for _, group := range allGroups {
		result = append(result, vjailbreakv1alpha1.ServerGroupInfo{
			Name:    group.Name,
			ID:      group.ID,
			Policy:  strings.Join(group.Policies, ","),
			Members: len(group.Members),
		})
	}

	return result, nil
}

// CinderVolumeService represents a Cinder volume service
type CinderVolumeService struct {
	Host   string
	Status string
	State  string
}

// GetCinderVolumeServices returns the list of Cinder volume services
func (osclient *OpenStackClients) GetCinderVolumeServices(ctx context.Context) (interface{}, error) {
	PrintLog(fmt.Sprintf("OPENSTACK API: Fetching Cinder volume services, authurl %s, tenant %s", osclient.AuthURL, osclient.Tenant))

	// Query Cinder volume services using raw API call
	endpoint := osclient.BlockStorageClient.ServiceURL("os-services")

	type Service struct {
		Binary string `json:"binary"`
		Host   string `json:"host"`
		Status string `json:"status"`
		State  string `json:"state"`
	}

	type ServicesResponse struct {
		Services []Service `json:"services"`
	}

	var response ServicesResponse
	_, err := osclient.BlockStorageClient.Get(ctx, endpoint, &response, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query Cinder volume services: %w", err)
	}

	var result []CinderVolumeService
	for _, svc := range response.Services {
		if svc.Binary == "cinder-volume" {
			result = append(result, CinderVolumeService{
				Host:   svc.Host,
				Status: svc.Status,
				State:  svc.State,
			})
		}
	}

	return result, nil
}
