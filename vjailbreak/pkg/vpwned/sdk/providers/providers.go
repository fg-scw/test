package providers

import (
	"context"
	"errors"
	"strings"

	api "github.com/platform9/vjailbreak/pkg/vpwned/api/proto/v1/service"
)

var providers map[string]BMCProvider = make(map[string]BMCProvider)

type BMCProvider interface {
	//BM provisoner functions
	//a function to connect to the underlying provider
	Connect(accessInfo BMAccessInfo) error
	//a function to disconnect from the underlying provider
	Disconnect() error
	// a function to check if the BM is in ready state
	// this state could be a combination of different states in the
	// underlying BM provisioner
	IsBMReady(ctx context.Context, req api.IsBMReadyRequest) (api.IsBMReadyResponse, error)
	// a function to check if the BM is in running state
	IsBMRunning(ctx context.Context, req api.IsBMRunningRequest) (api.IsBMRunningResponse, error)
	//Power Functions
	//start the BM
	StartBM(ctx context.Context, req api.StartBMRequest) (api.StartBMResponse, error)
	//Stop the BM
	StopBM(ctx context.Context, req api.StopBMRequest) (api.StopBMResponse, error)
	//Deploy Machine
	DeployMachine(ctx context.Context, req api.DeployMachineRequest) (api.DeployMachineResponse, error)
	//Set to PXE boot on next reboot
	SetBM2PXEBoot(ctx context.Context, resourceID string, power_cycle bool, ipmi_interface *api.IpmiType) error
	//Reclaim functions
	//Reclaim the VM such that we erase and resuse the VM as a PCD Host
	ReclaimBM(ctx context.Context, req api.ReclaimBMRequest) error
	// Identify the provider
	WhoAmI() string
	// List resources
	ListResources(ctx context.Context) ([]api.MachineInfo, error)
	// Set resource power
	SetResourcePower(ctx context.Context, resourceID string, action api.PowerStatus) error
	// Get Resource status
	GetResourceInfo(ctx context.Context, resourceID string) (api.MachineInfo, error)
	// List Boot Sources
	ListBootSource(ctx context.Context, req api.ListBootSourceRequest) ([]api.BootsourceSelections, error)
	// Get IPMI client
	GetIPMIClient(ctx context.Context, host, username, password string, ipmi_interface *api.IpmiType) (*api.IpmiType, error)
}

type BMAccessInfo struct {
	Username    string
	Password    string
	APIKey      string
	BaseURL     string
	UseInsecure bool
	Provider    string
}

func RegisterProvider(name string, provider BMCProvider) {
	providers[strings.ToLower(name)] = provider
}

func DeleteProvider(name string) {
	delete(providers, strings.ToLower(name))
}

func GetProviders() []string {
	var names []string
	for name := range providers {
		names = append(names, name)
	}
	return names
}

func GetProvider(name string) (BMCProvider, error) {
	provider, ok := providers[strings.ToLower(name)]
	if !ok {
		return nil, errors.New("provider not found")
	}
	return provider, nil
}
