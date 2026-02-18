package maas

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	ipmi "github.com/bougou/go-ipmi"
	gomaasclient "github.com/canonical/gomaasclient/client"
	"github.com/canonical/gomaasclient/entity"
	api "github.com/platform9/vjailbreak/pkg/vpwned/api/proto/v1/service"
	"github.com/sirupsen/logrus"
)

// MaasClient represents a client for interacting with MaaS API
type MaasClient struct {
	BaseURL string
	ApiKey  string
	Client  *gomaasclient.Client
}

// NewMaasClient creates a new MaaS API client
func NewMaasClient(accessInfo MaasAccessInfo) (*MaasClient, error) {
	if !strings.Contains(accessInfo.BaseURL, "MAAS") {
		return nil, errors.New("invalid base URL")
	}
	client := &MaasClient{
		BaseURL: strings.TrimRight(accessInfo.BaseURL, "/"),
		ApiKey:  accessInfo.APIKey,
	}
	c, err := gomaasclient.GetClient(client.BaseURL, client.ApiKey, "2.0")
	if err != nil {
		logrus.Errorf("Failed to create MaaS client: %v", err)
		return nil, errors.Wrap(err, "failed to create maas client")
	}
	client.Client = c
	return client, nil
}

// ListMachines retrieves a list of machines from MaaS
func (m *MaasClient) ListMachines(ctx context.Context) ([]api.MachineInfo, error) {
	if m.Client == nil {
		return nil, errors.New("client not initialized")
	}
	machines, err := m.Client.Machines.Get(&entity.MachinesParams{})
	if err != nil {
		logrus.Errorf("Failed to list machines: %v", err)
		return nil, errors.Wrap(err, "failed to list machines")
	}
	var result []api.MachineInfo
	result = make([]api.MachineInfo, len(machines))
	for i, v := range machines {
		result[i] = api.MachineInfo{
			Id:              v.SystemID,
			Fqdn:            v.FQDN,
			Os:              v.OSystem,
			PowerState:      v.PowerState,
			Hostname:        v.Hostname,
			Architecture:    v.Architecture,
			Memory:          fmt.Sprintf("%d", v.Memory),
			CpuCount:        fmt.Sprintf("%d", v.CPUCount),
			CpuSpeed:        fmt.Sprintf("%d", v.CPUSpeed),
			BootDiskSize:    fmt.Sprintf("%d", v.BootDisk.Size),
			Status:          v.StatusName,
			StatusMessage:   v.StatusMessage,
			StatusAction:    v.StatusAction,
			Description:     v.Description,
			Domain:          v.Domain.Name,
			Zone:            v.Zone.Name,
			Pool:            v.Pool.Name,
			TagNames:        strings.Join(v.TagNames, ","),
			Netboot:         v.Netboot,
			EphemeralDeploy: v.EphemeralDeploy,
			PowerType:       v.PowerType,
			BiosBootMethod:  v.BiosBootMethod,
			HardwareUuid:    v.HardwareUUID,
			MacAddress:      v.BootInterface.MACAddress,
		}
	}
	return result, nil
}

// SetMachinePower changes the power state of a machine
func (m *MaasClient) SetMachinePower(ctx context.Context, systemID string, action api.PowerStatus) error {
	if m.Client == nil {
		return errors.New("client not initialized")
	}
	_, err := m.Client.Machine.Get(systemID)
	if err != nil {
		logrus.Errorf("Failed to get machine: %v", err)
		return errors.Wrap(err, "failed to get machine")
	}

	// Determine the power action
	var errs error
	errs = nil
	switch action {
	case api.PowerStatus_POWERED_ON:
		_, errs = m.Client.Machine.PowerOn(systemID, &entity.MachinePowerOnParams{})
		logrus.Infof("Machine %s powered on", systemID)
	case api.PowerStatus_POWERED_OFF:
		_, errs = m.Client.Machine.PowerOff(systemID, &entity.MachinePowerOffParams{})
		logrus.Infof("Machine %s powered off", systemID)
	default:
		return fmt.Errorf("unsupported power action: %v", action)
	}

	if errs != nil {
		logrus.Errorf("Failed to change power state: %v", errs)
		return errs
	}

	return nil
}

func (m *MaasClient) GetMachineFromID(ctx context.Context, systemID string) (*entity.Machine, error) {
	if m.Client == nil {
		return nil, errors.New("client not initialized")
	}
	return m.Client.Machine.Get(systemID)
}

