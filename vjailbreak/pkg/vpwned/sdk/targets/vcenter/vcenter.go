package vcenter

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	commonutils "github.com/platform9/vjailbreak/pkg/common/utils"
	api "github.com/platform9/vjailbreak/pkg/vpwned/api/proto/v1/service"
	"github.com/platform9/vjailbreak/pkg/vpwned/sdk/targets"
	"github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

type Vcenter struct {
	targets.Targets
}

type VMCenterAccessInfo struct {
	targets.AccessInfo
	Datacenter string
}

type VCenterOpts struct {
	key   string
	value string
}

func NewVCenterCreds(a targets.AccessInfo, opts ...VCenterOpts) VMCenterAccessInfo {
	creds := VMCenterAccessInfo{
		AccessInfo: a,
	}
	for _, opt := range opts {
		switch opt.key {
		case "datacenter":
			creds.Datacenter = opt.value
		}
	}
	return creds
}

func findPowerStatus(state types.VirtualMachinePowerState) api.PowerStatus {
	if strings.EqualFold(string(state), "poweredOn") {
		return api.PowerStatus_POWERED_ON
	} else if strings.EqualFold(string(state), "poweredOff") {
		return api.PowerStatus_POWERED_OFF
	} else if strings.EqualFold(string(state), "suspended") {
		return api.PowerStatus_POWERED_ON
	}
	return api.PowerStatus_POWER_STATE_UNKNOWN
}

func findBootDevice(devices []types.BaseVirtualMachineBootOptionsBootableDevice) string {
	for _, bootDevice := range devices {
		// Type switch to handle different device types
		switch device := bootDevice.(type) {
		case *types.VirtualMachineBootOptionsBootableCdromDevice:
			return "cdrom"
		case *types.VirtualMachineBootOptionsBootableDiskDevice:
			return "disk"
		case *types.VirtualMachineBootOptionsBootableEthernetDevice:
			return "network"
		default:
			logrus.Warnf("Unknown Boot Device Type: %T", device)
		}
	}
	return "unknown"
}

func (v *Vcenter) connect(ctx context.Context, a api.TargetAccessInfo) (*govmomi.Client, *find.Finder, error) {
	logrus.Info("Connecting to vCenter...")
	// Parse vCenter URL
	u, err := commonutils.NormalizeVCenterURL(a.HostnameOrIp)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse vCenter URL: %v", err)
	}

	// Set credentials
	u.User = url.UserPassword(a.Username, a.Password)

	// Create vSphere client
	client, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to vCenter: %v", err)
	}

	// Create finder
	finder := find.NewFinder(client.Client, true)

	// If VMCenterAccessInfo is provided and Datacenter is set, set the datacenter
	if a.Datacenter != "" {
		dc, err := finder.Datacenter(ctx, a.Datacenter)
		if err != nil {
			client.Logout(ctx)
			return nil, nil, fmt.Errorf("failed to find datacenter: %v", err)
		}
		finder.SetDatacenter(dc)
	}
	logrus.Info("Connected to vCenter")
	return client, finder, nil
}

func (v *Vcenter) ListVMs(ctx context.Context, a api.TargetAccessInfo) ([]targets.VMInfo, error) {

	// Connect to vCenter
	client, _, err := v.connect(ctx, a)
	if err != nil {
		return nil, err
	}
	defer client.Logout(ctx)
	logrus.Info("Connected to vCenter")

	// Create container view of all virtual machines
	m := view.NewManager(client.Client)
	containerView, err := m.CreateContainerView(ctx, client.Client.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create container view: %v", err)
	}
	defer containerView.Destroy(ctx)

	// Retrieve all VMs
	var vms []mo.VirtualMachine
	err = containerView.Retrieve(ctx, []string{"VirtualMachine"}, []string{"summary", "runtime.powerState", "config.bootOptions"}, &vms)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve VMs: %v", err)
	}

	// Convert VMs to VMInfo
	vmInfos := make([]targets.VMInfo, len(vms))
	for i, vmRef := range vms {
		// Populate VM info
		vmInfos[i] = targets.VMInfo{
			Name:        vmRef.Summary.Config.Name,
			GuestOS:     vmRef.Summary.Config.GuestFullName,
			PowerStatus: findPowerStatus(vmRef.Runtime.PowerState),
			CPU:         int64(vmRef.Summary.Config.NumCpu),
			Memory:      int64(vmRef.Summary.Config.MemorySizeMB),
			IPv4Addr:    vmRef.Summary.Guest.IpAddress,
			IPv6Addr:    vmRef.Summary.Guest.IpAddress,
		}

		// Set boot device if available
		if vmRef.Config != nil && vmRef.Config.BootOptions != nil {
			vmInfos[i].BootDevice = findBootDevice(vmRef.Config.BootOptions.BootOrder)
		}
	}

	return vmInfos, nil
}

func (v *Vcenter) GetVM(ctx context.Context, a api.TargetAccessInfo, name string) (targets.VMInfo, error) {
	// Connect to vCenter
	client, finder, err := v.connect(ctx, a)
	if err != nil {
		return targets.VMInfo{}, err
	}
	defer client.Logout(ctx)

	// Find specific VM
	vm, err := finder.VirtualMachine(ctx, name)
	if err != nil {
		return targets.VMInfo{}, fmt.Errorf("VM not found: %v", err)
	}

	// Populate VM info
	vmInfo := targets.VMInfo{
		Name: vm.Name(),
	}

	// Get power state
	powerState, err := vm.PowerState(ctx)
	if err == nil {
		vmInfo.PowerStatus = findPowerStatus(powerState)
	}

	return vmInfo, nil
}

