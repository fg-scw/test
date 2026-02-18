package server

import (
	"context"

	api "github.com/platform9/vjailbreak/pkg/vpwned/api/proto/v1/service"
	"github.com/platform9/vjailbreak/pkg/vpwned/sdk/targets"
	"github.com/sirupsen/logrus"
)

type targetVcenterGRPC struct {
	api.UnimplementedVCenterServer
}

func (p *targetVcenterGRPC) populateCredsFromAccessInfo(accessInfo *api.TargetAccessInfo) *targets.AccessInfo {
	return &targets.AccessInfo{
		Username:     accessInfo.Username,
		Password:     accessInfo.Password,
		HostnameOrIP: accessInfo.HostnameOrIp,
		Port:         accessInfo.Port,
		UseInsecure:  accessInfo.UseInsecure,
		Datacenter:   accessInfo.Datacenter,
	}
}

func (p *targetVcenterGRPC) getTargetFromRequest(in *api.Targets) string {
	switch in.Target.(type) {
	case *api.Targets_Vcenter:
		return "vcenter"
	case *api.Targets_Pcd:
		return "pcd"
	case *api.Targets_Unknown:
		return "unknown"
	default:
		return "unknown"
	}
}

func (p *targetVcenterGRPC) ListVMs(ctx context.Context, in *api.ListVMsRequest) (*api.ListVMsResponse, error) {
	retval := &api.ListVMsResponse{}
	//creds := p.populateCredsFromAccessInfo(in.AccessInfo)
	target := p.getTargetFromRequest(in.Target)
	logrus.Debugf("Trying to get target %s", target)
	t, err := targets.GetTarget(target)
	if err != nil {
		return retval, err
	}
	vmList, err := t.ListVMs(ctx, *in.AccessInfo)
	if err != nil {
		return retval, err
	}
	for _, v := range vmList {
		bd, ok := api.BootDevice_value[v.BootDevice]
		if !ok {
			bd = int32(api.BootDevice_BOOT_DEVICE_UNKNOWN)
		}
		retval.Vms = append(retval.Vms, &api.VMInfo{
			Name:        v.Name,
			Cpu:         v.CPU,
			Memory:      v.Memory,
			Ipv4Addr:    v.IPv4Addr,
			Ipv6Addr:    v.IPv6Addr,
			PowerStatus: v.PowerStatus,
			BootDevice:  api.BootDevice(bd),
			GuestOs:     v.GuestOS,
		})
	}
	return retval, nil
}

func (p *targetVcenterGRPC) GetVM(ctx context.Context, in *api.GetVMRequest) (*api.GetVMResponse, error) {
	retval := &api.GetVMResponse{}
	//creds := p.populateCredsFromAccessInfo(in.AccessInfo)
	target := p.getTargetFromRequest(in.Target)
	logrus.Debugf("Trying to get target %s", target)
	t, err := targets.GetTarget(target)
	if err != nil {
		return retval, err
	}
	vm, err := t.GetVM(ctx, *in.AccessInfo, in.GetName())
	if err != nil {
		return retval, err
	}
	bd, ok := api.BootDevice_value[vm.BootDevice]
	if !ok {
		bd = int32(api.BootDevice_BOOT_DEVICE_UNKNOWN)
	}
	retval.Vm = &api.VMInfo{
		Name:        vm.Name,
		Cpu:         vm.CPU,
		Memory:      vm.Memory,
		Ipv4Addr:    vm.IPv4Addr,
		Ipv6Addr:    vm.IPv6Addr,
		PowerStatus: vm.PowerStatus,
		BootDevice:  api.BootDevice(bd),
		GuestOs:     vm.GuestOS,
	}
	return retval, nil
}

func (p *targetVcenterGRPC) ReclaimVM(ctx context.Context, in *api.ReclaimVMRequest) (*api.ReclaimVMResponse, error) {
	retval := &api.ReclaimVMResponse{}
	//creds := p.populateCredsFromAccessInfo(in.AccessInfo)
	target := p.getTargetFromRequest(in.Target)
	logrus.Debugf("Trying to get target %s", target)
	t, err := targets.GetTarget(target)
	if err != nil {
		return retval, err
	}
	err = t.ReclaimVM(ctx, *in.AccessInfo, in.GetName(), in.GetArgs()...)
	if err != nil {
		return retval, err
	}
	return retval, nil
}

func (p *targetVcenterGRPC) CordonHost(ctx context.Context, in *api.CordonHostRequest) (*api.CordonHostResponse, error) {
	retval := &api.CordonHostResponse{}
	//creds := p.populateCredsFromAccessInfo(in.AccessInfo)
	target := p.getTargetFromRequest(in.Target)
	logrus.Debugf("Trying to get target %s", target)
	t, err := targets.GetTarget(target)
	if err != nil {
		return retval, err
	}
	err = t.CordonHost(ctx, *in.AccessInfo, in.GetEsxiName())
	if err != nil {
		return retval, err
	}
	return retval, nil
}

func (p *targetVcenterGRPC) UnCordonHost(ctx context.Context, in *api.UnCordonHostRequest) (*api.UnCordonHostResponse, error) {
	retval := &api.UnCordonHostResponse{}
	//creds := p.populateCredsFromAccessInfo(in.AccessInfo)
	target := p.getTargetFromRequest(in.Target)
	logrus.Debugf("Trying to get target %s", target)
	t, err := targets.GetTarget(target)
	if err != nil {
		return retval, err
	}
	err = t.UnCordonHost(ctx, *in.AccessInfo, in.GetEsxiName())
	if err != nil {
		return retval, err
	}
	return retval, nil
}

func (p *targetVcenterGRPC) ListHosts(ctx context.Context, in *api.ListHostsRequest) (*api.ListHostsResponse, error) {
	target := p.getTargetFromRequest(in.Target)
	logrus.Debugf("Trying to get target %s", target)
	t, err := targets.GetTarget(target)
	if err != nil {
		return nil, err
	}
	hosts, err := t.ListHosts(ctx, *in.AccessInfo)
	if err != nil {
		return nil, err
	}
	return hosts, nil
}
