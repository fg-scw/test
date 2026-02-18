// Copyright © 2024 The vjailbreak authors

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/platform9/vjailbreak/v2v-helper/migrate"
	"github.com/platform9/vjailbreak/v2v-helper/nbd"
	"github.com/platform9/vjailbreak/v2v-helper/openstack"
	"github.com/platform9/vjailbreak/v2v-helper/pkg/constants"
	"github.com/platform9/vjailbreak/v2v-helper/pkg/utils"
	"github.com/platform9/vjailbreak/v2v-helper/reporter"
	"github.com/platform9/vjailbreak/v2v-helper/vcenter"
	"github.com/platform9/vjailbreak/v2v-helper/vm"
	"github.com/vmware/govmomi/vim25/types"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is canceled when we exit

	// Initialize error reporter early
	eventReporter, err := reporter.NewReporter()
	if err != nil {
		utils.PrintLog(fmt.Sprintf("Failed to create reporter: %v", err))
		return
	}

	eventReporterChan := make(chan string)
	podLabelWatcherChan := make(chan string)
	ackChan := make(chan struct{})

	defer close(eventReporterChan)
	defer close(podLabelWatcherChan)

	// Start reporter goroutines
	eventReporter.UpdatePodEvents(ctx, eventReporterChan, ackChan)
	eventReporter.WatchPodLabels(ctx, podLabelWatcherChan)

	// Helper function to report and handle errors
	handleError := func(msg string) {
		if reporter.IsRunningInPod() {
			eventReporterChan <- msg
			// Wait for the reporter to process the message
			<-ackChan
		}
		utils.PrintLog(msg)
	}

	client, err := utils.GetInclusterClient()
	if err != nil {
		handleError(fmt.Sprintf("Failed to get in-cluster client: %v", err))
	}

	migrationparams, err := utils.GetMigrationParams(ctx, client)
	if err != nil {
		handleError(fmt.Sprintf("Failed to get migration parameters: %v", err))
	}
	utils.WriteToLogFile(fmt.Sprintf("-----\t Migration started at %s for VM %s -----", time.Now().Format(time.RFC3339), migrationparams.SourceVMName))

	logMigrationParams(migrationparams)

	var (
		vCenterURL        = strings.TrimSpace(os.Getenv("VCENTER_HOST"))
		vCenterUserName   = strings.TrimSpace(os.Getenv("VCENTER_USERNAME"))
		vCenterPassword   = strings.TrimSpace(os.Getenv("VCENTER_PASSWORD"))
		vCenterInsecure   = strings.EqualFold(strings.TrimSpace(os.Getenv("VCENTER_INSECURE")), constants.TrueString)
		openstackInsecure = strings.EqualFold(strings.TrimSpace(os.Getenv("OS_INSECURE")), constants.TrueString)
		arrayHost         = strings.TrimSpace(os.Getenv("ARRAY_HOSTNAME"))
		arrayUser         = strings.TrimSpace(os.Getenv("ARRAY_USERNAME"))
		arrayPassword     = strings.TrimSpace(os.Getenv("ARRAY_PASSWORD"))
		arrayInsecure     = strings.EqualFold(strings.TrimSpace(os.Getenv("ARRAY_INSECURE")), constants.TrueString)
	)

	openstackProjectName := strings.TrimSpace(os.Getenv("OS_PROJECT_NAME"))
	if openstackProjectName == "" {
		openstackProjectName = strings.TrimSpace(os.Getenv("OS_TENANT_NAME"))
	}

	starttime, _ := time.Parse(time.RFC3339, migrationparams.DataCopyStart)
	cutstart, _ := time.Parse(time.RFC3339, migrationparams.VMcutoverStart)
	cutend, _ := time.Parse(time.RFC3339, migrationparams.VMcutoverEnd)

	// Validate vCenter connection
	vcclient, err := vcenter.VCenterClientBuilder(ctx, vCenterUserName, vCenterPassword, vCenterURL, vCenterInsecure)
	if err != nil {
		handleError(fmt.Sprintf("Failed to validate vCenter connection: %v", err))
		return
	}
	utils.PrintLog(fmt.Sprintf("Connected to vCenter: %s\n", vCenterURL))
	defer vcclient.VCClient.CloseIdleConnections()
	// Validate OpenStack connection
	openstackclients, err := openstack.NewOpenStackClients(ctx, openstackInsecure)
	if err != nil {
		handleError(fmt.Sprintf("Failed to validate OpenStack connection: %v", err))
		return
	}
	openstackclients.K8sClient = client
	utils.PrintLog("Connected to OpenStack")

	// Get thumbprint
	thumbprint, err := vcenter.GetThumbprint(vCenterURL)
	if err != nil {
		handleError(fmt.Sprintf("Failed to get thumbprint: %s", err))
		return
	}
	utils.PrintLog(fmt.Sprintf("VCenter Thumbprint: %s\n", thumbprint))

	// Retrieve the source VM
	vmops, err := vm.VMOpsBuilder(ctx, *vcclient, migrationparams.SourceVMName, client)
	if err != nil {
		handleError(fmt.Sprintf("Failed to get source VM: %v", err))
		return
	}
	migrationobj := migrate.Migrate{
		URL:                     vCenterURL,
		UserName:                vCenterUserName,
		Password:                vCenterPassword,
		Insecure:                vCenterInsecure,
		Networknames:            utils.RemoveEmptyStrings(strings.Split(migrationparams.OpenstackNetworkNames, ",")),
		Networkports:            utils.RemoveEmptyStrings(strings.Split(migrationparams.OpenstackNetworkPorts, ",")),
		Volumetypes:             utils.RemoveEmptyStrings(strings.Split(migrationparams.OpenstackVolumeTypes, ",")),
		Virtiowin:               migrationparams.OpenstackVirtioWin,
		Ostype:                  migrationparams.OpenstackOSType,
		Thumbprint:              thumbprint,
		Convert:                 migrationparams.OpenstackConvert,
		DisconnectSourceNetwork: migrationparams.DisconnectSourceNetwork,
		Openstackclients:        openstackclients,
		Vcclient:                vcclient,
		VMops:                   vmops,
		Nbdops:                  []nbd.NBDOperations{},
		EventReporter:           eventReporterChan,
		PodLabelWatcher:         podLabelWatcherChan,
		InPod:                   reporter.IsRunningInPod(),
		MigrationTimes: migrate.MigrationTimes{
			DataCopyStart:  starttime,
			VMCutoverStart: cutstart,
			VMCutoverEnd:   cutend,
		},
		MigrationType:          migrationparams.MigrationType,
		PerformHealthChecks:    migrationparams.PerformHealthChecks,
		HealthCheckPort:        migrationparams.HealthCheckPort,
		K8sClient:              client,
		TargetFlavorId:         migrationparams.TARGET_FLAVOR_ID,
		TargetAvailabilityZone: migrationparams.TargetAvailabilityZone,
		AssignedIP:             migrationparams.AssignedIP,
		SecurityGroups:         utils.RemoveEmptyStrings(strings.Split(migrationparams.SecurityGroups, ",")),
		ServerGroup:            migrationparams.ServerGroup,
		RDMDisks:               utils.RemoveEmptyStrings(strings.Split(migrationparams.RDMDisks, ",")),
		UseFlavorless:          os.Getenv("USE_FLAVORLESS") == "true",
		TenantName:             openstackProjectName,
		Reporter:               eventReporter,
		FallbackToDHCP:         migrationparams.FallbackToDHCP,
		StorageCopyMethod:      migrationparams.StorageCopyMethod,
		ArrayHost:              arrayHost,
		ArrayUser:              arrayUser,
		ArrayPassword:          arrayPassword,
		ArrayInsecure:          arrayInsecure,
		VendorType:             migrationparams.VendorType,
		ArrayCredsMapping:      migrationparams.ArrayCredsMapping,
	}

	if migrationobj.ServerGroup != "" {
		utils.PrintLog(fmt.Sprintf("Server group configured: %s", migrationobj.ServerGroup))
	} else {
		utils.PrintLog("No server group configured for this migration")
	}

	PreMigrationPowerState, err := vmops.GetVmPowerState()
	if err != nil {
		utils.PrintLog(fmt.Sprintf("Failed to get VM power state: %v", err))
		PreMigrationPowerState = types.VirtualMachinePowerStatePoweredOn
	}
	if err := migrationobj.MigrateVM(ctx); err != nil {
		msg := fmt.Sprintf("Failed to migrate VM: %v. ", err)

		// Try to power on the VM if migration failed
		if PreMigrationPowerState == types.VirtualMachinePowerStatePoweredOff {
			msg += fmt.Sprintf("\nDetected Cold Migration. Not powering on VM")
		} else {
			powerOnErr := vmops.VMPowerOn()
			if powerOnErr != nil {
				msg += fmt.Sprintf("\nAlso Failed to power on VM after migration failure: %v", powerOnErr)
			} else {
				msg += fmt.Sprintf("\nVM %s was powered on after migration failure", migrationparams.SourceVMName)
			}
		}

		handleError(msg)
		utils.PrintLog(fmt.Sprintf("----- Migration completed with errors at %s for VM %s -----", time.Now().Format(time.RFC3339), migrationparams.SourceVMName))
		return
	}

	utils.PrintLog(fmt.Sprintf("----- Migration completed successfully at %s for VM %s -----", time.Now().Format(time.RFC3339), migrationparams.SourceVMName))
}

