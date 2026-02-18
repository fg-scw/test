package netapp

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/platform9/vjailbreak/pkg/vpwned/sdk/storage"
	"k8s.io/klog/v2"
)

// NetApp ONTAP NAA prefix (OUI for NetApp)
const NetAppProviderID = "600a0980"

func init() {
	storage.RegisterStorageProvider("netapp", &NetAppStorageProvider{})
}

// NetAppStorageProvider implements StorageProvider for NetApp ONTAP
type NetAppStorageProvider struct {
	storage.BaseStorageProvider
}

// ONTAP API response structures
type OntapClusterInfo struct {
	Name    string `json:"name"`
	UUID    string `json:"uuid"`
	Version struct {
		Full string `json:"full"`
	} `json:"version"`
}

type OntapLUN struct {
	UUID         string `json:"uuid"`
	Name         string `json:"name"`
	SerialNumber string `json:"serial_number"`
	Space        struct {
		Size int64 `json:"size"`
	} `json:"space"`
	SVM struct {
		Name string `json:"name"`
		UUID string `json:"uuid"`
	} `json:"svm"`
	Location struct {
		Volume struct {
			Name string `json:"name"`
			UUID string `json:"uuid"`
		} `json:"volume"`
	} `json:"location"`
	CreateTime string `json:"create_time"`
}

type OntapLUNResponse struct {
	Records    []OntapLUN `json:"records"`
	NumRecords int        `json:"num_records"`
}

type OntapIgroup struct {
	UUID       string `json:"uuid"`
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	Initiators []struct {
		Name string `json:"name"`
	} `json:"initiators"`
}

type OntapIgroupResponse struct {
	Records    []OntapIgroup `json:"records"`
	NumRecords int           `json:"num_records"`
}


// Connect establishes connection to NetApp ONTAP array
func (n *NetAppStorageProvider) Connect(ctx context.Context, accessInfo storage.StorageAccessInfo) error {
	n.AccessInfo = accessInfo
	n.Config = storage.VendorConfig{
		NAAPrefix: NetAppProviderID,
		Name:      "NetApp",
	}
	n.BaseURL = fmt.Sprintf("https://%s/api", accessInfo.Hostname)
	n.Username = accessInfo.Username
	n.Password = accessInfo.Password

	n.InitHTTPClient(accessInfo.SkipSSLVerification)
	n.SetConnected(true)

	// Validate connection by getting cluster info
	cluster, err := n.getClusterInfo(ctx)
	if err != nil {
		n.SetConnected(false)
		return fmt.Errorf("failed to connect to NetApp ONTAP cluster: %w", err)
	}

	klog.Infof("Connected to NetApp ONTAP Cluster: %s, Version: %s", cluster.Name, cluster.Version.Full)
	return nil
}

// Disconnect closes the connection
func (n *NetAppStorageProvider) Disconnect() error {
	n.SetConnected(false)
	return nil
}

// ValidateCredentials validates the credentials
func (n *NetAppStorageProvider) ValidateCredentials(ctx context.Context) error {
	if !n.GetConnected() {
		err := n.Connect(ctx, n.AccessInfo)
		if err != nil {
			return err
		}
	}

	// Try to get cluster info as validation
	_, err := n.getClusterInfo(ctx)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}

