package targets

import (
	"context"
	"errors"
	"strings"

	api "github.com/platform9/vjailbreak/pkg/vpwned/api/proto/v1/service"
)

var targets map[string]Targets = make(map[string]Targets)

type VMInfo struct {
	Name        string
	CPU         int64
	Memory      int64
	IPv4Addr    string
	IPv6Addr    string
	PowerStatus api.PowerStatus
	BootDevice  string
	GuestOS     string
}

type AccessInfo struct {
	Username     string
	Password     string
	HostnameOrIP string
	Port         string
	UseInsecure  bool
	Datacenter   string
}

type Targets interface {
	GetVM(ctx context.Context, a api.TargetAccessInfo, name string) (VMInfo, error)
	ListVMs(ctx context.Context, a api.TargetAccessInfo) ([]VMInfo, error)
	ReclaimVM(ctx context.Context, a api.TargetAccessInfo, name string, args ...string) error
	CordonHost(ctx context.Context, a api.TargetAccessInfo, name string) error
	UnCordonHost(ctx context.Context, a api.TargetAccessInfo, name string) error
	ListHosts(ctx context.Context, a api.TargetAccessInfo) (*api.ListHostsResponse, error)
}

func RegisterTarget(name string, target Targets) {
	targets[strings.ToLower(name)] = target
}

func DeleteTarget(name string) {
	delete(targets, strings.ToLower(name))
}

func GetTargets() []string {
	var names []string
	for name := range targets {
		names = append(names, name)
	}
	return names
}

func GetTarget(name string) (Targets, error) {
	target, ok := targets[strings.ToLower(name)]
	if !ok {
		return nil, errors.New("target not found")
	}
	return target, nil
}
