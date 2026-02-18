package server

import (
	"context"
	"errors"

	api "github.com/platform9/vjailbreak/pkg/vpwned/api/proto/v1/service"
	"github.com/platform9/vjailbreak/pkg/vpwned/sdk/providers"
	"github.com/sirupsen/logrus"
)

type providersGRPC struct {
	api.UnimplementedBMProviderServer
	Creds providers.BMAccessInfo
}

func (p *providersGRPC) whichProvider(a *api.BMProvisionerAccessInfo) string {
	switch a.GetProviders().(type) {
	case *api.BMProvisionerAccessInfo_Maas:
		return "MAAS"
	case *api.BMProvisionerAccessInfo_UnknownProvider:
		return "Unknown"
	default:
		logrus.Errorf("Unknown provider type: %v", a.GetProviders())
		return "Unknown"
	}
}

func (p *providersGRPC) populateCredsFromAccessInfo(accessInfo *api.BMProvisionerAccessInfo) {
	p.Creds = providers.BMAccessInfo{
		Username:    accessInfo.Username,
		Password:    accessInfo.Password,
		APIKey:      accessInfo.ApiKey,
		BaseURL:     accessInfo.BaseUrl,
		UseInsecure: accessInfo.UseInsecure,
		Provider:    p.whichProvider(accessInfo),
	}
	logrus.Infof("Provider: %s", p.Creds.Provider)
}

func (p *providersGRPC) ListMachines(ctx context.Context, in *api.BMListMachinesRequest) (*api.BMListMachinesResponse, error) {
	retval := &api.BMListMachinesResponse{}
	p.populateCredsFromAccessInfo(in.AccessInfo)
	provider, err := providers.GetProvider(p.Creds.Provider)
	if err != nil {
		return retval, errors.New("unknown provider")
	}
	err = provider.Connect(p.Creds)
	if err != nil {
		return retval, err
	}
	defer provider.Disconnect()
	info, err := provider.ListResources(ctx)
	if err != nil {
		return retval, err
	}
	for _, i := range info {
		retval.Machines = append(retval.Machines, &api.MachineInfo{
			Id:              i.Id,
			Fqdn:            i.Fqdn,
			Os:              i.Os,
			PowerState:      i.PowerState,
			Hostname:        i.Hostname,
			Architecture:    i.Architecture,
			Memory:          i.Memory,
			CpuCount:        i.CpuCount,
			CpuSpeed:        i.CpuSpeed,
			BootDiskSize:    i.BootDiskSize,
			Status:          i.Status,
			StatusMessage:   i.StatusMessage,
			StatusAction:    i.StatusAction,
			Description:     i.Description,
			Domain:          i.Domain,
			Zone:            i.Zone,
			Pool:            i.Pool,
			TagNames:        i.TagNames,
			VmHost:          i.VmHost,
			Netboot:         i.Netboot,
			EphemeralDeploy: i.EphemeralDeploy,
			PowerParams:     i.PowerParams,
			PowerType:       i.PowerType,
			BiosBootMethod:  i.BiosBootMethod,
			HardwareUuid:    i.HardwareUuid,
			MacAddress:      i.MacAddress,
		})
	}
	return retval, nil
}
func (p *providersGRPC) GetResourceInfo(ctx context.Context, in *api.GetResourceInfoRequest) (*api.GetResourceInfoResponse, error) {
	retval := &api.GetResourceInfoResponse{}
	p.populateCredsFromAccessInfo(in.AccessInfo)
	provider, err := providers.GetProvider(p.Creds.Provider)
	if err != nil {
		return retval, errors.New("unknown provider")
	}
	err = provider.Connect(p.Creds)
	if err != nil {
		return nil, err
	}
	defer provider.Disconnect()
	info, err := provider.GetResourceInfo(ctx, in.ResourceId)
	if err != nil {
		return nil, err
	}
	return &api.GetResourceInfoResponse{
		Machine: &api.MachineInfo{
			Id:              info.Id,
			Fqdn:            info.Fqdn,
			Os:              info.Os,
			PowerState:      info.PowerState,
			Hostname:        info.Hostname,
			Architecture:    info.Architecture,
			Memory:          info.Memory,
			CpuCount:        info.CpuCount,
			CpuSpeed:        info.CpuSpeed,
			BootDiskSize:    info.BootDiskSize,
			Status:          info.Status,
			StatusMessage:   info.StatusMessage,
			StatusAction:    info.StatusAction,
			Description:     info.Description,
			Domain:          info.Domain,
			Zone:            info.Zone,
			Pool:            info.Pool,
			TagNames:        info.TagNames,
			VmHost:          info.VmHost,
			Netboot:         info.Netboot,
			EphemeralDeploy: info.EphemeralDeploy,
			PowerParams:     info.PowerParams,
			PowerType:       info.PowerType,
			BiosBootMethod:  info.BiosBootMethod,
			HardwareUuid:    info.HardwareUuid,
			MacAddress:      info.MacAddress,
		},
	}, nil
}
func (p *providersGRPC) SetResourcePower(ctx context.Context, in *api.SetResourcePowerRequest) (*api.SetResourcePowerResponse, error) {
	retval := &api.SetResourcePowerResponse{}
	p.populateCredsFromAccessInfo(in.AccessInfo)
	provider, err := providers.GetProvider(p.Creds.Provider)
	if err != nil {
		return retval, errors.New("unknown provider")
	}
	err = provider.Connect(p.Creds)
	if err != nil {
		return retval, err
	}
	defer provider.Disconnect()
	err = provider.SetResourcePower(ctx, in.ResourceId, in.PowerStatus)
	if err != nil {
		return retval, err
	}
	return retval, nil
}
func (p *providersGRPC) SetResourceBM2PXEBoot(ctx context.Context, in *api.SetResourceBM2PXEBootRequest) (*api.SetResourceBM2PXEBootResponse, error) {
	retval := &api.SetResourceBM2PXEBootResponse{}
	p.populateCredsFromAccessInfo(in.AccessInfo)
	provider, err := providers.GetProvider(p.Creds.Provider)
	if err != nil {
		return retval, errors.New("unknown provider")
	}
	err = provider.Connect(p.Creds)
	if err != nil {
		return retval, err
	}
	defer provider.Disconnect()
	var ipmi_interface *api.IpmiType
	switch in.IpmiInterface.(type) {
	case *api.SetResourceBM2PXEBootRequest_Lan:
		ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_Lan{}}
	case *api.SetResourceBM2PXEBootRequest_Lanplus:
		ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_Lanplus{}}
	case *api.SetResourceBM2PXEBootRequest_OpenIpmi:
		ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_OpenIpmi{}}
	case *api.SetResourceBM2PXEBootRequest_Tool:
		ipmi_interface = &api.IpmiType{IpmiInterface: &api.IpmiType_Tool{}}
	}
	err = provider.SetBM2PXEBoot(ctx, in.ResourceId, in.PowerCycle, ipmi_interface)
	if err != nil {
		return retval, err
	}
	return retval, nil
}