func (m *MaasClient) GetIPMIInterface(ctx context.Context, req *api.IpmiType) (ipmi.Interface, error) {
	con_interface := ipmi.InterfaceLanplus
	if req == nil {
		return con_interface, nil
	}
	switch req.IpmiInterface.(type) {
	case *api.IpmiType_Lan:
		con_interface = ipmi.InterfaceLan
	case *api.IpmiType_Lanplus:
		con_interface = ipmi.InterfaceLanplus
	case *api.IpmiType_OpenIpmi:
		con_interface = ipmi.InterfaceOpen
	case *api.IpmiType_Tool:
		con_interface = ipmi.InterfaceTool
	}
	return con_interface, nil
}

func (m *MaasClient) ReleaseMachine(ctx context.Context, machine *entity.Machine, eraseDisk bool) error {
	if m.Client == nil {
		return errors.New("release: client not initialized")
	}
	_, err := m.Client.Machine.Release(machine.SystemID, &entity.MachineReleaseParams{
		Comment: "vJailbreak: Releasing machine for re-deployment",
		Erase:   eraseDisk,
	})
	if err != nil {
		logrus.Errorf("Failed to release machine: %v", err)
		return errors.Wrap(err, "failed to release machine")
	}
	return nil
}

func (m *MaasClient) DeployMachine(ctx context.Context, machine *entity.Machine, cloudInitScript, OSRelease string) error {
	if m.Client == nil {
		return errors.New("deploy: client not initialized")
	}
	logrus.Debugf("Deploying machine %s with cloud-init script %s and OS release %s", machine.Hostname, cloudInitScript, OSRelease)
	_, err := m.Client.Machine.Deploy(machine.SystemID, &entity.MachineDeployParams{
		Comment:      "vJailbreak: Deploying machine with cloud-init script",
		UserData:     cloudInitScript,
		DistroSeries: OSRelease,
	})
	if err != nil {
		logrus.Errorf("Failed to deploy machine: %v", err)
		return errors.Wrap(err, "failed to deploy machine")
	}
	return nil
}

// Reclaim does the following:
// Take a cloud_init script that will be passed for deploying
// by default we will not erase disk, but an option is provided for the same.
// 1. Releases the Machine
// 2. Deploys the machine again with the parameters

