// Copyright © 2024 The vjailbreak authors

package vm

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/pkg/errors"
	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	"github.com/platform9/vjailbreak/v2v-helper/pkg/constants"
	"github.com/platform9/vjailbreak/v2v-helper/pkg/k8sutils"
	"github.com/platform9/vjailbreak/v2v-helper/vcenter"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -source=vmops.go -destination=vmops_mock.go -package=vm

type VMOperations interface {
	GetVMInfo(ostype string, rdmDisks []string) (VMInfo, error)
	GetVMObj() *object.VirtualMachine
	UpdateDiskInfo(*VMInfo, VMDisk, bool) error
	UpdateDisksInfo(*VMInfo) error
	IsCBTEnabled() (bool, error)
	EnableCBT() error
	TakeSnapshot(name string) error
	DeleteSnapshot(name string) error
	DeleteSnapshotByRef(snap *types.ManagedObjectReference) error
	GetSnapshot(name string) (*types.ManagedObjectReference, error)
	ListSnapshots() ([]types.VirtualMachineSnapshotTree, error)
	CleanUpSnapshots(ignoreerror bool) error
	DeleteMigrationSnapshots(snapshots []types.VirtualMachineSnapshotTree, ignoreerror bool) error
	CustomQueryChangedDiskAreas(baseChangeID string, curSnapshot *types.ManagedObjectReference, disk *types.VirtualDisk, offset int64) (types.DiskChangeInfo, error)
	VMGuestShutdown() error
	VMPowerOff() error
	VMPowerOn() error
	DisconnectNetworkInterfaces() error
}

type IpEntry struct {
	IP     string
	Prefix int32
}

type VMInfo struct {
	CPU               int32
	Memory            int32
	State             types.VirtualMachinePowerState
	Mac               []string
	IPperMac          map[string][]IpEntry
	UUID              string
	Host              string
	VMDisks           []VMDisk
	UEFI              bool
	Name              string
	OSType            string
	GuestNetworks     []vjailbreakv1alpha1.GuestNetwork
	NetworkInterfaces []vjailbreakv1alpha1.NIC
	RDMDisks          []vjailbreakv1alpha1.RDMDisk
	GatewayIP         map[string]string
}

type NIC struct {
	Network string
	MAC     string
	Index   int
}

type GuestNetwork struct {
	MAC          string
	IP           string
	Origin       string
	PrefixLength int32
	DNS          []string
	Device       string
}

type ChangeID struct {
	UUID   string
	Number string
	Value  string
}

type VMDisk struct {
	Name            string
	Size            int64
	OpenstackVol    *volumes.Volume
	Path            string
	Disk            *types.VirtualDisk
	Snapname        string
	SnapBackingDisk string
	ChangeID        string
	Boot            bool
	Datastore       string
	DatastoreID     string
}

type VMOps struct {
	vcclient  *vcenter.VCenterClient
	VMObj     *object.VirtualMachine
	ctx       context.Context
	k8sClient k8sclient.Client
}

func VMOpsBuilder(ctx context.Context, vcclient vcenter.VCenterClient, name string, k8sClient k8sclient.Client) (*VMOps, error) {
	vm, err := vcclient.GetVMByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM: %s", err)
	}
	return &VMOps{vcclient: &vcclient, VMObj: vm, ctx: ctx, k8sClient: k8sClient}, nil

}
func (vmops *VMOps) GetVmPowerState() (types.VirtualMachinePowerState, error) {
	return vmops.VMObj.PowerState(vmops.ctx)
}
func (vmops *VMOps) GetVMObj() *object.VirtualMachine {
	return vmops.VMObj
}

func (vmops *VMOps) GetVCenterClient() *vcenter.VCenterClient {
	return vmops.vcclient
}

func (vmops *VMOps) RefreshVM() error {
	vmobj, err := vmops.vcclient.GetVMByName(vmops.ctx, vmops.VMObj.Name())
	if err != nil {
		return fmt.Errorf("failed to refresh VM reference: %s", err)
	}
	vmops.VMObj = vmobj
	return nil
}