func (p *providersGRPC) WhoAmI(ctx context.Context, in *api.WhoAmIRequest) (*api.WhoAmIResponse, error) {
	return &api.WhoAmIResponse{ProviderName: p.Creds.Provider}, nil
}

func (p *providersGRPC) ListBootSource(ctx context.Context, in *api.ListBootSourceRequest) (*api.ListBootSourceResponse, error) {
	retval := &api.ListBootSourceResponse{}
	p.populateCredsFromAccessInfo(in.AccessInfo)
	provider, err := providers.GetProvider(p.Creds.Provider)
	if err != nil {
		return retval, errors.New("unknown provider")
	}
	err = provider.Connect(p.Creds)
	if err != nil {
		return retval, err
	}
	defer provider.Disconnect()
	bootSource, err := provider.ListBootSource(ctx, *in)
	if err != nil {
		return nil, err
	}
	for i := range bootSource {
		retval.BootSourceSelections = append(retval.BootSourceSelections, &bootSource[i])
	}
	return retval, nil
}

func (p *providersGRPC) ReclaimBM(ctx context.Context, in *api.ReclaimBMRequest) (*api.ReclaimBMResponse, error) {
	retval := &api.ReclaimBMResponse{}
	p.populateCredsFromAccessInfo(in.AccessInfo)
	provider, err := providers.GetProvider(p.Creds.Provider)
	if err != nil {
		return retval, errors.New("unknown provider")
	}
	err = provider.Connect(p.Creds)
	if err != nil {
		return retval, err
	}
	defer provider.Disconnect()
	err = provider.ReclaimBM(ctx, *in)
	if err != nil {
		return retval, err
	}
	return retval, nil
}

func (p *providersGRPC) StartBM(ctx context.Context, in *api.StartBMRequest) (*api.StartBMResponse, error) {
	retval := &api.StartBMResponse{}
	p.populateCredsFromAccessInfo(in.AccessInfo)
	provider, err := providers.GetProvider(p.Creds.Provider)
	if err != nil {
		return retval, errors.New("unknown provider")
	}
	err = provider.Connect(p.Creds)
	if err != nil {
		return retval, err
	}
	defer provider.Disconnect()
	_, err = provider.StartBM(ctx, *in)
	if err != nil {
		return retval, err
	}
	return retval, nil
}

func (p *providersGRPC) StopBM(ctx context.Context, in *api.StopBMRequest) (*api.StopBMResponse, error) {
	retval := &api.StopBMResponse{}
	p.populateCredsFromAccessInfo(in.AccessInfo)
	provider, err := providers.GetProvider(p.Creds.Provider)
	if err != nil {
		return retval, errors.New("unknown provider")
	}
	err = provider.Connect(p.Creds)
	if err != nil {
		return retval, err
	}
	defer provider.Disconnect()
	_, err = provider.StopBM(ctx, *in)
	if err != nil {
		return retval, err
	}
	return retval, nil
}