func (m *MaasClient) Reclaim(ctx context.Context, req api.ReclaimBMRequest) error {
	if m.Client == nil {
		return errors.New("reclaim: client not initialized")
	}
	systemID := req.ResourceId
	eraseDisk := req.EraseDisk
	cloudInitScript := req.UserData
	bootSource := req.BootSource
	var err error
	machine, err := m.Client.Machine.Get(systemID)
	if err != nil {
		logrus.Errorf("Failed to get machine: %v", err)
		return errors.Wrap(err, "failed to get machine")
	}

	con_interface := ipmi.InterfaceLanplus
	if req.IpmiInterface == nil {
		req.IpmiInterface = &api.IpmiType{IpmiInterface: &api.IpmiType_Lanplus{}}
	} else if req.IpmiInterface.IpmiInterface == nil {
		req.IpmiInterface.IpmiInterface = &api.IpmiType_Lanplus{}
	}
	switch req.IpmiInterface.IpmiInterface.(type) {
	case *api.IpmiType_Lan:
		con_interface = ipmi.InterfaceLan
	case *api.IpmiType_Lanplus:
		con_interface = ipmi.InterfaceLanplus
	case *api.IpmiType_OpenIpmi:
		con_interface = ipmi.InterfaceOpen
	case *api.IpmiType_Tool:
		con_interface = ipmi.InterfaceTool
	}

	// Handle machine in "New" state - needs commissioning first
	if strings.EqualFold(machine.StatusName, "New") {
		logrus.Infof("%s Machine %s is in New state, commissioning first", ctx, systemID)

		// Commission the machine
		_, err = m.Client.Machine.Commission(systemID, &entity.MachineCommissionParams{})
		if err != nil {
			logrus.Errorf("%s Failed to commission machine: %v", ctx, err)
			return errors.Wrap(err, "failed to commission machine")
		}

		// Wait for commissioning to complete (machine becomes "Ready")
		logrus.Infof("%s Waiting for machine %s to complete commissioning", ctx, systemID)
		err = m.waitForMachineState(ctx, systemID, "Ready", 10*time.Minute)
		if err != nil {
			logrus.Errorf("%s Failed waiting for commissioning: %v", ctx, err)
			return errors.Wrap(err, "failed waiting for commissioning to complete")
		}

		// Refresh machine state after commissioning
		machine, err = m.Client.Machine.Get(systemID)
		if err != nil {
			logrus.Errorf("Failed to get machine after commissioning: %v", err)
			return errors.Wrap(err, "failed to get machine after commissioning")
		}
	}

	if !req.ManualPowerControl {
		//Set machine to PXE Boot
		logrus.Infof("Setting %s to PXE boot over %s", machine.Hostname, con_interface)
		err = m.SetMachine2PXEBoot(ctx, systemID, req.PowerCycle, req.IpmiInterface)
		if err != nil {
			logrus.Errorf("%s Failed to set machine to PXE boot: %v", ctx, err)
			return errors.Wrap(err, "failed to set machine to pxe boot")
		}
	}
	// Check if machine needs to be released first
	if strings.EqualFold(machine.StatusName, "Ready") || strings.EqualFold(machine.StatusName, "Releasing") || strings.EqualFold(machine.StatusName, "released") {
		logrus.Infof("%s Machine %s is already %s", ctx, systemID, machine.StatusName)
	} else if !strings.EqualFold(machine.StatusName, "Allocated") {
		// Release the machine (unless it's already Allocated, which is ready for deployment)
		logrus.Infof("%s Releasing machine %s from state %s", ctx, systemID, machine.StatusName)
		_, err = m.Client.Machine.Release(systemID, &entity.MachineReleaseParams{
			Comment: "vJailbreak: Releasing machine for re-deployment",
			Erase:   eraseDisk,
		})
		if err != nil {
			logrus.Errorf("%s Failed to release machine: %v", ctx, err)
			return errors.Wrap(err, "failed to release machine")
		}

		// Wait for machine to be released (becomes "Ready")
		logrus.Infof("%s Waiting for machine %s to be released", ctx, systemID)
		err = m.waitForMachineState(ctx, systemID, "Ready", 5*time.Minute)
		if err != nil {
			logrus.Errorf("%s Failed waiting for release: %v", ctx, err)
			return errors.Wrap(err, "failed waiting for machine to be released")
		}

		// Refresh machine state after release
		machine, err = m.Client.Machine.Get(systemID)
		if err != nil {
			logrus.Errorf("Failed to get machine after release: %v", err)
			return errors.Wrap(err, "failed to get machine after release")
		}
	}

	// Allocate the machine before deployment (only if not already Allocated)
	if !strings.EqualFold(machine.StatusName, "Allocated") {
		logrus.Infof("%s Allocating machine %s", ctx, systemID)
		_, err = m.Client.Machines.Allocate(&entity.MachineAllocateParams{
			SystemID: systemID,
			Comment:  "vJailbreak: Allocating machine for deployment",
		})
		if err != nil {
			logrus.Errorf("%s Failed to allocate machine: %v", ctx, err)
			return errors.Wrap(err, "failed to allocate machine")
		}

		// Wait for allocation to complete
		logrus.Infof("%s Waiting for machine %s to be allocated", ctx, systemID)
		err = m.waitForMachineState(ctx, systemID, "Allocated", 1*time.Minute)
		if err != nil {
			logrus.Errorf("%s Failed waiting for allocation: %v", ctx, err)
			return errors.Wrap(err, "failed waiting for machine to be allocated")
		}
	} else {
		logrus.Infof("%s Machine %s is already Allocated, skipping allocation step", ctx, systemID)
	}

	if !req.ManualPowerControl {
		//Call PXE boot again to deploy the machine
		err = m.SetMachine2PXEBoot(ctx, systemID, req.PowerCycle, req.IpmiInterface)
		if err != nil {
			logrus.Errorf("%s Failed to set machine to PXE boot: %v", ctx, err)
			return errors.Wrap(err, "failed to set machine to pxe boot")
		}
	}
	// Deploy the machine again
	logrus.Infof("%s Deploying machine %s", ctx, systemID)
	_, err = m.Client.Machine.Deploy(systemID, &entity.MachineDeployParams{
		Comment:      "vJailbreak: Deploying machine with cloud-init script",
		UserData:     cloudInitScript,
		DistroSeries: bootSource.Release,
	})
	if err != nil {
		logrus.Errorf("Failed to deploy machine: %v", err)
		return err
	}

	return nil
}