func (vmops *VMOps) GetVMInfo(ostype string, rdmDisks []string) (VMInfo, error) {
	vm := vmops.VMObj

	var o mo.VirtualMachine
	err := vm.Properties(vmops.ctx, vm.Reference(), []string{}, &o)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return VMInfo{}, fmt.Errorf("failed to get VM properties: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return VMInfo{}, fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		err = vm.Properties(vmops.ctx, vm.Reference(), []string{}, &o)
		if err != nil {
			return VMInfo{}, fmt.Errorf("failed to get VM properties: %s", err)
		}
	}
	var mac []string
	for _, device := range o.Config.Hardware.Device {
		if nic, ok := device.(types.BaseVirtualEthernetCard); ok {
			mac = append(mac, nic.GetVirtualEthernetCard().MacAddress)
		}
	}
	// Get IP addresses of the VM from vmwaremachines
	ips := []string{}
	ipPerMac := make(map[string][]IpEntry)

	// Get the vmware machine from k8s
	vmk8sName, err := k8sutils.GetVMwareMachineName()
	if err != nil {
		return VMInfo{}, fmt.Errorf("failed to get vmware machine name: %w", err)
	}

	vmwareMachine, err := k8sutils.GetVMwareMachine(vmops.ctx, vmk8sName)
	if err != nil {
		return VMInfo{}, fmt.Errorf("failed to get vmware machine: %s", err)
	}

	for _, macAddresss := range mac {
		// Get the IPs from the vmware machine.
		if vmwareMachine.Spec.VMInfo.GuestNetworks != nil {
			for _, guestNetwork := range vmwareMachine.Spec.VMInfo.GuestNetworks {
				// Every mac should have a corresponding IP, Ignore link layer ip
				if strings.EqualFold(guestNetwork.MAC, macAddresss) {
					if _, ok := ipPerMac[guestNetwork.MAC]; !ok {
						ipPerMac[guestNetwork.MAC] = []IpEntry{}
					}
					if !strings.Contains(guestNetwork.IP, ":") {
						ips = append(ips, guestNetwork.IP)
						ipPerMac[guestNetwork.MAC] = append(ipPerMac[guestNetwork.MAC], IpEntry{
							IP:     guestNetwork.IP,
							Prefix: guestNetwork.PrefixLength,
						})
					}
				}
			}
		} else {
			if vmwareMachine.Spec.VMInfo.NetworkInterfaces != nil {
				for _, networkInterface := range vmwareMachine.Spec.VMInfo.NetworkInterfaces {
					if networkInterface.MAC == macAddresss {
						if _, ok := ipPerMac[networkInterface.MAC]; !ok {
							ipPerMac[networkInterface.MAC] = []IpEntry{}
						}
						if !strings.Contains(networkInterface.IPAddress, ":") {
							ips = append(ips, networkInterface.IPAddress)
							ipPerMac[networkInterface.MAC] = append(ipPerMac[networkInterface.MAC], IpEntry{
								IP:     networkInterface.IPAddress,
								Prefix: 0,
							})
						}
					}
				}
			}
			if len(ips) == 0 {
				return VMInfo{}, errors.New(`No IP address found for the VM, if VM is powered off, 
				please make sure to provide IP address in the vmwaremachine CR`)
			}
		}
	}

	vmdisks := []VMDisk{}
	for _, device := range o.Config.Hardware.Device {
		if disk, ok := device.(*types.VirtualDisk); ok {
			if _, ok := disk.Backing.(*types.VirtualDiskRawDiskMappingVer1BackingInfo); ok {
				continue
			}

			// Get datastore information and VMDK path from disk backing
			var datastoreName string
			var datastoreID string
			var vmdkPath string

			if backing, ok := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo); ok {
				vmdkPath = backing.FileName
				if backing.Datastore != nil {
					datastoreID = backing.Datastore.Value
					// Get datastore name
					ds := object.NewDatastore(vmops.vcclient.VCClient, *backing.Datastore)
					var dsObj mo.Datastore
					if err := ds.Properties(vmops.ctx, ds.Reference(), []string{"name"}, &dsObj); err == nil {
						datastoreName = dsObj.Name
					}
				}
			} else if backing, ok := disk.Backing.(*types.VirtualDiskSparseVer2BackingInfo); ok {
				vmdkPath = backing.FileName
				if backing.Datastore != nil {
					datastoreID = backing.Datastore.Value
					ds := object.NewDatastore(vmops.vcclient.VCClient, *backing.Datastore)
					var dsObj mo.Datastore
					if err := ds.Properties(vmops.ctx, ds.Reference(), []string{"name"}, &dsObj); err == nil {
						datastoreName = dsObj.Name
					}
				}
			}
			vmdisks = append(vmdisks, VMDisk{
				Name:        disk.DeviceInfo.GetDescription().Label,
				Size:        disk.CapacityInBytes,
				Disk:        disk,
				Path:        vmdkPath,
				Datastore:   datastoreName,
				DatastoreID: datastoreID,
			})
		}
	}
	uefi := false
	if o.Config.Firmware == "efi" {
		uefi = true
	}
	if ostype == "" {
		if strings.EqualFold(string(o.Guest.GuestFamily), string(types.VirtualMachineGuestOsFamilyWindowsGuest)) {
			ostype = constants.OSFamilyWindows
		} else if strings.EqualFold(string(o.Guest.GuestFamily), string(types.VirtualMachineGuestOsFamilyLinuxGuest)) {
			ostype = constants.OSFamilyLinux
		} else {
			return VMInfo{}, fmt.Errorf("no OS type provided and unable to determine OS type")
		}
	}
	rdmDiskSlice := make([]vjailbreakv1alpha1.RDMDisk, 0)
	// Get RDM disks from vmware machine
	for _, rdm := range rdmDisks {
		rdmDisk, err := k8sutils.GetRDMDisk(vmops.ctx, rdm)
		if err != nil {
			return VMInfo{}, fmt.Errorf("failed to get RDM disk: %w", err)
		}
		rdmDiskSlice = append(rdmDiskSlice, *rdmDisk)
	}
	vminfo := VMInfo{
		CPU:               o.Config.Hardware.NumCPU,
		Memory:            o.Config.Hardware.MemoryMB,
		State:             o.Runtime.PowerState,
		Mac:               mac,
		IPperMac:          ipPerMac,
		UUID:              o.Config.Uuid,
		Host:              o.Runtime.Host.Reference().Value,
		Name:              o.Name,
		VMDisks:           vmdisks,
		RDMDisks:          rdmDiskSlice,
		UEFI:              uefi,
		OSType:            ostype,
		NetworkInterfaces: vmwareMachine.Spec.VMInfo.NetworkInterfaces,
		GuestNetworks:     vmwareMachine.Spec.VMInfo.GuestNetworks,
		GatewayIP:         make(map[string]string),
	}
	return vminfo, nil
}

