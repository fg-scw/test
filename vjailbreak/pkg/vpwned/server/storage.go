package server

import (
	"context"
	"fmt"

	api "github.com/platform9/vjailbreak/pkg/vpwned/api/proto/v1/service"
	storagesdk "github.com/platform9/vjailbreak/pkg/vpwned/sdk/storage"

	// Import all storage providers via providers hub
	_ "github.com/platform9/vjailbreak/pkg/vpwned/sdk/storage/providers"
	"github.com/sirupsen/logrus"
)

// storageArrayGRPC implements the StorageArray gRPC service
type storageArrayGRPC struct {
	api.UnimplementedStorageArrayServer
}

// ValidateCredentials validates storage array credentials
func (s *storageArrayGRPC) ValidateCredentials(ctx context.Context, req *api.ValidateStorageCredsRequest) (*api.ValidateStorageCredsResponse, error) {
	if req.AccessInfo == nil {
		return &api.ValidateStorageCredsResponse{
			Success: false,
			Message: "access_info is required",
		}, nil
	}

	logrus.Infof("Validating credentials for %s storage array at %s", req.AccessInfo.VendorType, req.AccessInfo.Hostname)

	// Get the storage provider
	provider, err := storagesdk.NewStorageProvider(req.AccessInfo.VendorType)
	if err != nil {
		return &api.ValidateStorageCredsResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get storage provider: %v", err),
		}, nil
	}

	// Create access info
	accessInfo := storagesdk.StorageAccessInfo{
		Hostname:            req.AccessInfo.Hostname,
		Username:            req.AccessInfo.Username,
		Password:            req.AccessInfo.Password,
		SkipSSLVerification: req.AccessInfo.SkipSslVerification,
		VendorType:          req.AccessInfo.VendorType,
	}

	// Connect to storage array
	if err := provider.Connect(ctx, accessInfo); err != nil {
		return &api.ValidateStorageCredsResponse{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}
	defer provider.Disconnect()

	// Validate credentials
	if err := provider.ValidateCredentials(ctx); err != nil {
		return &api.ValidateStorageCredsResponse{
			Success: false,
			Message: fmt.Sprintf("Validation failed: %v", err),
		}, nil
	}

	return &api.ValidateStorageCredsResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully validated %s storage array credentials", req.AccessInfo.VendorType),
	}, nil
}

// CreateOrUpdateInitiatorGroup creates or updates an initiator group
func (s *storageArrayGRPC) CreateOrUpdateInitiatorGroup(ctx context.Context, req *api.CreateInitiatorGroupRequest) (*api.CreateInitiatorGroupResponse, error) {
	if req.AccessInfo == nil {
		return &api.CreateInitiatorGroupResponse{
			Success: false,
			Message: "access_info is required",
		}, nil
	}

	logrus.Infof("Creating/updating initiator group %s for %s", req.InitiatorGroupName, req.AccessInfo.VendorType)

	provider, err := storagesdk.NewStorageProvider(req.AccessInfo.VendorType)
	if err != nil {
		return &api.CreateInitiatorGroupResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get storage provider: %v", err),
		}, nil
	}

	accessInfo := storagesdk.StorageAccessInfo{
		Hostname:            req.AccessInfo.Hostname,
		Username:            req.AccessInfo.Username,
		Password:            req.AccessInfo.Password,
		SkipSSLVerification: req.AccessInfo.SkipSslVerification,
		VendorType:          req.AccessInfo.VendorType,
	}

	if err := provider.Connect(ctx, accessInfo); err != nil {
		return &api.CreateInitiatorGroupResponse{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}
	defer provider.Disconnect()

	mappingContext, err := provider.CreateOrUpdateInitiatorGroup(req.InitiatorGroupName, req.HbaIdentifiers)
	if err != nil {
		return &api.CreateInitiatorGroupResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create/update initiator group: %v", err),
		}, nil
	}

	// Convert MappingContext to proto format
	protoContext := convertMappingContextToProto(mappingContext)

	return &api.CreateInitiatorGroupResponse{
		Success:        true,
		Message:        fmt.Sprintf("Successfully created/updated initiator group %s", req.InitiatorGroupName),
		MappingContext: protoContext,
	}, nil
}