func (m *MaasClient) ListBootSource(ctx context.Context) ([]api.BootsourceSelections, error) {
	if m.Client == nil {
		return nil, errors.New("client not initialized")
	}
	// Get boot sources
	bootSources, err := m.Client.BootSources.Get()
	if err != nil {
		logrus.Errorf("cannot get boot sources, err: %v", err)
		return nil, err
	}
	bootSource_ids := make(map[int]int)
	// Get Boot source using the ID
	for _, v := range bootSources {
		bs, err := m.Client.BootSource.Get(v.ID)
		if err != nil {
			logrus.Errorf("cannot get boot source, err: %v", err)
			return nil, err
		}
		bootSource_ids[v.ID] = bs.ID
	}

	var bootSourcesList []api.BootsourceSelections
	// Get boot source selections for that ID
	for _, v := range bootSource_ids {
		bootSourceSelections, err := m.Client.BootSourceSelections.Get(v)
		if err != nil {
			logrus.Errorf("cannot get boot source selections, err: %v", err)
			return nil, err
		}

		for _, v := range bootSourceSelections {
			bs, err := m.Client.BootSourceSelection.Get(v.BootSourceID, v.ID)
			if err != nil {
				logrus.Errorf("cannot get boot source, err: %v", err)
				return nil, errors.Wrap(err, "failed to get boot source")
			}
			bootSourcesList = append(bootSourcesList, api.BootsourceSelections{
				OS:           bs.OS,
				Release:      bs.Release,
				ResourceURI:  bs.ResourceURI,
				Arches:       bs.Arches,
				Subarches:    bs.Subarches,
				Labels:       bs.Labels,
				ID:           int32(v.ID),
				BootSourceID: int32(v.BootSourceID),
			})
		}
	}
	return bootSourcesList, nil
}

func (m *MaasClient) GetPowerParameters(ctx context.Context, systemID string) (map[string]interface{}, error) {
	if m.Client == nil {
		return nil, errors.New("client not initialized")
	}
	powerParams, err := m.Client.Machine.GetPowerParameters(systemID)
	if err != nil {
		logrus.Errorf("Failed to get power parameters: %v", err)
		return nil, errors.Wrap(err, "failed to get power parameters")
	}
	return powerParams, nil
}

func (m *MaasClient) GetIPMIClient(ctx context.Context, host, username, password string, req *api.IpmiType) (*ipmi.Client, error) {
	// Configure IPMI connection
	con_interface := ipmi.InterfaceLanplus
	switch req.IpmiInterface.(type) {
	case *api.IpmiType_Lan:
		con_interface = ipmi.InterfaceLan
	case *api.IpmiType_Lanplus:
		con_interface = ipmi.InterfaceLanplus
	case *api.IpmiType_OpenIpmi:
		con_interface = ipmi.InterfaceOpen
	case *api.IpmiType_Tool:
		con_interface = ipmi.InterfaceTool
	}
	logrus.Infof("Creating IPMI client for host %s with interface %v", host, con_interface)

	// Create client with extended timeouts to accommodate slower BMCs
	config, err := ipmi.NewClient(host, 623, username, password)
	if err != nil {
		logrus.Errorf("Failed to create IPMI client: %v", err)
		return nil, errors.Wrap(err, "failed to create ipmi client")
	}

	// Set longer timeout for operations
	config = config.WithTimeout(30 * time.Second)

	// Set the interface type
	config = config.WithInterface(con_interface)

	logrus.Debugf("IPMI client configured with interface %v", con_interface)
	return config, nil
}

// TODO:
// Check if we can get PowerState Options and if they are IPMI
// we should try using goipmi to set the bootdev to PXE
// this would trigger MaaS to boot an ephemeral image on next reboot for this host
// then we can move to Deploy state for the host.
// waitForMachineState polls the machine status until it reaches the desired state or times out
func (m *MaasClient) waitForMachineState(ctx context.Context, systemID string, desiredState string, timeout time.Duration) error {
	if m.Client == nil {
		return errors.New("client not initialized")
	}

	deadline := time.Now().Add(timeout)
	pollInterval := 10 * time.Second

	for time.Now().Before(deadline) {
		machine, err := m.Client.Machine.Get(systemID)
		if err != nil {
			logrus.Errorf("Failed to get machine status: %v", err)
			return errors.Wrap(err, "failed to get machine status")
		}

		logrus.Debugf("Machine %s current state: %s (waiting for %s)", systemID, machine.StatusName, desiredState)

		if strings.EqualFold(machine.StatusName, desiredState) {
			logrus.Infof("Machine %s reached desired state: %s", systemID, desiredState)
			return nil
		}

		// Check for failed states
		if strings.EqualFold(machine.StatusName, "Failed") ||
			strings.EqualFold(machine.StatusName, "Failed commissioning") ||
			strings.EqualFold(machine.StatusName, "Failed deployment") {
			return errors.Errorf("machine %s entered failed state: %s - %s", systemID, machine.StatusName, machine.StatusMessage)
		}

		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "context cancelled while waiting for machine state")
		case <-time.After(pollInterval):
			// Continue polling
		}
	}

	return errors.Errorf("timeout waiting for machine %s to reach state %s after %v", systemID, desiredState, timeout)
}

