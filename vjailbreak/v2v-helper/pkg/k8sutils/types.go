package k8sutils

type VjailbreakSettings struct {
	ChangedBlocksCopyIterationThreshold int
	PeriodicSyncInterval                string
	VMActiveWaitIntervalSeconds         int
	VMActiveWaitRetryLimit              int
	DefaultMigrationMethod              string
	VCenterScanConcurrencyLimit         int
	CleanupVolumesAfterConvertFailure   bool
	CleanupPortsAfterMigrationFailure   bool
	PopulateVMwareMachineFlavors        bool
	VolumeAvailableWaitIntervalSeconds  int
	VolumeAvailableWaitRetryLimit       int
	VCenterLoginRetryLimit              int
	OpenstackCredsRequeueAfterMinutes   int
	VMwareCredsRequeueAfterMinutes      int
	ValidateRDMOwnerVMs                 bool
	PeriodicSyncMaxRetries              uint64
	PeriodicSyncRetryCap                string
	AutoFstabUpdate                     bool
	AutoPXEBootOnConversion             bool
	// V2VHelperPod resource configuration
	V2VHelperPodCPURequest              string
	V2VHelperPodMemoryRequest           string
	V2VHelperPodCPULimit                string
	V2VHelperPodMemoryLimit             string
	V2VHelperPodEphemeralStorageRequest string
	V2VHelperPodEphemeralStorageLimit   string
}