// MapVolumeToGroup maps a volume to an initiator group
func (s *storageArrayGRPC) MapVolumeToGroup(ctx context.Context, req *api.MapVolumeRequest) (*api.MapVolumeResponse, error) {
	if req.AccessInfo == nil {
		return &api.MapVolumeResponse{
			Success: false,
			Message: "access_info is required",
		}, nil
	}

	logrus.Infof("Mapping volume %s to group %s", req.Volume.Name, req.InitiatorGroupName)

	provider, err := storagesdk.NewStorageProvider(req.AccessInfo.VendorType)
	if err != nil {
		return &api.MapVolumeResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get storage provider: %v", err),
		}, nil
	}

	accessInfo := storagesdk.StorageAccessInfo{
		Hostname:            req.AccessInfo.Hostname,
		Username:            req.AccessInfo.Username,
		Password:            req.AccessInfo.Password,
		SkipSSLVerification: req.AccessInfo.SkipSslVerification,
		VendorType:          req.AccessInfo.VendorType,
	}

	if err := provider.Connect(ctx, accessInfo); err != nil {
		return &api.MapVolumeResponse{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}
	defer provider.Disconnect()

	// Convert proto volume to SDK volume
	volume := storagesdk.Volume{
		Name:         req.Volume.Name,
		Size:         req.Volume.Size,
		Id:           req.Volume.Id,
		SerialNumber: req.Volume.SerialNumber,
		NAA:          req.Volume.Naa,
	}

	// Convert proto mapping context to SDK mapping context
	mappingContext := convertProtoToMappingContext(req.MappingContext)

	mappedVolume, err := provider.MapVolumeToGroup(req.InitiatorGroupName, volume, mappingContext)
	if err != nil {
		return &api.MapVolumeResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to map volume: %v", err),
		}, nil
	}

	// Convert back to proto
	protoVolume := &api.VolumeInfo{
		Name:         mappedVolume.Name,
		Size:         mappedVolume.Size,
		Id:           mappedVolume.Id,
		SerialNumber: mappedVolume.SerialNumber,
		Naa:          mappedVolume.NAA,
	}

	return &api.MapVolumeResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully mapped volume %s", req.Volume.Name),
		Volume:  protoVolume,
	}, nil
}

// UnmapVolumeFromGroup unmaps a volume from an initiator group
func (s *storageArrayGRPC) UnmapVolumeFromGroup(ctx context.Context, req *api.UnmapVolumeRequest) (*api.UnmapVolumeResponse, error) {
	if req.AccessInfo == nil {
		return &api.UnmapVolumeResponse{
			Success: false,
			Message: "access_info is required",
		}, nil
	}

	logrus.Infof("Unmapping volume %s from group %s", req.Volume.Name, req.InitiatorGroupName)

	provider, err := storagesdk.NewStorageProvider(req.AccessInfo.VendorType)
	if err != nil {
		return &api.UnmapVolumeResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get storage provider: %v", err),
		}, nil
	}

	accessInfo := storagesdk.StorageAccessInfo{
		Hostname:            req.AccessInfo.Hostname,
		Username:            req.AccessInfo.Username,
		Password:            req.AccessInfo.Password,
		SkipSSLVerification: req.AccessInfo.SkipSslVerification,
		VendorType:          req.AccessInfo.VendorType,
	}

	if err := provider.Connect(ctx, accessInfo); err != nil {
		return &api.UnmapVolumeResponse{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}
	defer provider.Disconnect()

	volume := storagesdk.Volume{
		Name:         req.Volume.Name,
		Size:         req.Volume.Size,
		Id:           req.Volume.Id,
		SerialNumber: req.Volume.SerialNumber,
		NAA:          req.Volume.Naa,
	}

	mappingContext := convertProtoToMappingContext(req.MappingContext)

	if err := provider.UnmapVolumeFromGroup(req.InitiatorGroupName, volume, mappingContext); err != nil {
		return &api.UnmapVolumeResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to unmap volume: %v", err),
		}, nil
	}

	return &api.UnmapVolumeResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully unmapped volume %s", req.Volume.Name),
	}, nil
}

