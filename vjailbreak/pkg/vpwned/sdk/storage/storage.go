package storage

import (
	"context"
	"errors"
	"strings"
)

var storageProviders map[string]StorageProvider = make(map[string]StorageProvider)

// StorageProvider defines the interface for storage array operations
type StorageProvider interface {
	// Connect establishes connection to the storage array
	Connect(ctx context.Context, accessInfo StorageAccessInfo) error

	// Disconnect closes the connection to the storage array
	Disconnect() error

	// ValidateCredentials validates the credentials and connectivity
	ValidateCredentials(ctx context.Context) error

	// CreateVolume creates a new volume on the storage array
	CreateVolume(volumeName string, size int64) (Volume, error)

	// DeleteVolume deletes a volume from the storage array
	DeleteVolume(volumeName string) error

	// GetVolumeInfo retrieves information about a volume from the storage array
	GetVolumeInfo(volumeName string) (VolumeInfo, error)

	// ListAllVolumes retrieves all volumes from the storage array
	ListAllVolumes() ([]VolumeInfo, error)

	// GetAllVolumeNAAs retrieves NAA identifiers for all volumes on the array
	GetAllVolumeNAAs() ([]string, error)

	// CreateOrUpdateInitiatorGroup creates or updates an initiator group with the provided HBA identifiers.
	// Returns a MappingContext that contains provider-specific information needed for mapping.
	CreateOrUpdateInitiatorGroup(initiatorGroupName string, hbaIdentifiers []string) (MappingContext, error)

	// MapVolumeToGroup maps a target volume to an initiator group.
	// Uses the MappingContext from CreateOrUpdateInitiatorGroup.
	MapVolumeToGroup(initiatorGroupName string, targetVolume Volume, context MappingContext) (Volume, error)

	// UnmapVolumeFromGroup unmaps a target volume from an initiator group.
	UnmapVolumeFromGroup(initiatorGroupName string, targetVolume Volume, context MappingContext) error

	// GetMappedGroups retrieves the initiator groups a target volume is currently mapped to.
	GetMappedGroups(targetVolume Volume, context MappingContext) ([]string, error)

	// ResolveCinderVolumeToLUN resolves a persistent volume name to a storage Volume/LUN.
	ResolveCinderVolumeToLUN(volumeName string) (Volume, error)

	// WhoAmI returns the provider name
	WhoAmI() string
}

// MappingContext holds context information for volume mapping
// It's a flexible map to store provider-specific context
type MappingContext map[string]interface{}

// Volume represents a storage volume/LUN
type Volume struct {
	Name         string
	Size         int64
	Id           string
	SerialNumber string
	NAA          string // Network Address Authority identifier
	OpenstackVol OpenstackVolume
}

// OpenstackVolume represents a Cinder volume
type OpenstackVolume struct {
	ID string
}

// StorageAccessInfo holds connection information for storage arrays
type StorageAccessInfo struct {
	Hostname            string
	Username            string
	Password            string
	SkipSSLVerification bool
	VendorType          string
}

// ArrayInfo holds basic storage array information
type ArrayInfo struct {
	Name         string
	Model        string
	Version      string
	SerialNumber string
	VendorType   string
}

// VolumeInfo holds volume information
type VolumeInfo struct {
	Name    string
	Size    int64
	Created string
	NAA     string
}

// CapacityInfo holds capacity information
type CapacityInfo struct {
	TotalCapacity int64
	UsedCapacity  int64
	FreeCapacity  int64
}

// RegisterStorageProvider registers a storage provider
func RegisterStorageProvider(name string, provider StorageProvider) {
	storageProviders[strings.ToLower(name)] = provider
}

// DeleteStorageProvider removes a storage provider
func DeleteStorageProvider(name string) {
	delete(storageProviders, strings.ToLower(name))
}

// GetStorageProviders returns all registered provider names
func GetStorageProviders() []string {
	var names []string
	for name := range storageProviders {
		names = append(names, name)
	}
	return names
}

// GetStorageProvider retrieves a storage provider by name
func GetStorageProvider(name string) (StorageProvider, error) {
	provider, ok := storageProviders[strings.ToLower(name)]
	if !ok {
		return nil, errors.New("storage provider not found: " + name)
	}
	return provider, nil
}

// NewStorageProvider creates a new storage provider based on vendor type
func NewStorageProvider(vendorType string) (StorageProvider, error) {
	return GetStorageProvider(vendorType)
}
