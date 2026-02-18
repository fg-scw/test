// Copyright © 2024 The vjailbreak authors

package vcenter

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	commonutils "github.com/platform9/vjailbreak/pkg/common/utils"
	"github.com/platform9/vjailbreak/v2v-helper/pkg/k8sutils"
	"github.com/vmware/govmomi/cli/esx"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/session/cache"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/types"
)

//go:generate mockgen -source=../vcenter/vcenterops.go -destination=../vcenter/vcenterops_mock.go -package=vcenter

type VCenterOperations interface {
	getDatacenters(ctx context.Context) ([]*object.Datacenter, error)
	GetVMByName(ctx context.Context, name string) (*object.VirtualMachine, error)
	RunCommandOnEsxi(ctx context.Context, host object.HostSystem, command []string) ([]esx.Values, error)
	GetDataStores(ctx context.Context, dataCenter *object.Datacenter, datastore string) (*object.Datastore, error)
}

type VCenterClient struct {
	VCClient            *vim25.Client
	VCFinder            *find.Finder
	VCPropertyCollector *property.Collector
	Session             *cache.Session
}

func validateVCenter(ctx context.Context, username, password, host string, disableSSLVerification bool) (*vim25.Client, *cache.Session, error) {

	u, err := commonutils.NormalizeVCenterURL(host)
	if err != nil {
		return nil, nil, err
	}
	u.User = url.UserPassword(username, password)

	// Create a session with automatic re-authentication
	s := &cache.Session{
		URL:      u,
		Insecure: disableSSLVerification,
		Reauth:   true, // Enable automatic re-authentication
	}

	// Create the client
	c := new(vim25.Client)
	// Exponential retry logic
	client, err := k8sutils.GetInclusterClient()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get in-cluster client: %v", err)
	}
	migrationSettings, err := k8sutils.GetVjailbreakSettings(ctx, client)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get vjailbreak settings: %v", err)
	}
	maxRetries := migrationSettings.VCenterLoginRetryLimit
	baseDelay := 500 * time.Millisecond // Initial delay
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = s.Login(ctx, c, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to login: %v", err)
		}
		if attempt < maxRetries {
			delayNum := math.Pow(2, float64(attempt)) * 500
			baseDelay = time.Duration(delayNum) * time.Millisecond
			time.Sleep(baseDelay * time.Duration(1<<uint(attempt-1))) // Exponential backoff
		}
	}

	// Return both the client and the session for persistent re-authentication
	return c, s, nil
}

func VCenterClientBuilder(ctx context.Context, username, password, host string, disableSSLVerification bool) (*VCenterClient, error) {
	client, session, err := validateVCenter(ctx, username, password, host, disableSSLVerification)
	if err != nil {
		return nil, fmt.Errorf("failed to validate vCenter connection: %v", err)
	}
	finder := find.NewFinder(client, false)
	pc := property.DefaultCollector(client)
	return &VCenterClient{VCClient: client, VCFinder: finder, VCPropertyCollector: pc, Session: session}, nil
}

func GetThumbprint(host string) (string, error) {
	// Get the thumbprint of the vCenter server
	host = strings.TrimRight(host, "/")

	// Establish a TLS connection to the server
	conn, err := tls.Dial("tcp", host+":443", &tls.Config{
		InsecureSkipVerify: true, // Skip verification
	})
	if err != nil {
		return "", fmt.Errorf("failed to connect to vCenter: %v", err)
	}
	defer conn.Close()

	// Get the server's certificates
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return "", fmt.Errorf("no certificates found")
	}

	// Compute the SHA-1 thumbprint of the first certificate
	cert := certs[0]
	thumbprint := ""

	for idx, thumbyte := range sha1.Sum(cert.Raw) {
		thumbprint += hex.EncodeToString([]byte{thumbyte})
		if idx < len(sha1.Sum(cert.Raw))-1 {
			thumbprint += ":"
		}
	}

	// Return the thumbprint as a hexadecimal string
	return thumbprint, nil
}

