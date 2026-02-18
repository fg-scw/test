package utils

import (
	"context"

	"github.com/pkg/errors"
	"github.com/platform9/vjailbreak/v2v-helper/pkg/constants"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type MigrationParams struct {
	// vCenter params
	SourceVMName string

	// openstack params
	OpenstackNetworkNames string
	OpenstackNetworkPorts string
	OpenstackVolumeTypes  string
	OpenstackVirtioWin    string
	OpenstackOSType       string
	OpenstackConvert      bool

	// Migration params
	DataCopyStart           string
	VMcutoverStart          string
	VMcutoverEnd            string
	MigrationType           string
	PerformHealthChecks     bool
	HealthCheckPort         string
	Debug                   bool
	TARGET_FLAVOR_ID        string
	TargetAvailabilityZone  string
	AssignedIP              string
	VMwareMachineName       string
	DisconnectSourceNetwork bool
	SecurityGroups          string
	ServerGroup             string
	RDMDisks                string
	FallbackToDHCP          bool
	PeriodicSyncInterval    string
	PeriodicSyncEnabled     bool
	NetworkPersistance      bool
	AcknowledgeNetworkConflictRisk bool

	StorageCopyMethod string
	VendorType        string
	ArrayCredsMapping string
}

// GetMigrationParams is function that returns the migration parameters
func GetMigrationParams(ctx context.Context, client client.Client) (*MigrationParams, error) {
	// Get the values from the secret
	configMapName, err := GetMigrationConfigMapName()
	if err != nil {
		return nil, err
	}
	configMap := &v1.ConfigMap{}
	err = client.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: constants.NamespaceMigrationSystem,
	}, configMap)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get configmap")
	}
	return &MigrationParams{
		SourceVMName:            string(configMap.Data["SOURCE_VM_NAME"]),
		OpenstackNetworkNames:   string(configMap.Data["NEUTRON_NETWORK_NAMES"]),
		OpenstackNetworkPorts:   string(configMap.Data["NEUTRON_PORT_IDS"]),
		OpenstackVolumeTypes:    string(configMap.Data["CINDER_VOLUME_TYPES"]),
		OpenstackVirtioWin:      string(configMap.Data["VIRTIO_WIN_DRIVER"]),
		OpenstackOSType:         string(configMap.Data["OS_FAMILY"]),
		OpenstackConvert:        string(configMap.Data["CONVERT"]) == constants.TrueString,
		DataCopyStart:           string(configMap.Data["DATACOPYSTART"]),
		VMcutoverStart:          string(configMap.Data["CUTOVERSTART"]),
		VMcutoverEnd:            string(configMap.Data["CUTOVEREND"]),
		MigrationType:           string(configMap.Data["TYPE"]),
		PerformHealthChecks:     string(configMap.Data["PERFORM_HEALTH_CHECKS"]) == constants.TrueString,
		HealthCheckPort:         string(configMap.Data["HEALTH_CHECK_PORT"]),
		Debug:                   string(configMap.Data["DEBUG"]) == constants.TrueString,
		TARGET_FLAVOR_ID:        string(configMap.Data["TARGET_FLAVOR_ID"]),
		TargetAvailabilityZone:  string(configMap.Data["TARGET_AVAILABILITY_ZONE"]),
		AssignedIP:              string(configMap.Data["ASSIGNED_IP"]),
		VMwareMachineName:       string(configMap.Data["VMWARE_MACHINE_OBJECT_NAME"]),
		DisconnectSourceNetwork: string(configMap.Data["DISCONNECT_SOURCE_NETWORK"]) == constants.TrueString,
		SecurityGroups:          string(configMap.Data["SECURITY_GROUPS"]),
		ServerGroup:             string(configMap.Data["SERVER_GROUP"]),
		RDMDisks:                string(configMap.Data["RDM_DISK_NAMES"]),
		FallbackToDHCP:          string(configMap.Data["FALLBACK_TO_DHCP"]) == constants.TrueString,
		PeriodicSyncInterval:    string(configMap.Data["PERIODIC_SYNC_INTERVAL"]),
		PeriodicSyncEnabled:     string(configMap.Data["PERIODIC_SYNC_ENABLED"]) == constants.TrueString,
		NetworkPersistance:      string(configMap.Data["NETWORK_PERSISTENCE"]) == constants.TrueString,
		StorageCopyMethod:       string(configMap.Data["STORAGE_COPY_METHOD"]),
		VendorType:              string(configMap.Data["VENDOR_TYPE"]),
		ArrayCredsMapping:       string(configMap.Data["ARRAY_CREDS_MAPPING"]),
	AcknowledgeNetworkConflictRisk: string(configMap.Data["ACKNOWLEDGE_NETWORK_CONFLICT_RISK"]) == constants.TrueString,
	}, nil
}