func (m *MaasClient) SetMachine2PXEBoot(ctx context.Context, systemID string, power_cycle bool, ipmi_interface *api.IpmiType) error {
	if m.Client == nil {
		return errors.New("client not initialized")
	}
	machine, err := m.Client.Machine.Get(systemID)
	if err != nil {
		logrus.Errorf("Failed to get machine: %v", err)
		return errors.Wrap(err, "failed to get machine")
	}
	if !strings.EqualFold(machine.PowerType, "ipmi") {
		logrus.Errorf("Machine %s does not support IPMI power type", systemID)
		return errors.New("machine does not support IPMI power type")
	}

	powerParams, err := m.GetPowerParameters(ctx, systemID)
	if err != nil {
		logrus.Errorf("Failed to get power parameters: %v", err)
		return errors.Wrap(err, "failed to get power parameters")
	}
	host := powerParams["power_address"]
	username := powerParams["power_user"]
	password := powerParams["power_pass"]
	if host == nil || username == nil || password == nil {
		logrus.Errorf("Failed to get power parameters: %v", err)
		return errors.New("failed to get power parameters")
	}

	config, err := m.GetIPMIClient(ctx, host.(string), username.(string), password.(string), ipmi_interface)
	if err != nil {
		logrus.Errorf("Failed to create IPMI client: %v", err)
		return errors.Wrap(err, "failed to create ipmi client")
	}
	// Open IPMI connection
	err = config.Connect(ctx)
	if err != nil {
		logrus.Errorf("Failed to open IPMI connection: %v", err)
		return errors.Wrap(err, "failed to open ipmi connection")
	}
	defer config.Close(ctx)

	// Set boot device to PXE
	bootDevice := ipmi.BootDeviceSelectorForcePXE
	boot_type := ipmi.BIOSBootTypeLegacy
	if machine.BiosBootMethod == "efi" {
		boot_type = ipmi.BIOSBootTypeEFI
	}

	logrus.Debugf("IPMI connection details: host=%s, username=%s, ipmi_interface=%s, boot_type=%s", host, username, ipmi_interface, boot_type)

	err = config.SetBootDevice(ctx, bootDevice, boot_type, false)
	if err != nil {
		logrus.Errorf("Failed to set boot device to PXE: %v", err)
		return errors.Wrap(err, "failed to set boot device to pxe")
	}

	// Power cycle the machine to apply PXE boot based on the flag passed in [optional]
	powerState, err := config.GetChassisStatus(ctx)
	if err != nil {
		logrus.Errorf("Failed to get chassis status: %v", err)
		return errors.Wrap(err, "failed to get chassis status")
	}
	logrus.Infof("current power state for %s is %s", systemID, powerState.ChassisIdentifyState)

	//If machine is on, perform a power reset
	if power_cycle && powerState.PowerIsOn {
		_, err = config.ChassisControl(ctx, ipmi.ChassisControlPowerDown)
		if err != nil {
			logrus.Errorf("Failed to power down machine %s: %v, current power state: %v", systemID, err, powerState)
			return errors.Wrapf(err, "failed to power down machine %s", systemID)
		}

		_, err = config.ChassisControl(ctx, ipmi.ChassisControlPowerUp)
		if err != nil {
			logrus.Errorf("Failed to power up machine %s after power down: %v, current power state: %v", systemID, err, powerState)
			return errors.Wrapf(err, "failed to power up machine %s after power down", systemID)
		}
	} else if !powerState.PowerIsOn {
		_, err = config.ChassisControl(ctx, ipmi.ChassisControlPowerUp)
		if err != nil {
			logrus.Errorf("Failed to power up machine %s: %v, current power state: %v", systemID, err, powerState)
			return errors.Wrapf(err, "failed to power up machine %s", systemID)
		}
	}
	logrus.Infof("Successfully set machine %s to PXE boot", systemID)
	return nil
}