// Get all datacenters with retry and explicit authentication
func (vcclient *VCenterClient) getDatacenters(ctx context.Context) ([]*object.Datacenter, error) {
	// Create a new finder with the current client each time to ensure we're using the most up-to-date client
	vcclient.VCFinder = find.NewFinder(vcclient.VCClient, false)

	// Try to get datacenters
	datacenters, err := vcclient.VCFinder.DatacenterList(ctx, "*")
	if err != nil {
		// If we encounter an authentication error, force an explicit re-login and retry once
		if strings.Contains(err.Error(), "NotAuthenticated") && vcclient.Session != nil {
			// Explicitly force re-login
			login := vcclient.Session.Login
			if err := login(ctx, vcclient.VCClient, nil); err != nil {
				return nil, fmt.Errorf("failed to re-login during datacenter refresh: %v", err)
			}

			// Create a new finder with the refreshed client
			vcclient.VCFinder = find.NewFinder(vcclient.VCClient, false)

			// Try again
			datacenters, err = vcclient.VCFinder.DatacenterList(ctx, "*")
			if err != nil {
				return nil, fmt.Errorf("failed to get datacenters after re-login: %v", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get datacenters: %v", err)
		}
	}

	return datacenters, nil
}

// get VM by name
func (vcclient *VCenterClient) GetVMByName(ctx context.Context, name string) (*object.VirtualMachine, error) {
	datacenters, err := vcclient.getDatacenters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get datacenters: %v", err)
	}
	for _, datacenter := range datacenters {
		vcclient.VCFinder.SetDatacenter(datacenter)
		vm, err := vcclient.VCFinder.VirtualMachine(ctx, name)
		if err == nil {
			return vm, nil
		}
	}
	return nil, fmt.Errorf("VM not found")
}

// RenameVM renames a VM in vCenter by appending a suffix to its name
func (vcclient *VCenterClient) RenameVM(ctx context.Context, vmName, newVMName string) error {
	// Find the VM
	vm, err := vcclient.GetVMByName(ctx, vmName)
	if err != nil {
		return fmt.Errorf("failed to find VM '%s': %v", vmName, err)
	}

	// Rename the VM
	spec := types.VirtualMachineConfigSpec{
		Name: newVMName,
	}
	task, err := vm.Reconfigure(ctx, spec)
	if err != nil {
		return fmt.Errorf("failed to reconfigure VM '%s' with new name '%s': %v", vmName, newVMName, err)
	}
	err = task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to rename VM '%s' to '%s': %v", vmName, newVMName, err)
	}

	return nil
}

// MoveVMFolder moves a VM to a specified folder in vCenter
func (vcclient *VCenterClient) MoveVMFolder(ctx context.Context, vmName, folderName string) error {
	// Find the VM
	vm, err := vcclient.GetVMByName(ctx, vmName)
	if err != nil {
		return fmt.Errorf("failed to find VM '%s': %v", vmName, err)
	}

	// Find the target folder
	folderRef, err := vcclient.VCFinder.Folder(ctx, folderName)
	if err != nil {
		return fmt.Errorf("failed to find folder '%s': %v", folderName, err)
	}

	// Move the VM to the folder
	task, err := folderRef.MoveInto(ctx, []types.ManagedObjectReference{vm.Reference()})
	if err != nil {
		return fmt.Errorf("failed to initiate move of VM '%s' to folder '%s': %v", vmName, folderName, err)
	}
	err = task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to move VM '%s' to folder '%s': %v", vmName, folderName, err)
	}

	return nil
}

// RunCommandOnEsxi runs a command on an ESXi host
func (vcclient *VCenterClient) RunCommandOnEsxi(ctx context.Context, host object.HostSystem, command []string) ([]esx.Values, error) {
	esxCliExec, err := esx.NewExecutor(ctx, vcclient.VCClient, host.Reference())
	if err != nil {
		return nil, fmt.Errorf("failed to create esxcli executor: %v", err)
	}

	response, err := esxCliExec.Run(ctx, command)
	if err != nil {
		fmt.Println("Failed to run command on ESXi host: ", err)
		if fault, ok := err.(*esx.Fault); ok {
			fmt.Println("ESXi CLI Fault: ", fault)
		}
		return nil, fmt.Errorf("failed to run command on ESXi host: %v", err)

	}

	for _, value := range response.Values {
		message, ok := value["message"]
		if ok {
			fmt.Println("ESXi CLI Message: ", message)
		}
		status, ok := value["status"]
		if ok && strings.Join(status, "") != "0" {
			fmt.Println("ESXi CLI Status: ", status)
			return nil, fmt.Errorf("failed to run command on ESXi host: %v", err)
		}
	}

	return response.Values, nil

}

// GetDataStore gives the datastore object for the name
func (vcclient *VCenterClient) GetDataStores(ctx context.Context, dataCenter *object.Datacenter, datastore string) (*object.Datastore, error) {
	datastoreRef, err := vcclient.VCFinder.Datastore(ctx, datastore)
	if err != nil {
		return nil, fmt.Errorf("failed to find datastore '%s': %v", datastore, err)
	}
	return datastoreRef, nil
}