// GetMappedGroups retrieves the groups a volume is mapped to
func (s *storageArrayGRPC) GetMappedGroups(ctx context.Context, req *api.GetMappedGroupsRequest) (*api.GetMappedGroupsResponse, error) {
	if req.AccessInfo == nil {
		return nil, fmt.Errorf("access_info is required")
	}

	provider, err := storagesdk.NewStorageProvider(req.AccessInfo.VendorType)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage provider: %w", err)
	}

	accessInfo := storagesdk.StorageAccessInfo{
		Hostname:            req.AccessInfo.Hostname,
		Username:            req.AccessInfo.Username,
		Password:            req.AccessInfo.Password,
		SkipSSLVerification: req.AccessInfo.SkipSslVerification,
		VendorType:          req.AccessInfo.VendorType,
	}

	if err := provider.Connect(ctx, accessInfo); err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer provider.Disconnect()

	volume := storagesdk.Volume{
		Name:         req.Volume.Name,
		Size:         req.Volume.Size,
		Id:           req.Volume.Id,
		SerialNumber: req.Volume.SerialNumber,
		NAA:          req.Volume.Naa,
	}

	mappingContext := convertProtoToMappingContext(req.MappingContext)

	groups, err := provider.GetMappedGroups(volume, mappingContext)
	if err != nil {
		return nil, fmt.Errorf("failed to get mapped groups: %w", err)
	}

	return &api.GetMappedGroupsResponse{
		Groups: groups,
	}, nil
}

// ResolveCinderVolume resolves a Cinder volume name to a storage volume
func (s *storageArrayGRPC) ResolveCinderVolume(ctx context.Context, req *api.ResolveCinderVolumeRequest) (*api.ResolveCinderVolumeResponse, error) {
	if req.AccessInfo == nil {
		return &api.ResolveCinderVolumeResponse{
			Success: false,
			Message: "access_info is required",
		}, nil
	}

	logrus.Infof("Resolving Cinder volume %s", req.VolumeName)

	provider, err := storagesdk.NewStorageProvider(req.AccessInfo.VendorType)
	if err != nil {
		return &api.ResolveCinderVolumeResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get storage provider: %v", err),
		}, nil
	}

	accessInfo := storagesdk.StorageAccessInfo{
		Hostname:            req.AccessInfo.Hostname,
		Username:            req.AccessInfo.Username,
		Password:            req.AccessInfo.Password,
		SkipSSLVerification: req.AccessInfo.SkipSslVerification,
		VendorType:          req.AccessInfo.VendorType,
	}

	if err := provider.Connect(ctx, accessInfo); err != nil {
		return &api.ResolveCinderVolumeResponse{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}
	defer provider.Disconnect()

	volume, err := provider.ResolveCinderVolumeToLUN(req.VolumeName)
	if err != nil {
		return &api.ResolveCinderVolumeResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to resolve volume: %v", err),
		}, nil
	}

	protoVolume := &api.VolumeInfo{
		Name:         volume.Name,
		Size:         volume.Size,
		Id:           volume.Id,
		SerialNumber: volume.SerialNumber,
		Naa:          volume.NAA,
	}

	return &api.ResolveCinderVolumeResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully resolved volume %s", req.VolumeName),
		Volume:  protoVolume,
	}, nil
}

// Helper functions to convert between proto and SDK types

func convertMappingContextToProto(ctx storagesdk.MappingContext) []*api.MappingContextEntry {
	entries := make([]*api.MappingContextEntry, 0)
	for key, value := range ctx {
		entry := &api.MappingContextEntry{
			Key: key,
		}
		// Convert value to string slice
		switch v := value.(type) {
		case []string:
			entry.Values = v
		case string:
			entry.Values = []string{v}
		case []interface{}:
			strValues := make([]string, len(v))
			for i, val := range v {
				strValues[i] = fmt.Sprintf("%v", val)
			}
			entry.Values = strValues
		default:
			entry.Values = []string{fmt.Sprintf("%v", v)}
		}
		entries = append(entries, entry)
	}
	return entries
}

func convertProtoToMappingContext(entries []*api.MappingContextEntry) storagesdk.MappingContext {
	ctx := make(storagesdk.MappingContext)
	for _, entry := range entries {
		ctx[entry.Key] = entry.Values
	}
	return ctx
}