func parseChangeID(changeId string) (*ChangeID, error) {
	changeIdParts := strings.Split(changeId, "/")
	if len(changeIdParts) != 2 {
		return nil, fmt.Errorf("invalid change ID format")
	}

	return &ChangeID{
		UUID:   changeIdParts[0],
		Number: changeIdParts[1],
		Value:  changeId,
	}, nil
}

func getChangeID(disk *types.VirtualDisk) (*ChangeID, error) {
	var changeId string
	if b, ok := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo); ok {
		changeId = b.ChangeId
	} else if b, ok := disk.Backing.(*types.VirtualDiskSparseVer2BackingInfo); ok {
		changeId = b.ChangeId
	} else if b, ok := disk.Backing.(*types.VirtualDiskRawDiskMappingVer1BackingInfo); ok {
		changeId = b.ChangeId
	} else if b, ok := disk.Backing.(*types.VirtualDiskRawDiskVer2BackingInfo); ok {
		changeId = b.ChangeId
	} else {
		return nil, fmt.Errorf("failed to get change ID")
	}

	if changeId == "" {
		return nil, fmt.Errorf("CBT is not enabled on disk %d", disk.Key)
	}
	return parseChangeID(changeId)
}

func (vmops *VMOps) UpdateDisksInfo(vminfo *VMInfo) error {
	pc := vmops.vcclient.VCPropertyCollector
	var snapbackingdisk []string
	var snapname []string
	var snapid []string

	vm := vmops.VMObj

	var o mo.VirtualMachine
	err := vm.Properties(vmops.ctx, vm.Reference(), []string{}, &o)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return fmt.Errorf("failed to get VM properties: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		pc = property.DefaultCollector(vmops.vcclient.VCClient)
		err = vm.Properties(vmops.ctx, vm.Reference(), []string{}, &o)
		if err != nil {
			return fmt.Errorf("failed to get VM properties: %s", err)
		}
	}

	if o.Snapshot != nil {
		// get backing disk of snapshot
		var s mo.VirtualMachineSnapshot
		err := pc.RetrieveOne(vmops.ctx, o.Snapshot.CurrentSnapshot.Reference(), []string{}, &s)
		if err != nil {
			return fmt.Errorf("failed to get snapshot properties: %s", err)
		}

		// Build map of snapshot info indexed by device key
		// This ensures correct matching even if disk order differs between VM and snapshot
		snapshotInfo := make(map[int32]struct {
			FileName string
			ChangeID string
		})

		for _, device := range s.Config.Hardware.Device {
			switch disk := device.(type) {
			case *types.VirtualDisk:
				backing := disk.Backing.(types.BaseVirtualDeviceFileBackingInfo)
				info := backing.GetVirtualDeviceFileBackingInfo()
				changeid, err := getChangeID(disk)
				if err != nil {
					return fmt.Errorf("failed to get change ID for device key %d: %s", disk.Key, err)
				}
				snapshotInfo[disk.Key] = struct {
					FileName string
					ChangeID string
				}{
					FileName: info.FileName,
					ChangeID: changeid.Value,
				}
			}
		}

		// Match snapshot disks to VM disks by device key (not by index)
		// Fixes bug where vCenter returns disks in different order for VM vs snapshot
		for idx := range vminfo.VMDisks {
			deviceKey := vminfo.VMDisks[idx].Disk.Key

			// Fallback: if device key is invalid, use index-based matching with warning
			if deviceKey == 0 {
				log.Printf("WARNING: Disk %s has invalid device key, using index-based fallback", vminfo.VMDisks[idx].Name)
				if idx < len(snapbackingdisk) {
					vminfo.VMDisks[idx].SnapBackingDisk = snapbackingdisk[idx]
					vminfo.VMDisks[idx].Snapname = snapname[idx]
					vminfo.VMDisks[idx].ChangeID = snapid[idx]
				}
				continue
			}

			snapInfo, found := snapshotInfo[deviceKey]
			if !found {
				return fmt.Errorf("snapshot not found for disk %s (device key=%d)", vminfo.VMDisks[idx].Name, deviceKey)
			}

			vminfo.VMDisks[idx].SnapBackingDisk = snapInfo.FileName
			vminfo.VMDisks[idx].Snapname = o.Snapshot.CurrentSnapshot.Value
			vminfo.VMDisks[idx].ChangeID = snapInfo.ChangeID
		}
	}

	return nil
}

