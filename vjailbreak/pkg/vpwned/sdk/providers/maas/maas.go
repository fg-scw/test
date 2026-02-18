package maas

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	ipmi "github.com/bougou/go-ipmi"
	api "github.com/platform9/vjailbreak/pkg/vpwned/api/proto/v1/service"
	"github.com/platform9/vjailbreak/pkg/vpwned/sdk/providers"
	"github.com/platform9/vjailbreak/pkg/vpwned/sdk/providers/base"
	"github.com/sirupsen/logrus"
)

const (
	MaasProviderName = "maas"
)

// MaasAccessInfo contains credentials and connection details for MaaS
type MaasAccessInfo struct {
	APIKey      string
	BaseURL     string
	UseInsecure bool
}

// MaasProvider implements the Provider interface for MaaS
type MaasProvider struct {
	base.UnimplementedBaseProvider
	client *MaasClient
}

// Connect establishes a connection to the MaaS server
func (p *MaasProvider) Connect(auth providers.BMAccessInfo) error {
	accessInfo := MaasAccessInfo{
		BaseURL:     auth.BaseURL,
		APIKey:      auth.APIKey,
		UseInsecure: auth.UseInsecure,
	}
	// Create MaaS client
	client, err := NewMaasClient(accessInfo)
	if err != nil {
		return errors.Wrap(err, "failed to create maas client")
	}
	p.client = client
	return nil
}

// ListResources retrieves a list of machines
func (p *MaasProvider) ListResources(ctx context.Context) ([]api.MachineInfo, error) {
	return p.client.ListMachines(ctx)
}

// SetResourcePower changes the power state of a machine
func (p *MaasProvider) SetResourcePower(ctx context.Context, resourceID string, action api.PowerStatus) error {
	return p.client.SetMachinePower(ctx, resourceID, action)
}

// GetResourceInfo retrieves information about a machine
func (p *MaasProvider) GetResourceInfo(ctx context.Context, resourceID string) (api.MachineInfo, error) {
	if p.client == nil || p.client.Client == nil {
		return api.MachineInfo{}, errors.New("client not initialized")
	}
	machine, err := p.client.Client.Machine.Get(resourceID)
	if err != nil {
		return api.MachineInfo{}, errors.Wrap(err, "failed to get machine")
	}
	powerParams, err := p.client.Client.Machine.GetPowerParameters(resourceID)
	if err != nil {
		return api.MachineInfo{}, errors.Wrap(err, "failed to get power parameters")
	}
	pw_val := ""
	for k, v := range powerParams {
		pw_val += fmt.Sprintf("%s=%s\n", k, v)
	}
	return api.MachineInfo{
		Id:              machine.SystemID,
		Fqdn:            machine.FQDN,
		Os:              machine.OSystem,
		PowerState:      machine.PowerState,
		Hostname:        machine.Hostname,
		Architecture:    machine.Architecture,
		Memory:          fmt.Sprintf("%d", machine.Memory),
		CpuCount:        fmt.Sprintf("%d", machine.CPUCount),
		CpuSpeed:        fmt.Sprintf("%d", machine.CPUSpeed),
		BootDiskSize:    fmt.Sprintf("%d", machine.BootDisk.Size),
		Status:          machine.StatusName,
		StatusMessage:   machine.StatusMessage,
		StatusAction:    machine.StatusAction,
		Description:     machine.Description,
		Domain:          machine.Domain.Name,
		Zone:            machine.Zone.Name,
		Pool:            machine.Pool.Name,
		TagNames:        strings.Join(machine.TagNames, ","),
		Netboot:         machine.Netboot,
		EphemeralDeploy: machine.EphemeralDeploy,
		PowerType:       machine.PowerType,
		PowerParams:     pw_val,
		BiosBootMethod:  machine.BiosBootMethod,
		HardwareUuid:    machine.HardwareUUID,
		MacAddress:      machine.BootInterface.MACAddress,
	}, nil
}

func (p *MaasProvider) ListBootSource(ctx context.Context, req api.ListBootSourceRequest) ([]api.BootsourceSelections, error) {
	var err error
	if p.client == nil || p.client.Client == nil {
		err = p.Connect(providers.BMAccessInfo{
			BaseURL:     req.AccessInfo.BaseUrl,
			APIKey:      req.AccessInfo.ApiKey,
			UseInsecure: req.AccessInfo.UseInsecure,
		})
		if err != nil {
			return nil, errors.Wrap(err, "List Boot Source Failed")
		}
	}
	return p.client.ListBootSource(ctx)
}

func (p *MaasProvider) Disconnect() error {
	if p.client == nil {
		return nil
	}
	return nil
}

func (p *MaasProvider) SetBM2PXEBoot(ctx context.Context, resourceID string, power_cycle bool, ipmi_interface *api.IpmiType) error {
	return p.client.SetMachine2PXEBoot(ctx, resourceID, power_cycle, ipmi_interface)
}

func (p *MaasProvider) ReclaimBM(ctx context.Context, req api.ReclaimBMRequest) error {
	var err error
	if p.client == nil || p.client.Client == nil {
		err = p.Connect(providers.BMAccessInfo{
			BaseURL:     req.AccessInfo.BaseUrl,
			APIKey:      req.AccessInfo.ApiKey,
			UseInsecure: req.AccessInfo.UseInsecure,
		})
		if err != nil {
			return errors.Wrap(err, "Reclaim VM Failed")
		}
	}
	//Steps to reclain the host.
	return p.client.Reclaim(ctx, req)
}