func logMigrationParams(migrationparams *utils.MigrationParams) {
	utils.PrintLog(fmt.Sprintf(
		`Received migration parameters:
SOURCE_VM_NAME=%s
OS_FAMILY=%s
TYPE=%s
TARGET_FLAVOR_ID=%s
TARGET_AVAILABILITY_ZONE=%s
ASSIGNED_IP=%s
DISCONNECT_SOURCE_NETWORK=%s
SECURITY_GROUPS=%s
SERVER_GROUP=%s
RDM_DISKS=%s
FALLBACK_TO_DHCP=%s
PERIODIC_SYNC_INTERVAL=%s
PERIODIC_SYNC_ENABLED=%s
NETWORK_PERSISTENCE=%s
STORAGE_COPY_METHOD=%s
VENDOR_TYPE=%s
ARRAY_CREDS_MAPPING=%s
ACKNOWLEDGE_NETWORK_CONFLICT_RISK=%s`,
		migrationparams.SourceVMName,
		migrationparams.OpenstackOSType,
		migrationparams.MigrationType,
		migrationparams.TARGET_FLAVOR_ID,
		migrationparams.TargetAvailabilityZone,
		migrationparams.AssignedIP,
		migrationparams.DisconnectSourceNetwork,
		migrationparams.SecurityGroups,
		migrationparams.ServerGroup,
		migrationparams.RDMDisks,
		migrationparams.FallbackToDHCP,
		migrationparams.PeriodicSyncInterval,
		migrationparams.PeriodicSyncEnabled,
		migrationparams.NetworkPersistance,
		migrationparams.StorageCopyMethod,
		migrationparams.VendorType,
		migrationparams.ArrayCredsMapping,
		migrationparams.AcknowledgeNetworkConflictRisk,
	))
}
