package pure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/devans10/pugo/flasharray"
	"github.com/platform9/vjailbreak/pkg/vpwned/sdk/storage"
	"k8s.io/klog/v2"
)

const FlashProviderID = "624a9370"

func init() {
	storage.RegisterStorageProvider("pure", &PureStorageProvider{})
}

// PureStorageProvider implements StorageProvider for Pure Storage FlashArray
type PureStorageProvider struct {
	storage.BaseStorageProvider
	client *flasharray.Client
}

// Connect establishes connection to Pure Storage array
func (p *PureStorageProvider) Connect(ctx context.Context, accessInfo storage.StorageAccessInfo) error {
	p.AccessInfo = accessInfo
	p.Config = storage.VendorConfig{
		NAAPrefix: FlashProviderID,
		Name:      "Pure",
	}

	// Create Pure Storage client
	client, err := flasharray.NewClient(
		accessInfo.Hostname,             // target
		accessInfo.Username,             // username
		accessInfo.Password,             // password
		"",                              // apiToken
		"",                              // restVersion
		!accessInfo.SkipSSLVerification, // verifyHTTPS (opposite of skip)
		accessInfo.SkipSSLVerification,  // sslCert
		"",                              // userAgent
		map[string]string{},             // requestKwargs
	)
	if err != nil {
		return fmt.Errorf("failed to create Pure Storage client: %w", err)
	}

	p.client = client
	p.SetConnected(true)

	// Log array info
	array, err := p.client.Array.Get(nil)
	if err == nil {
		klog.Infof("Connected to Pure Array: %s, ID: %s", array.ArrayName, array.ID)
	}

	return nil
}

// Disconnect closes the connection
func (p *PureStorageProvider) Disconnect() error {
	p.SetConnected(false)
	return nil
}

// ValidateCredentials validates the credentials
func (p *PureStorageProvider) ValidateCredentials(ctx context.Context) error {
	if !p.GetConnected() {
		err := p.Connect(ctx, p.AccessInfo)
		if err != nil {
			return err
		}
	}

	// Try to get array info as validation
	_, err := p.client.Array.Get(nil)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}

// CreateVolume creates a new volume on the storage array
func (p *PureStorageProvider) CreateVolume(volumeName string, size int64) (storage.Volume, error) {
	volume, err := p.client.Volumes.CreateVolume(volumeName, int(size))
	if err != nil {
		return storage.Volume{}, fmt.Errorf("failed to create volume %s: %w", volumeName, err)
	}
	return storage.Volume{
		Name:         volume.Name,
		Size:         volume.Size,
		Id:           "", // Pure sdk doesn't provide volume ID
		SerialNumber: volume.Serial,
		NAA:          p.BuildNAA(volume.Serial),
	}, nil
}

// DeleteVolume deletes a volume from the storage array
func (p *PureStorageProvider) DeleteVolume(volumeName string) error {
	_, err := p.client.Volumes.DeleteVolume(volumeName)
	if err != nil {
		return fmt.Errorf("failed to delete volume %s: %w", volumeName, err)
	}
	return nil
}

// RenameVolume renames a volume on the Pure array to Cinder format
func (p *PureStorageProvider) RenameVolume(oldName, newName string) error {
	_, err := p.client.Volumes.RenameVolume(oldName, newName)
	if err != nil {
		return fmt.Errorf("failed to rename volume from %s to %s: %w", oldName, newName, err)
	}
	klog.Infof("Renamed Pure volume from %s to %s", oldName, newName)
	return nil
}

// GetVolumeInfo retrieves information about a volume from the storage array
func (p *PureStorageProvider) GetVolumeInfo(volumeName string) (storage.VolumeInfo, error) {
	v, err := p.client.Volumes.GetVolume(volumeName, nil)
	if err != nil {
		return storage.VolumeInfo{}, fmt.Errorf("failed to get volume %s: %w", volumeName, err)
	}
	return storage.VolumeInfo{
		Name:    v.Name,
		Size:    v.Size,
		Created: v.Created,
		NAA:     p.BuildNAA(v.Serial),
	}, nil
}

