package base

import (
	"context"
	"errors"

	api "github.com/platform9/vjailbreak/pkg/vpwned/api/proto/v1/service"
	"github.com/platform9/vjailbreak/pkg/vpwned/sdk/providers"
)

const (
	UnimplementedBaseProviderName = "base"
)

// UnimplementedBaseProvider is a base implementation of the BMCProvider interface
// that provides common functionality for all providers

type UnimplementedBaseProvider struct {
	providers.BMCProvider
}

func (p *UnimplementedBaseProvider) Connect(auth providers.BMAccessInfo) error {
	return errors.New("not implemented")
}

func (p *UnimplementedBaseProvider) Disconnect() error {
	return errors.New("not implemented")
}

func (p *UnimplementedBaseProvider) GetProviderBMStatus() (string, error) {
	return "", errors.New("not implemented")
}

func (p *UnimplementedBaseProvider) IsBMReady(ctx context.Context, req api.IsBMReadyRequest) (api.IsBMReadyResponse, error) {
	return api.IsBMReadyResponse{}, errors.New("not implemented")
}

func (p *UnimplementedBaseProvider) IsBMRunning(ctx context.Context, req api.IsBMRunningRequest) (api.IsBMRunningResponse, error) {
	return api.IsBMRunningResponse{}, errors.New("not implemented")
}

func (p *UnimplementedBaseProvider) StartBM(ctx context.Context, req api.StartBMRequest) (api.StartBMResponse, error) {
	return api.StartBMResponse{}, errors.New("not implemented")
}

func (p *UnimplementedBaseProvider) StopBM(ctx context.Context, req api.StopBMRequest) (api.StopBMResponse, error) {
	return api.StopBMResponse{}, errors.New("not implemented")
}

func (p *UnimplementedBaseProvider) SetBM2PXEBoot(ctx context.Context, resourceID string, power_cycle bool, ipmi_interface *api.IpmiType) error {
	return errors.New("not implemented")
}

func (p *UnimplementedBaseProvider) ReclaimBM(ctx context.Context, req api.ReclaimBMRequest) error {
	return errors.New("not implemented")
}

func (p *UnimplementedBaseProvider) WhoAmI() string {
	return UnimplementedBaseProviderName
}

func (p *UnimplementedBaseProvider) ListResources(ctx context.Context) ([]api.MachineInfo, error) {
	return nil, errors.New("not implemented")
}

func (p *UnimplementedBaseProvider) SetResourcePower(ctx context.Context, resourceID string, action api.PowerStatus) error {
	return errors.New("not implemented")
}

func (p *UnimplementedBaseProvider) GetResourceInfo(ctx context.Context, resourceID string) (api.MachineInfo, error) {
	return api.MachineInfo{}, errors.New("not implemented")
}

func (p *UnimplementedBaseProvider) ListBootSource(ctx context.Context, req api.ListBootSourceRequest) ([]api.BootsourceSelections, error) {
	return nil, errors.New("not implemented")
}

func (p *UnimplementedBaseProvider) DeployMachine(ctx context.Context, req api.DeployMachineRequest) (api.DeployMachineResponse, error) {
	return api.DeployMachineResponse{}, errors.New("not implemented")
}

func init() {
	providers.RegisterProvider(UnimplementedBaseProviderName, &UnimplementedBaseProvider{})
}