func (v *Vcenter) ReclaimVM(ctx context.Context, a api.TargetAccessInfo, name string, args ...string) error {
	// Connect to vCenter
	client, finder, err := v.connect(ctx, a)
	if err != nil {
		return err
	}
	defer client.Logout(ctx)

	// Find specific VM
	vm, err := finder.VirtualMachine(ctx, name)
	if err != nil {
		return fmt.Errorf("VM not found: %v", err)
	}

	// Power off VM if it's running
	powerState, err := vm.PowerState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get VM power state: %v", err)
	}

	if powerState == types.VirtualMachinePowerStatePoweredOn {
		task, err := vm.PowerOff(ctx)
		if err != nil {
			return fmt.Errorf("failed to power off VM: %v", err)
		}

		// Wait for power off to complete
		if err := task.Wait(ctx); err != nil {
			return fmt.Errorf("error waiting for VM power off: %v", err)
		}
	}

	// Destroy/delete the VM
	task, err := vm.Destroy(ctx)
	if err != nil {
		return fmt.Errorf("failed to destroy VM: %v", err)
	}

	// Wait for destroy to complete
	if err := task.Wait(ctx); err != nil {
		return fmt.Errorf("error waiting for VM destruction: %v", err)
	}

	return nil
}

func (v *Vcenter) CordonHost(ctx context.Context, a api.TargetAccessInfo, esxi_name string) error {
	// Connect to vCenter
	client, finder, err := v.connect(ctx, a)
	if err != nil {
		return err
	}
	defer client.Logout(ctx)

	// Find specific ESXi host
	esxi, err := finder.HostSystem(ctx, esxi_name)
	if err != nil {
		return fmt.Errorf("ESXi host not found: %v", err)
	}

	// Enter maintenance mode
	task, err := esxi.EnterMaintenanceMode(ctx, 0, true, &types.HostMaintenanceSpec{})
	if err != nil {
		return fmt.Errorf("failed to enter maintenance mode: %v", err)
	}

	// Wait for maintenance mode to complete
	if err := task.Wait(ctx); err != nil {
		return fmt.Errorf("error waiting for ESXi host maintenance mode: %v", err)
	}

	return nil
}

func (v *Vcenter) UnCordonHost(ctx context.Context, a api.TargetAccessInfo, esxi_name string) error {
	// Connect to vCenter
	client, finder, err := v.connect(ctx, a)
	if err != nil {
		return err
	}
	defer client.Logout(ctx)

	// Find specific ESXi host
	esxi, err := finder.HostSystem(ctx, esxi_name)
	if err != nil {
		return fmt.Errorf("ESXi host not found: %v", err)
	}

	// Enter maintenance mode
	task, err := esxi.ExitMaintenanceMode(ctx, 0)
	if err != nil {
		return fmt.Errorf("failed to exit maintenance mode: %v", err)
	}

	// Wait for maintenance mode to complete
	if err := task.Wait(ctx); err != nil {
		return fmt.Errorf("error waiting for ESXi host maintenance mode: %v", err)
	}

	return nil
}

func (v *Vcenter) ListHosts(ctx context.Context, a api.TargetAccessInfo) (*api.ListHostsResponse, error) {
	// Connect to vCenter
	client, finder, err := v.connect(ctx, a)
	if err != nil {
		return nil, err
	}
	defer client.Logout(ctx)

	if a.Datacenter != "" {
		// Set datacenter
		dc, err := finder.Datacenter(ctx, a.Datacenter)
		if err != nil {
			return nil, fmt.Errorf("failed to find datacenter %s: %v", a.Datacenter, err)
		}
		finder.SetDatacenter(dc)
	}

	// Create container view of all virtual machines
	m := view.NewManager(client.Client)
	var hosts []mo.HostSystem
	containerView, err := m.CreateContainerView(ctx, client.ServiceContent.RootFolder, []string{"HostSystem"}, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create container view: %v", err)
	}
	defer containerView.Destroy(ctx)

	// Retrieve all hosts with necessary properties
	// Include network configuration to get IP addresses
	err = containerView.Retrieve(ctx, []string{"HostSystem"}, []string{
		"name",
		"parent",
		"hardware.systemInfo",
		"config.network",
		"summary.managementServerIp",
	}, &hosts)
	if err != nil {
		logrus.Fatalf("Failed to retrieve host information: %s", err)
	}

	// Extract host names and IP addresses
	hostNames := make([]*api.ListHostsResponseItem, 0, len(hosts))
	for _, host := range hosts {
		// First check if management IP is available in summary
		ip := host.Summary.ManagementServerIp

		// If no IP in summary, try to get from network configuration
		if ip == "" && host.Config != nil && host.Config.Network != nil {
			// Check virtual NICs for management network IP
			for _, vnic := range host.Config.Network.Vnic {
				if vnic.Spec.Ip != nil && vnic.Spec.Ip.IpAddress != "" {
					ip = vnic.Spec.Ip.IpAddress
					break
				}
			}
		}
		var macAddresses []string
		if host.Config != nil && host.Config.Network != nil {
			for _, pnic := range host.Config.Network.Pnic {
				macAddresses = append(macAddresses, pnic.Mac)
			}
		}

		hostNames = append(hostNames, &api.ListHostsResponseItem{
			Host:       host.Name,
			Ip:         ip,
			BiosUuid:   host.Hardware.SystemInfo.Uuid,
			Serial:     host.Hardware.SystemInfo.SerialNumber,
			MacAddress: macAddresses,
		})
	}
	logrus.Debugf("returned %d hosts as response", len(hostNames))

	return &api.ListHostsResponse{Hosts: hostNames}, nil
}

func init() {
	targets.RegisterTarget("vcenter", &Vcenter{})
}