// ListAllVolumes retrieves all volumes from the Pure Storage array with their NAA identifiers
func (p *PureStorageProvider) ListAllVolumes() ([]storage.VolumeInfo, error) {
	volumes, err := p.client.Volumes.ListVolumes(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	var volumeInfos []storage.VolumeInfo
	for _, v := range volumes {
		volumeInfos = append(volumeInfos, storage.VolumeInfo{
			Name:    v.Name,
			Size:    v.Size,
			Created: v.Created,
			NAA:     p.BuildNAA(v.Serial),
		})
	}

	return volumeInfos, nil
}

// GetAllVolumeNAAs retrieves NAA identifiers for all volumes on the array
func (p *PureStorageProvider) GetAllVolumeNAAs() ([]string, error) {
	return p.BaseStorageProvider.GetAllVolumeNAAs(p.ListAllVolumes)
}

// CreateOrUpdateInitiatorGroup creates or updates an initiator group with the ESX adapters
// mapping esxi's hba adapters initiator group to the volume host in pure.
func (p *PureStorageProvider) CreateOrUpdateInitiatorGroup(initiatorGroupName string, hbaIdentifiers []string) (storage.MappingContext, error) {
	hosts, err := p.client.Hosts.ListHosts(nil)
	if err != nil {
		return nil, err
	}

	matchedHosts := []string{}

	for _, h := range hosts {
		klog.Infof("Checking host %s, iqns: %v, wwns: %v", h.Name, h.Iqn, h.Wwn)

		// Check IQNs (case-insensitive since IQNs can be presented with different casing)
		for _, iqn := range h.Iqn {
			if storage.ContainsIgnoreCase(hbaIdentifiers, iqn) {
				klog.Infof("Adding host %s to group (matched IQN: %s)", h.Name, iqn)
				matchedHosts = append(matchedHosts, h.Name)
				break
			}
		}
	}

	if len(matchedHosts) == 0 {
		return nil, fmt.Errorf("no hosts found matching any of the provided IQNs/FC adapters: %v", hbaIdentifiers)
	}

	return storage.MappingContext{"hosts": matchedHosts}, nil
}

// MapVolumeToGroup maps a volume to hosts (not groups in Pure's case)
func (p *PureStorageProvider) MapVolumeToGroup(initiatorGroupName string, targetVolume storage.Volume, context storage.MappingContext) (storage.Volume, error) {
	hostsVal, ok := context["hosts"]
	if !ok {
		return storage.Volume{}, errors.New("hosts not found in mapping context")
	}

	hosts, ok := hostsVal.([]string)
	if !ok || len(hosts) == 0 {
		return storage.Volume{}, errors.New("invalid or empty hosts list in mapping context")
	}

	for _, host := range hosts {
		klog.Infof("Connecting host %s to volume %s", host, targetVolume.Name)
		_, err := p.client.Hosts.ConnectHost(host, targetVolume.Name, nil)
		if err != nil {
			if strings.Contains(err.Error(), "Connection already exists.") {
				klog.Infof("Connection already exists for host %s and volume %s", host, targetVolume.Name)
				continue
			}
			return storage.Volume{}, fmt.Errorf("connect host %q to volume %q: %w", host, targetVolume.Name, err)
		}
	}

	return targetVolume, nil
}

// UnmapVolumeFromGroup unmaps a volume from hosts
func (p *PureStorageProvider) UnmapVolumeFromGroup(initiatorGroupName string, targetVolume storage.Volume, context storage.MappingContext) error {
	hostsVal, ok := context["hosts"]
	if !ok {
		return nil // No hosts to unmap
	}

	hosts, ok := hostsVal.([]string)
	if !ok || len(hosts) == 0 {
		return nil
	}

	for _, host := range hosts {
		klog.Infof("Disconnecting host %s from volume %s", host, targetVolume.Name)
		_, err := p.client.Hosts.DisconnectHost(host, targetVolume.Name)
		if err != nil {
			return fmt.Errorf("disconnect host %q from volume %q: %w", host, targetVolume.Name, err)
		}
	}

	return nil
}

// GetMappedGroups returns the hosts the volume is mapped to
// We don't use the host group feature in Pure, so this returns nil
func (p *PureStorageProvider) GetMappedGroups(targetVolume storage.Volume, context storage.MappingContext) ([]string, error) {
	// Pure doesn't use host groups for this use case
	// A host can't belong to two separate groups
	return nil, nil
}

// ResolveCinderVolumeToLUN resolves a Cinder volume name to a storage Volume/LUN
func (p *PureStorageProvider) ResolveCinderVolumeToLUN(volumeID string) (storage.Volume, error) {
	// Get volume by name
	// Pure driver adds prefix volume and suffix -cinder to the volume name
	volumeName := fmt.Sprintf("volume-%s-cinder", volumeID)
	v, err := p.client.Volumes.GetVolume(volumeName, nil)
	if err != nil {
		return storage.Volume{}, fmt.Errorf("failed to get volume %s: %w", volumeName, err)
	}

	klog.Infof("Resolved cinder volume %s to volume: %+v", volumeName, v)

	lun := storage.Volume{
		Name:         v.Name,
		SerialNumber: v.Serial,
		NAA:          p.BuildNAA(v.Serial),
	}

	return lun, nil
}

// GetVolumeFromNAA retrieves a Pure volume by its NAA identifier
func (p *PureStorageProvider) GetVolumeFromNAA(naaID string) (storage.Volume, error) {
	serial, err := p.ExtractSerialFromNAA(naaID)
	if err != nil {
		return storage.Volume{}, err
	}

	// List all volumes and find matching serial
	volumes, err := p.client.Volumes.ListVolumes(nil)
	if err != nil {
		return storage.Volume{}, fmt.Errorf("failed to list volumes: %w", err)
	}

	for _, v := range volumes {
		if strings.ToUpper(v.Serial) == serial {
			klog.Infof("Found Pure volume %s with serial %s matching NAA %s", v.Name, v.Serial, naaID)
			return storage.Volume{
				Name:         v.Name,
				Size:         v.Size,
				SerialNumber: v.Serial,
				NAA:          naaID,
			}, nil
		}
	}

	return storage.Volume{}, fmt.Errorf("no Pure volume found with NAA %s (serial: %s)", naaID, serial)
}

// WhoAmI returns the provider name
func (p *PureStorageProvider) WhoAmI() string {
	return "pure"
}
