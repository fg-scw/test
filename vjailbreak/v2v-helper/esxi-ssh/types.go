// Copyright © 2024 The vjailbreak authors

package esxissh

type ESXiCredentials struct {
	Host       string
	Port       int
	Username   string
	Password   string
	PrivateKey []byte
	SSHKeyPath string
}

type DiskInfo struct {
	Path          string // Full datastore path, e.g., /vmfs/volumes/datastore1/vm-name/disk.vmdk
	Name          string // Disk filename
	SizeBytes     int64  // Disk size in bytes
	ProvisionType string // thin, thick, etc.
	Datastore     string // Datastore name
}

type VMInfo struct {
	Name      string
	ID        string // VM ID from ESXi
	VMXPath   string // Path to .vmx file
	Path      string // VM directory path
	Disks     []DiskInfo
	Datastore string
}

type DatastoreInfo struct {
	Name      string
	Path      string // /vmfs/volumes/...
	UUID      string // Datastore UUID
	Type      string // VMFS, NFS, etc.
	Capacity  int64
	FreeSpace int64
}

type StorageDeviceInfo struct {
	DeviceID    string // e.g., naa.600508b1001c1234567890abcdef1234
	DisplayName string
	Size        int64  // Size in bytes
	DeviceType  string // Direct-Access, CD-ROM, etc.
	Vendor      string
	Model       string
	IsLocal     bool
	IsSSD       bool
	DevfsPath   string // /vmfs/devices/disks/naa.xxx
}