func (vmops *VMOps) UpdateDiskInfo(vminfo *VMInfo, disk VMDisk, blockCopySuccess bool) error {
	pc := vmops.vcclient.VCPropertyCollector
	var snapbackingdisk []string
	var snapname []string
	var snapid []string

	vm := vmops.VMObj

	var o mo.VirtualMachine
	err := vm.Properties(vmops.ctx, vm.Reference(), []string{}, &o)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return fmt.Errorf("failed to get VM properties: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		pc = property.DefaultCollector(vmops.vcclient.VCClient)
		err = vm.Properties(vmops.ctx, vm.Reference(), []string{}, &o)
		if err != nil {
			return fmt.Errorf("failed to get VM properties: %s", err)
		}
	}

	if o.Snapshot != nil {
		// get backing disk of snapshot
		var s mo.VirtualMachineSnapshot
		err := pc.RetrieveOne(vmops.ctx, o.Snapshot.CurrentSnapshot.Reference(), []string{}, &s)
		if err != nil {
			return fmt.Errorf("failed to get snapshot properties: %s", err)
		}

		for _, device := range s.Config.Hardware.Device {
			switch disk := device.(type) {
			case *types.VirtualDisk:
				backing := disk.Backing.(types.BaseVirtualDeviceFileBackingInfo)
				info := backing.GetVirtualDeviceFileBackingInfo()
				snapbackingdisk = append(snapbackingdisk, info.FileName)
				snapname = append(snapname, o.Snapshot.CurrentSnapshot.Value)
				changeid, err := getChangeID(disk)
				if err != nil {
					return fmt.Errorf("failed to get change ID: %s", err)
				}
				snapid = append(snapid, changeid.Value)
			}
		}
		for idx, _ := range vminfo.VMDisks {
			if vminfo.VMDisks[idx].Name == disk.Name {
				if blockCopySuccess {
					vminfo.VMDisks[idx].ChangeID = snapid[idx]
					log.Printf("Updating Change ID: %s", snapid[idx])
				}
				vminfo.VMDisks[idx].SnapBackingDisk = snapbackingdisk[idx]
				vminfo.VMDisks[idx].Snapname = snapname[idx]
				log.Printf("Updated disk info for %s", disk.Name)
				log.Printf("Snapshot backing disk: %s", snapbackingdisk[idx])
				log.Printf("Snapshot name: %s", snapname[idx])

				break
			}
		}
	}

	return nil
}