func (p *MaasProvider) WhoAmI() string {
	return MaasProviderName
}

func (p *MaasProvider) DeployMachine(ctx context.Context, req api.DeployMachineRequest) (api.DeployMachineResponse, error) {
	if p.client == nil || p.client.Client == nil {
		return api.DeployMachineResponse{}, errors.New("client not initialized")
	}
	m, err := p.client.GetMachineFromID(ctx, req.ResourceId)
	if err != nil {
		return api.DeployMachineResponse{}, errors.Wrap(err, "Deploy Machine Failed")
	}
	logrus.Debugf("machine found")
	err = p.client.DeployMachine(ctx, m, req.UserData, req.OsReleaseName)
	if err != nil {
		return api.DeployMachineResponse{}, errors.Wrap(err, "Deploy Machine Failed")
	}
	return api.DeployMachineResponse{Success: true}, nil
}

func (p *MaasProvider) IsBMReady(ctx context.Context, req api.IsBMReadyRequest) (api.IsBMReadyResponse, error) {
	if p.client == nil || p.client.Client == nil {
		return api.IsBMReadyResponse{}, errors.New("client not initialized")
	}
	m, err := p.client.GetMachineFromID(ctx, req.ResourceId)
	if err != nil {
		return api.IsBMReadyResponse{}, errors.Wrap(err, "IsBMReady Failed")
	}
	return api.IsBMReadyResponse{IsReady: m.StatusName == "Ready"}, nil
}

func (p *MaasProvider) IsBMRunning(ctx context.Context, req api.IsBMRunningRequest) (api.IsBMRunningResponse, error) {
	if p.client == nil || p.client.Client == nil {
		return api.IsBMRunningResponse{}, errors.New("client not initialized")
	}
	m, err := p.client.GetMachineFromID(ctx, req.ResourceId)
	if err != nil {
		return api.IsBMRunningResponse{}, errors.Wrap(err, "IsBMRunning Failed")
	}
	return api.IsBMRunningResponse{IsRunning: m.StatusName == "Running"}, nil
}

func (p *MaasProvider) StartBM(ctx context.Context, req api.StartBMRequest) (api.StartBMResponse, error) {
	if p.client == nil || p.client.Client == nil {
		return api.StartBMResponse{}, errors.New("client not initialized")
	}
	m, err := p.client.GetMachineFromID(ctx, req.ResourceId)
	if err != nil {
		return api.StartBMResponse{}, errors.Wrap(err, "StartBM Failed")
	}
	powerParams, err := p.client.GetPowerParameters(ctx, m.SystemID)
	if err != nil {
		return api.StartBMResponse{}, errors.Wrap(err, "StartBM Failed")
	}
	host := powerParams["power_address"]
	username := powerParams["power_user"]
	password := powerParams["power_pass"]
	if host == nil || username == nil || password == nil {
		return api.StartBMResponse{}, errors.New("failed to get power parameters")
	}
	config, err := p.client.GetIPMIClient(ctx, host.(string), username.(string), password.(string), req.IpmiInterface)
	if err != nil {
		return api.StartBMResponse{}, errors.Wrap(err, "StartBM Failed")
	}
	err = config.Connect(ctx)
	if err != nil {
		return api.StartBMResponse{}, errors.Wrap(err, "StartBM Failed")
	}
	defer config.Close(ctx)
	_, err = config.ChassisControl(ctx, ipmi.ChassisControlPowerUp)
	if err != nil {
		return api.StartBMResponse{}, errors.Wrap(err, "StartBM Failed")
	}
	return api.StartBMResponse{Success: true}, nil
}

func (p *MaasProvider) StopBM(ctx context.Context, req api.StopBMRequest) (api.StopBMResponse, error) {
	if p.client == nil || p.client.Client == nil {
		return api.StopBMResponse{}, errors.New("client not initialized")
	}
	m, err := p.client.GetMachineFromID(ctx, req.ResourceId)
	if err != nil {
		return api.StopBMResponse{}, errors.Wrap(err, "StopBM Failed")
	}
	powerParams, err := p.client.GetPowerParameters(ctx, m.SystemID)
	if err != nil {
		return api.StopBMResponse{}, errors.Wrap(err, "StopBM Failed")
	}
	host := powerParams["power_address"]
	username := powerParams["power_user"]
	password := powerParams["power_pass"]
	if host == nil || username == nil || password == nil {
		return api.StopBMResponse{}, errors.New("failed to get power parameters")
	}
	config, err := p.client.GetIPMIClient(ctx, host.(string), username.(string), password.(string), req.IpmiInterface)
	if err != nil {
		return api.StopBMResponse{}, errors.Wrap(err, "StopBM Failed")
	}
	err = config.Connect(ctx)
	if err != nil {
		return api.StopBMResponse{}, errors.Wrap(err, "StopBM Failed")
	}
	defer config.Close(ctx)
	_, err = config.ChassisControl(ctx, ipmi.ChassisControlPowerDown)
	if err != nil {
		return api.StopBMResponse{}, errors.Wrap(err, "StopBM Failed")
	}
	return api.StopBMResponse{Success: true}, nil
}

func init() {
	providers.RegisterProvider(MaasProviderName, &MaasProvider{client: nil})
}
