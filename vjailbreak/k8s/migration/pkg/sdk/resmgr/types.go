package resmgr

import (
	"encoding/json"
	"net/http"

	"github.com/platform9/vjailbreak/k8s/migration/pkg/sdk/keystone"
	pcd "github.com/platform9/vjailbreak/k8s/migration/pkg/sdk/pcd"
)

// Extensions represents extended host information including network interfaces and their configuration.
type Extensions struct {
	Interfaces InterfaceInfo `json:"interfaces"`
}

// InterfaceInfo contains the status and data of host network interfaces.
type InterfaceInfo struct {
	Status string           `json:"status"`
	Data   InterfaceDetails `json:"data"`
}

// InterfaceDetails contains detailed information about network interfaces,
// including IP address mappings for each interface.
type InterfaceDetails struct {
	InterfaceIP map[string]string `json:"iface_ip"`
}

// NICDetails contains details about a network interface card,
// including its MAC address and configured interfaces.
type NICDetails struct {
	MAC    string             `json:"mac"`
	Ifaces []InterfaceAddress `json:"ifaces"`
}

// InterfaceAddress contains IP addressing information for a network interface,
// including the IP address, netmask, and broadcast address.
type InterfaceAddress struct {
	Addr      string `json:"addr"`
	Netmask   string `json:"netmask"`
	Broadcast string `json:"broadcast"`
}

// Host represents a physical or virtual host in the Platform9 environment,
// containing detailed information about the system, its hardware, and configuration status.
type Host struct {
	ID   string `json:"id"`
	Info struct {
		Hostname         string      `json:"hostname"`
		OSFamily         string      `json:"os_family"`
		Arch             string      `json:"arch"`
		OSInfo           string      `json:"os_info"`
		Responding       bool        `json:"responding"`
		LastResponseTime interface{} `json:"last_response_time"`
		CPUInfo          json.RawMessage `json:"cpu_info,omitempty"`
	} `json:"info,omitempty"`
	Roles              []string               `json:"roles,omitempty"`
	RolesStatusDetails map[string]string      `json:"roles_status_details,omitempty"`
	RoleStatus         string                 `json:"role_status,omitempty"`
	RoleSettings       map[string]interface{} `json:"role_settings,omitempty"`
	HypervisorInfo     json.RawMessage `json:"hypervisor_info,omitempty"`
	RawExtensionData json.RawMessage   `json:"-"`
	CAPIExtension    PF9CAPIExtensions `json:"-"`
	Extensions       json.RawMessage `json:"extensions,omitempty"`
	Message      interface{} `json:"message,omitempty"`
	HostconfigID string      `json:"hostconfig_id,omitempty"`
}

// Cluster represents a group of hosts managed as a single entity in Platform9,
// including configuration for high availability and resource balancing capabilities.
type Cluster struct {
	Name               string `json:"name"`
	Description        string `json:"description"`
	VMHighAvailability struct {
		Enabled bool `json:"enabled"`
	} `json:"vmHighAvailability"`
	AutoResourceRebalancing struct {
		Enabled                  bool `json:"enabled"`
		RebalancingFrequencyMins int  `json:"rebalancingFrequencyMins"`
	} `json:"autoResourceRebalancing"`
	Hostlist    []string `json:"hostlist"`
	AggregateID int      `json:"aggregate_id"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// Impl is the implementation of the Resmgr interface that provides
// methods for interacting with the Platform9 Resource Manager service.
type Impl struct {
	url           string
	authenticator keystone.Authenticator
	httpClient    http.Client
	insecure      bool
}

// Config contains the configuration settings needed to initialize a Resource Manager client,
// including connection information and authentication details.
type Config struct {
	DU            pcd.Info
	Authenticator keystone.Authenticator
	HTTPClient    http.Client
}

// RoleResponse represents the response structure returned when querying host roles,
// containing the list of roles assigned to a host.
type RoleResponse struct {
	Roles []string `json:"roles"`
}

// PF9CAPIExtensions contains extended information about Platform9 Cluster API status
// and management state for a host or cluster.
type PF9CAPIExtensions struct {
	CapiManaged struct {
		Status string `json:"status"`
		Data   struct {
			Managed bool `json:"managed"`
		} `json:"data,omitempty"`
	} `json:"pf9_capi,omitempty"`
}

// Type definition for payload to be sent to bundle generation request.
type bundle struct {
	Upload string `json:"upload"`
	Label  string `json:"label"`
}

type assignHypervisor struct {
	ClusterName string `json:"hostcluster"`
}

// ExtensionsData defines the need from the 'extensions' field of a resmgr.Host.
type ExtensionsData struct {
	Interfaces struct {
		Data struct {
			IfaceInfo map[string]struct {
				MAC    string `json:"mac"`
				Ifaces []struct {
					Addr string `json:"addr"`
				} `json:"ifaces"`
			} `json:"iface_info"`
		} `json:"data"`
	} `json:"interfaces"`
}