func (vmops *VMOps) IsCBTEnabled() (bool, error) {
	vm := vmops.VMObj
	var o mo.VirtualMachine
	err := vm.Properties(vmops.ctx, vm.Reference(), []string{"config.changeTrackingEnabled"}, &o)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return false, fmt.Errorf("failed to get VM properties: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return false, fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		err = vm.Properties(vmops.ctx, vm.Reference(), []string{"config.changeTrackingEnabled"}, &o)
		if err != nil {
			return false, fmt.Errorf("failed to get VM properties: %s", err)
		}
	}
	return *o.Config.ChangeTrackingEnabled, nil
}

func (vmops *VMOps) EnableCBT() error {
	vm := vmops.VMObj

	configSpec := types.VirtualMachineConfigSpec{
		ChangeTrackingEnabled: types.NewBool(true),
	}

	task, err := vm.Reconfigure(vmops.ctx, configSpec)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return fmt.Errorf("failed to enable CBT: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		task, err = vm.Reconfigure(vmops.ctx, configSpec)
		if err != nil {
			return fmt.Errorf("failed to enable CBT: %s", err)
		}
	}
	err = task.Wait(vmops.ctx)
	if err != nil {
		return fmt.Errorf("failed while waiting for task: %s", err)
	}
	return nil
}

func (vmops *VMOps) TakeSnapshot(name string) error {
	vm := vmops.VMObj

	task, err := vm.CreateSnapshot(vmops.ctx, name, "", false, false)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return fmt.Errorf("failed to take snapshot: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		task, err = vm.CreateSnapshot(vmops.ctx, name, "", false, false)
		if err != nil {
			return fmt.Errorf("failed to take snapshot: %s", err)
		}
	}

	err = task.Wait(vmops.ctx)
	if err != nil {
		return fmt.Errorf("failed while waiting for task: %s", err)
	}
	return nil
}

func (vmops *VMOps) DeleteSnapshot(name string) error {
	vm := vmops.VMObj

	var consolidate = true
	task, err := vm.RemoveSnapshot(vmops.ctx, name, false, &consolidate)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return fmt.Errorf("failed to delete snapshot: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		task, err = vm.RemoveSnapshot(vmops.ctx, name, false, &consolidate)
		if err != nil {
			return fmt.Errorf("failed to delete snapshot: %s", err)
		}
	}
	err = task.Wait(vmops.ctx)
	if err != nil {
		return fmt.Errorf("failed while waiting for task: %s", err)
	}
	return nil
}