// CreateVolume creates a new LUN on the NetApp array
func (n *NetAppStorageProvider) CreateVolume(volumeName string, size int64) (storage.Volume, error) {
	ctx := context.Background()

	// Get the volume path and SVM from existing LUNs
	volumePath, svmName, err := n.getDefaultVolumePathAndSVM(ctx)
	if err != nil {
		return storage.Volume{}, fmt.Errorf("failed to determine volume path: %w", err)
	}

	// Build full LUN path: /vol/<volume_name>/<lun_name>
	lunPath := fmt.Sprintf("%s/%s", volumePath, volumeName)
	klog.Infof("Creating NetApp LUN at path: %s with size: %d on SVM: %s", lunPath, size, svmName)

	// Create LUN via ONTAP REST API
	reqBody := map[string]interface{}{
		"name": lunPath,
		"svm": map[string]interface{}{
			"name": svmName,
		},
		"space": map[string]interface{}{
			"size": size,
		},
		"os_type": "vmware",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return storage.Volume{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use return_records=true to get the created LUN in the response
	type CreateResponse struct {
		Records []OntapLUN `json:"records"`
	}
	var response CreateResponse
	err = n.DoRequestJSON(ctx, "POST", "/storage/luns?return_records=true", bytes.NewReader(jsonBody), &response)
	if err != nil {
		return storage.Volume{}, fmt.Errorf("failed to create LUN %s: %w", volumeName, err)
	}

	if len(response.Records) == 0 {
		return storage.Volume{}, fmt.Errorf("LUN creation succeeded but no records returned for %s", volumeName)
	}

	lun := response.Records[0]
	klog.Infof("Created NetApp LUN: %s, Serial: %s", lun.Name, lun.SerialNumber)

	return storage.Volume{
		Name:         lun.Name,
		Size:         lun.Space.Size,
		Id:           lun.UUID,
		SerialNumber: lun.SerialNumber,
		NAA:          n.BuildNAA(lun.SerialNumber),
	}, nil
}

// DeleteVolume deletes a LUN from the NetApp array
func (n *NetAppStorageProvider) DeleteVolume(volumeName string) error {
	ctx := context.Background()

	// Find the LUN by name
	luns, err := n.listLUNs(ctx, fmt.Sprintf("name=*%s*", volumeName))
	if err != nil {
		return fmt.Errorf("failed to find LUN %s: %w", volumeName, err)
	}

	if len(luns) == 0 {
		return fmt.Errorf("LUN %s not found", volumeName)
	}

	lun := luns[0]
	klog.Infof("Deleting NetApp LUN: %s (UUID: %s)", lun.Name, lun.UUID)

	// Delete LUN via ONTAP REST API
	err = n.DoRequestJSON(ctx, "DELETE", fmt.Sprintf("/storage/luns/%s", lun.UUID), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete LUN %s: %w", volumeName, err)
	}

	klog.Infof("Deleted NetApp LUN: %s", volumeName)
	return nil
}

// GetVolumeInfo retrieves information about a LUN from the NetApp array
func (n *NetAppStorageProvider) GetVolumeInfo(volumeName string) (storage.VolumeInfo, error) {
	ctx := context.Background()
	luns, err := n.listLUNs(ctx, fmt.Sprintf("name=%s", volumeName))
	if err != nil {
		return storage.VolumeInfo{}, fmt.Errorf("failed to get LUN %s: %w", volumeName, err)
	}

	if len(luns) == 0 {
		return storage.VolumeInfo{}, fmt.Errorf("LUN %s not found", volumeName)
	}

	lun := luns[0]
	return storage.VolumeInfo{
		Name:    lun.Name,
		Size:    lun.Space.Size,
		Created: lun.CreateTime,
		NAA:     n.BuildNAA(lun.SerialNumber),
	}, nil
}

// ListAllVolumes retrieves all LUNs from the NetApp array
func (n *NetAppStorageProvider) ListAllVolumes() ([]storage.VolumeInfo, error) {
	ctx := context.Background()
	luns, err := n.listLUNs(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list LUNs: %w", err)
	}

	var volumeInfos []storage.VolumeInfo
	for _, lun := range luns {
		volumeInfos = append(volumeInfos, storage.VolumeInfo{
			Name:    lun.Name,
			Size:    lun.Space.Size,
			Created: lun.CreateTime,
			NAA:     n.BuildNAA(lun.SerialNumber),
		})
	}

	return volumeInfos, nil
}

// GetAllVolumeNAAs retrieves NAA identifiers for all LUNs on the array
func (n *NetAppStorageProvider) GetAllVolumeNAAs() ([]string, error) {
	return n.BaseStorageProvider.GetAllVolumeNAAs(n.ListAllVolumes)
}

// CreateOrUpdateInitiatorGroup creates or updates an igroup with the ESX adapters
func (n *NetAppStorageProvider) CreateOrUpdateInitiatorGroup(initiatorGroupName string, hbaIdentifiers []string) (storage.MappingContext, error) {
	ctx := context.Background()

	// List existing igroups and find matches
	igroups, err := n.listIgroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list igroups: %w", err)
	}

	matchedIgroups := []string{}

	for _, ig := range igroups {
		initiatorNames := make([]string, len(ig.Initiators))
		for i, init := range ig.Initiators {
			initiatorNames[i] = init.Name
		}
		klog.Infof("Checking igroup %s, initiators: %v", ig.Name, initiatorNames)

		for _, init := range ig.Initiators {
			if storage.ContainsIgnoreCase(hbaIdentifiers, init.Name) {
				klog.Infof("Adding igroup %s (matched initiator: %s)", ig.Name, init.Name)
				matchedIgroups = append(matchedIgroups, ig.Name)
				break
			}
		}
	}

	if len(matchedIgroups) == 0 {
		return nil, fmt.Errorf("no igroups found matching any of the provided IQNs/WWNs: %v", hbaIdentifiers)
	}

	return storage.MappingContext{"igroups": matchedIgroups}, nil
}

// MapVolumeToGroup maps a LUN to igroups
func (n *NetAppStorageProvider) MapVolumeToGroup(initiatorGroupName string, targetVolume storage.Volume, mappingCtx storage.MappingContext) (storage.Volume, error) {
	ctx := context.Background()

	igroupsVal, ok := mappingCtx["igroups"]
	if !ok {
		return storage.Volume{}, errors.New("igroups not found in mapping context")
	}

	igroups, ok := igroupsVal.([]string)
	if !ok || len(igroups) == 0 {
		return storage.Volume{}, errors.New("invalid or empty igroups list in mapping context")
	}

	// Get LUN details to find UUID
	lun, err := n.getLUNByName(ctx, targetVolume.Name)
	if err != nil {
		return storage.Volume{}, fmt.Errorf("failed to get LUN %s: %w", targetVolume.Name, err)
	}

	for _, igroupName := range igroups {
		klog.Infof("Mapping LUN %s to igroup %s", targetVolume.Name, igroupName)

		// Get igroup UUID - ONTAP API requires UUID for lun-maps
		igroup, err := n.getIgroupByName(ctx, igroupName)
		if err != nil {
			return storage.Volume{}, fmt.Errorf("failed to get igroup %s: %w", igroupName, err)
		}

		reqBody := map[string]interface{}{
			"svm": map[string]interface{}{
				"name": lun.SVM.Name,
			},
			"lun": map[string]interface{}{
				"uuid": lun.UUID,
			},
			"igroup": map[string]interface{}{
				"uuid": igroup.UUID,
			},
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return storage.Volume{}, fmt.Errorf("failed to marshal request: %w", err)
		}

		err = n.DoRequestJSON(ctx, "POST", "/protocols/san/lun-maps", bytes.NewReader(jsonBody), nil)
		if err != nil {
			// Check if already mapped
			if strings.Contains(err.Error(), "already mapped") {
				klog.Infof("LUN %s already mapped to igroup %s", targetVolume.Name, igroupName)
				continue
			}
			return storage.Volume{}, fmt.Errorf("failed to map LUN %s to igroup %s: %w", targetVolume.Name, igroupName, err)
		}
		klog.Infof("Successfully mapped LUN %s to igroup %s", targetVolume.Name, igroupName)
	}

	return targetVolume, nil
}

// UnmapVolumeFromGroup unmaps a LUN from igroups
func (n *NetAppStorageProvider) UnmapVolumeFromGroup(initiatorGroupName string, targetVolume storage.Volume, mappingCtx storage.MappingContext) error {
	ctx := context.Background()

	igroupsVal, ok := mappingCtx["igroups"]
	if !ok {
		return nil // No igroups to unmap
	}

	igroups, ok := igroupsVal.([]string)
	if !ok || len(igroups) == 0 {
		return nil
	}

	// Get LUN details to find UUID
	lun, err := n.getLUNByName(ctx, targetVolume.Name)
	if err != nil {
		klog.Warningf("Failed to get LUN %s for unmapping: %v", targetVolume.Name, err)
		return nil // LUN might already be deleted
	}

	for _, igroupName := range igroups {
		klog.Infof("Unmapping LUN %s from igroup %s", targetVolume.Name, igroupName)

		// Get igroup UUID
		igroup, err := n.getIgroupByName(ctx, igroupName)
		if err != nil {
			klog.Warningf("Failed to get igroup %s: %v", igroupName, err)
			continue
		}

		endpoint := fmt.Sprintf("/protocols/san/lun-maps/%s/%s", lun.UUID, igroup.UUID)
		err = n.DoRequestJSON(ctx, "DELETE", endpoint, nil, nil)
		if err != nil {
			klog.Warningf("Failed to unmap LUN %s from igroup %s: %v", targetVolume.Name, igroupName, err)
			continue
		}
		klog.Infof("Successfully unmapped LUN %s from igroup %s", targetVolume.Name, igroupName)
	}

	return nil
}

// GetMappedGroups returns the igroups the LUN is mapped to
func (n *NetAppStorageProvider) GetMappedGroups(targetVolume storage.Volume, mappingCtx storage.MappingContext) ([]string, error) {
	ctx := context.Background()

	lun, err := n.getLUNByName(ctx, targetVolume.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get LUN %s: %w", targetVolume.Name, err)
	}

	var response struct {
		Records []struct {
			Igroup struct {
				Name string `json:"name"`
			} `json:"igroup"`
		} `json:"records"`
	}

	endpoint := fmt.Sprintf("/protocols/san/lun-maps?lun.uuid=%s", lun.UUID)
	err = n.DoRequestJSON(ctx, "GET", endpoint, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get LUN mappings: %w", err)
	}

	var groups []string
	for _, record := range response.Records {
		groups = append(groups, record.Igroup.Name)
	}

	return groups, nil
}

// ResolveCinderVolumeToLUN resolves a Cinder volume name to a storage LUN
func (n *NetAppStorageProvider) ResolveCinderVolumeToLUN(volumeID string) (storage.Volume, error) {
	ctx := context.Background()

	// NetApp Cinder driver naming convention - search for LUN containing the volume ID
	// Common patterns: "volume-<uuid>", "/vol/<volume>/lun-<uuid>"
	luns, err := n.listLUNs(ctx, fmt.Sprintf("name=*%s*", volumeID))
	if err != nil {
		return storage.Volume{}, fmt.Errorf("failed to search for LUN with volume ID %s: %w", volumeID, err)
	}

	if len(luns) == 0 {
		return storage.Volume{}, fmt.Errorf("no LUN found matching Cinder volume ID %s", volumeID)
	}

	lun := luns[0]
	klog.Infof("Resolved Cinder volume %s to LUN: %+v", volumeID, lun)

	return storage.Volume{
		Name:         lun.Name,
		Size:         lun.Space.Size,
		SerialNumber: lun.SerialNumber,
		NAA:          n.BuildNAA(lun.SerialNumber),
	}, nil
}

// GetVolumeFromNAA retrieves a NetApp LUN by its NAA identifier
func (n *NetAppStorageProvider) GetVolumeFromNAA(naaID string) (storage.Volume, error) {
	serial, err := n.ExtractSerialFromNAA(naaID)
	if err != nil {
		return storage.Volume{}, err
	}

	ctx := context.Background()
	luns, err := n.listLUNs(ctx, "")
	if err != nil {
		return storage.Volume{}, fmt.Errorf("failed to list LUNs: %w", err)
	}

	for _, lun := range luns {
		if strings.ToUpper(lun.SerialNumber) == strings.ToUpper(serial) {
			klog.Infof("Found NetApp LUN %s with serial %s matching NAA %s", lun.Name, lun.SerialNumber, naaID)
			return storage.Volume{
				Name:         lun.Name,
				Size:         lun.Space.Size,
				SerialNumber: lun.SerialNumber,
				NAA:          naaID,
			}, nil
		}
	}

	return storage.Volume{}, fmt.Errorf("no NetApp LUN found with NAA %s (serial: %s)", naaID, serial)
}

// WhoAmI returns the provider name
func (n *NetAppStorageProvider) WhoAmI() string {
	return "netapp"
}

// BuildNAA constructs a NAA identifier from a NetApp serial number
// NetApp serial numbers are ASCII and need to be hex-encoded for NAA
func (n *NetAppStorageProvider) BuildNAA(serialNumber string) string {
	hexSerial := hex.EncodeToString([]byte(serialNumber))
	return fmt.Sprintf("naa.%s%s", n.Config.NAAPrefix, strings.ToLower(hexSerial))
}

// ExtractSerialFromNAA extracts the serial number from a NAA identifier
// Decodes the hex-encoded serial back to ASCII
func (n *NetAppStorageProvider) ExtractSerialFromNAA(naaID string) (string, error) {
	prefix := "naa." + n.Config.NAAPrefix
	if !strings.HasPrefix(strings.ToLower(naaID), strings.ToLower(prefix)) {
		return "", fmt.Errorf("NAA ID %s is not from NetApp (expected prefix: %s)", naaID, prefix)
	}

	hexSerial := strings.TrimPrefix(strings.ToLower(naaID), strings.ToLower(prefix))
	serialBytes, err := hex.DecodeString(hexSerial)
	if err != nil {
		return "", fmt.Errorf("failed to decode NAA serial: %w", err)
	}

	return string(serialBytes), nil
}

// Helper methods

func (n *NetAppStorageProvider) getClusterInfo(ctx context.Context) (*OntapClusterInfo, error) {
	var cluster OntapClusterInfo
	err := n.DoRequestJSON(ctx, "GET", "/cluster", nil, &cluster)
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (n *NetAppStorageProvider) listLUNs(ctx context.Context, filter string) ([]OntapLUN, error) {
	endpoint := "/storage/luns?fields=serial_number,space,location,create_time,svm"
	if filter != "" {
		endpoint = fmt.Sprintf("%s&%s", endpoint, filter)
	}

	var response OntapLUNResponse
	err := n.DoRequestJSON(ctx, "GET", endpoint, nil, &response)
	if err != nil {
		return nil, err
	}

	return response.Records, nil
}

func (n *NetAppStorageProvider) listIgroups(ctx context.Context) ([]OntapIgroup, error) {
	var response OntapIgroupResponse
	err := n.DoRequestJSON(ctx, "GET", "/protocols/san/igroups?fields=uuid,name,initiators", nil, &response)
	if err != nil {
		return nil, err
	}

	return response.Records, nil
}

// getLUNByName retrieves a LUN by its name
func (n *NetAppStorageProvider) getLUNByName(ctx context.Context, name string) (*OntapLUN, error) {
	var luns []OntapLUN
	var err error

	// If name is a full path (starts with /), use exact match
	// Otherwise, use wildcard search since NetApp requires path format
	if strings.HasPrefix(name, "/") {
		luns, err = n.listLUNs(ctx, fmt.Sprintf("name=%s", name))
	} else {
		// Use wildcard search for non-path names (e.g., "volume-xxx-cinder")
		luns, err = n.listLUNs(ctx, fmt.Sprintf("name=*%s*", name))
	}

	if err != nil {
		return nil, err
	}
	if len(luns) == 0 {
		return nil, fmt.Errorf("LUN %s not found", name)
	}
	return &luns[0], nil
}

// getIgroupByName retrieves an igroup by its name
func (n *NetAppStorageProvider) getIgroupByName(ctx context.Context, name string) (*OntapIgroup, error) {
	igroups, err := n.listIgroups(ctx)
	if err != nil {
		return nil, err
	}
	for _, ig := range igroups {
		if ig.Name == name {
			return &ig, nil
		}
	}
	return nil, fmt.Errorf("igroup %s not found", name)
}

// getDefaultVolumePathAndSVM discovers the volume path and SVM from existing LUNs
// LUN paths are like /vol/cinder_vol/lun_name, we extract /vol/cinder_vol and SVM name
func (n *NetAppStorageProvider) getDefaultVolumePathAndSVM(ctx context.Context) (string, string, error) {
	luns, err := n.listLUNs(ctx, "")
	if err != nil {
		return "", "", fmt.Errorf("failed to list LUNs: %w", err)
	}

	if len(luns) == 0 {
		return "", "", fmt.Errorf("no existing LUNs found to determine volume path and SVM")
	}

	lun := luns[0]

	// LUN name format: /vol/<volume_name>/<lun_name>
	// Extract /vol/<volume_name>
	parts := strings.Split(lun.Name, "/")
	if len(parts) < 4 || parts[1] != "vol" {
		return "", "", fmt.Errorf("unexpected LUN path format: %s", lun.Name)
	}

	volumePath := fmt.Sprintf("/vol/%s", parts[2])
	svmName := lun.SVM.Name

	if svmName == "" {
		return "", "", fmt.Errorf("SVM name not found for LUN: %s", lun.Name)
	}

	klog.Infof("Discovered NetApp volume path: %s, SVM: %s from LUN: %s", volumePath, svmName, lun.Name)

	return volumePath, svmName, nil
}