func (vmops *VMOps) DeleteSnapshotByRef(snap *types.ManagedObjectReference) error {
	// Create a method to remove snapshot using the reference
	var consolidate = true

	// We need to use the session-aware client for this operation
	// Get a reference to the snapshot object using the session-aware client
	snap_obj := types.ManagedObjectReference{
		Type:  snap.Type,
		Value: snap.Value,
	}

	// Create a RemoveSnapshot_Task request
	req := types.RemoveSnapshot_Task{
		This:           snap_obj,
		RemoveChildren: true,
		Consolidate:    &consolidate,
	}

	// Send the request using the session-aware client
	res, err := methods.RemoveSnapshot_Task(vmops.ctx, vmops.vcclient.VCClient, &req)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return fmt.Errorf("failed to remove snapshot by ref: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		res, err = methods.RemoveSnapshot_Task(vmops.ctx, vmops.vcclient.VCClient, &req)
		if err != nil {
			return fmt.Errorf("failed to remove snapshot by ref: %s", err)
		}
	}

	// Create and monitor the task using the session-aware client
	task := object.NewTask(vmops.vcclient.VCClient, res.Returnval)
	err = task.Wait(vmops.ctx)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return fmt.Errorf("failed while waiting for task: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		task = object.NewTask(vmops.vcclient.VCClient, res.Returnval)
		err = task.Wait(vmops.ctx)
		if err != nil {
			return fmt.Errorf("failed while waiting for task: %s", err)
		}
	}
	return nil
}

func (vmops *VMOps) GetSnapshot(name string) (*types.ManagedObjectReference, error) {
	vm := vmops.VMObj

	snap, err := vm.FindSnapshot(vmops.ctx, name)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return nil, fmt.Errorf("failed to find snapshot: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return nil, fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		snap, err = vm.FindSnapshot(vmops.ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to find snapshot: %s", err)
		}
	}
	return snap, nil
}

func (vmops *VMOps) CustomQueryChangedDiskAreas(baseChangeID string, curSnapshot *types.ManagedObjectReference, disk *types.VirtualDisk, offset int64) (types.DiskChangeInfo, error) {
	var changedblocks types.DiskChangeInfo

	vm := vmops.VMObj

	req := types.QueryChangedDiskAreas{
		This:        vm.Reference(),
		ChangeId:    baseChangeID,
		DeviceKey:   disk.Key,
		StartOffset: offset,
	}

	if curSnapshot != nil {
		req.Snapshot = curSnapshot
	}
	for {
		res, err := methods.QueryChangedDiskAreas(vmops.ctx, vmops.vcclient.VCClient, &req)
		if err != nil {
			if !strings.Contains(err.Error(), "NotAuthenticated") {
				return changedblocks, fmt.Errorf("failed to query changed disk areas: %s", err)
			}
			if err := vmops.RefreshVM(); err != nil {
				return changedblocks, fmt.Errorf("failed to refresh VM reference: %s", err)
			}
			res, err = methods.QueryChangedDiskAreas(vmops.ctx, vmops.vcclient.VCClient, &req)
			if err != nil {
				return changedblocks, fmt.Errorf("failed to query changed disk areas: %s", err)
			}
		}
		// If there are no more changes, stop fetching the changed blocks
		if len(res.Returnval.ChangedArea) == 0 {
			break
		}
		// Append the changed blocks to the result
		changedblocks.ChangedArea = append(changedblocks.ChangedArea, res.Returnval.ChangedArea...)
		// Update the total length of the changed blocks
		changedblocks.Length += res.Returnval.Length
		// If the total length of the changed blocks is greater or equal to the disk capacity, break the loop
		if changedblocks.Length >= disk.CapacityInBytes {
			break
		}
		// Update the start offset for the next iteration
		req.StartOffset = changedblocks.Length
	}

	return changedblocks, nil
}

func (vmops *VMOps) VMGuestShutdown() error {
	vm := vmops.VMObj

	currstate, err := vm.PowerState(vmops.ctx)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return fmt.Errorf("failed to get VM power state: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		currstate, err = vm.PowerState(vmops.ctx)
		if err != nil {
			return fmt.Errorf("failed to get VM power state: %s", err)
		}
	}

	if currstate == types.VirtualMachinePowerStatePoweredOff {
		return nil
	}

	err = vm.ShutdownGuest(vmops.ctx)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return fmt.Errorf("failed to initiate guest shutdown: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		err = vm.ShutdownGuest(vmops.ctx)
		if err != nil {
			return fmt.Errorf("failed to initiate guest shutdown: %s", err)
		}
	}

	// Wait for up to 5 minutes for the VM to power off
	poweredOff := false
	ctx, cancel := context.WithTimeout(vmops.ctx, 5*time.Minute)
	defer cancel()

	for !poweredOff {
		state, err := vm.PowerState(ctx)
		if err != nil {
			return fmt.Errorf("failed to get VM power state: %s", err)
		}
		if state == types.VirtualMachinePowerStatePoweredOff {
			poweredOff = true
			break
		}

		// Check if timeout occurred
		select {
		case <-ctx.Done():
			return fmt.Errorf("guest shutdown timed out after 2 minutes")
		default:
			time.Sleep(5 * time.Second)
		}
	}

	return nil
}

func (vmops *VMOps) VMPowerOff() error {
	vm := vmops.VMObj

	currstate, err := vm.PowerState(vmops.ctx)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return fmt.Errorf("failed to get VM power state: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		currstate, err = vm.PowerState(vmops.ctx)
		if err != nil {
			return fmt.Errorf("failed to get VM power state: %s", err)
		}
	}

	if currstate == types.VirtualMachinePowerStatePoweredOff {
		return nil
	}

	// First try a clean guest shutdown
	err = vmops.VMGuestShutdown()
	if err == nil {
		// Guest shutdown succeeded
		return nil
	}

	// If guest shutdown failed, log the error and fall back to power off
	fmt.Printf("Guest shutdown failed, falling back to power off: %s\n", err)

	// Fall back to power off - get a fresh VM reference again for the power operation
	vm, err = vmops.vcclient.GetVMByName(vmops.ctx, vmops.VMObj.Name())
	if err != nil {
		return fmt.Errorf("failed to refresh VM reference: %s", err)
	}

	task, err := vm.PowerOff(vmops.ctx)
	if err != nil {
		return err
	}
	err = task.Wait(vmops.ctx)
	if err != nil {
		return fmt.Errorf("failed while waiting for power off task: %s", err)
	}
	return nil
}

func (vmops *VMOps) VMPowerOn() error {
	vm := vmops.VMObj

	currstate, err := vm.PowerState(vmops.ctx)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return fmt.Errorf("failed to get VM power state: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		currstate, err = vm.PowerState(vmops.ctx)
		if err != nil {
			return fmt.Errorf("failed to get VM power state: %s", err)
		}
	}

	if currstate == types.VirtualMachinePowerStatePoweredOn {
		return nil
	}

	task, err := vm.PowerOn(vmops.ctx)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return fmt.Errorf("failed to power on VM: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		task, err = vm.PowerOn(vmops.ctx)
		if err != nil {
			return fmt.Errorf("failed to power on VM: %s", err)
		}
	}

	if err := task.Wait(vmops.ctx); err != nil {
		return fmt.Errorf("failed to wait for power on task: %w", err)
	}
	return nil
}

func (vmops *VMOps) DisconnectNetworkInterfaces() error {
	ctx := vmops.ctx

	vm := vmops.VMObj

	var mvm mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), []string{"config.hardware"}, &mvm); err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return fmt.Errorf("failed to get VM properties: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		if err := vm.Properties(ctx, vm.Reference(), []string{"config.hardware"}, &mvm); err != nil {
			return fmt.Errorf("failed to get VM properties: %s", err)
		}
	}

	if mvm.Config == nil || len(mvm.Config.Hardware.Device) == 0 {
		return nil
	}
	var deviceChanges []types.BaseVirtualDeviceConfigSpec

	for _, device := range mvm.Config.Hardware.Device {
		if nic, ok := device.(types.BaseVirtualEthernetCard); ok {
			nicName := nic.GetVirtualEthernetCard().DeviceInfo.GetDescription().Label
			log.Printf("Found NIC to disconnect: %s", nicName)
			deviceCopy := device
			connectable := nic.GetVirtualEthernetCard().Connectable
			connectable.Connected = false
			connectable.StartConnected = false
			spec := &types.VirtualDeviceConfigSpec{
				Operation: types.VirtualDeviceConfigSpecOperationEdit,
				Device:    deviceCopy,
			}
			deviceChanges = append(deviceChanges, spec)
		}
	}

	if len(deviceChanges) == 0 {
		return nil
	}
	spec := &types.VirtualMachineConfigSpec{
		DeviceChange: deviceChanges,
	}

	task, err := vm.Reconfigure(ctx, *spec)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return fmt.Errorf("failed to reconfigure VM network interfaces: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		task, err = vm.Reconfigure(ctx, *spec)
		if err != nil {
			return fmt.Errorf("failed to reconfigure VM network interfaces: %s", err)
		}
	}

	if err := task.Wait(ctx); err != nil {
		return errors.Wrap(err, "failed to wait for VM reconfiguration")
	}

	return nil
}

func (vmops *VMOps) ListSnapshots() ([]types.VirtualMachineSnapshotTree, error) {
	vm := vmops.VMObj

	var o mo.VirtualMachine
	err := vm.Properties(vmops.ctx, vm.Reference(), []string{"snapshot"}, &o)
	if err != nil {
		if !strings.Contains(err.Error(), "NotAuthenticated") {
			return nil, fmt.Errorf("failed to get VM properties: %s", err)
		}
		if err := vmops.RefreshVM(); err != nil {
			return nil, fmt.Errorf("failed to refresh VM reference: %s", err)
		}
		vm = vmops.VMObj
		if err := vm.Properties(vmops.ctx, vm.Reference(), []string{"snapshot"}, &o); err != nil {
			return nil, fmt.Errorf("failed to get VM properties: %s", err)
		}
	}

	// Check if o.Snapshot is nil before accessing RootSnapshotList
	if o.Snapshot == nil {
		// VM has no snapshots
		return []types.VirtualMachineSnapshotTree{}, nil
	}
	return o.Snapshot.RootSnapshotList, nil
}

func (vmops *VMOps) DeleteMigrationSnapshots(snapshots []types.VirtualMachineSnapshotTree, ignoreerror bool) error {
	var lastError error
	snapshotsDeleted := 0

	// Helper function to recursively process snapshot tree
	var processSnapshotTree func(trees []types.VirtualMachineSnapshotTree) int
	processSnapshotTree = func(trees []types.VirtualMachineSnapshotTree) int {
		deleted := 0
		for _, snapshot := range trees {
			// First process any child snapshots
			if len(snapshot.ChildSnapshotList) > 0 {
				deleted += processSnapshotTree(snapshot.ChildSnapshotList)
			}

			// Then process this snapshot if it matches
			if snapshot.Name == constants.MigrationSnapshotName {
				// Delete snapshot by snapshot reference instead of name to handle duplicate names
				err := vmops.DeleteSnapshotByRef(&snapshot.Snapshot)
				if err != nil {
					if ignoreerror {
						lastError = err
						log.Printf("Failed to delete snapshot %s: %v (ignoring error)", snapshot.Name, err)
						continue
					} else {
						// Propagate error up
						lastError = fmt.Errorf("failed to delete snapshot %s: %v", snapshot.Name, err)
						return deleted
					}
				}
				deleted++
			}
		}
		return deleted
	}

	// Start the recursive processing
	snapshotsDeleted = processSnapshotTree(snapshots)

	if snapshotsDeleted > 0 {
		log.Printf("Successfully deleted %d snapshots with name '%s'", snapshotsDeleted, constants.MigrationSnapshotName)
	}

	if !ignoreerror && lastError != nil {
		return lastError
	}

	return nil
}

func (vmops *VMOps) CleanUpSnapshots(ignoreerror bool) error {
	// ListSnapshots already uses the session-aware client to get a fresh VM reference
	snapshotlist, err := vmops.ListSnapshots()
	if err != nil {
		return err
	}
	return vmops.DeleteMigrationSnapshots(snapshotlist, ignoreerror)
}
