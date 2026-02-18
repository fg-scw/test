package k8sutils

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	"github.com/platform9/vjailbreak/v2v-helper/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8stypes "k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetInclusterClient() (client.Client, error) {
	// Create a direct Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get in-cluster config")
	}
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(vjailbreakv1alpha1.AddToScheme(scheme))
	clientset, err := client.New(config, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get in-cluster config")
	}

	return clientset, err
}

func GetVMwareMachine(ctx context.Context, vmName string) (*vjailbreakv1alpha1.VMwareMachine, error) {
	client, err := GetInclusterClient()
	if err != nil {
		return nil, err
	}
	vmwareMachine := &vjailbreakv1alpha1.VMwareMachine{}
	vmK8sName, err := GetVMwareMachineName()
	if err != nil {
		return nil, err
	}
	err = client.Get(ctx, types.NamespacedName{
		Name:      vmK8sName,
		Namespace: constants.NamespaceMigrationSystem,
	}, vmwareMachine)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get vmware machine")
	}
	return vmwareMachine, nil
}

func GetVMwareMachineName() (string, error) {
	vmK8sName := os.Getenv("VMWARE_MACHINE_OBJECT_NAME")
	if vmK8sName == "" {
		return "", errors.New("VMWARE_MACHINE_OBJECT_NAME environment variable is not set")
	}
	return vmK8sName, nil
}

func GetRDMDisk(ctx context.Context, diskName string) (*vjailbreakv1alpha1.RDMDisk, error) {
	client, err := GetInclusterClient()
	if err != nil {
		return nil, err
	}
	rdmDisk := &vjailbreakv1alpha1.RDMDisk{}
	if err != nil {
		return nil, err
	}
	err = client.Get(ctx, types.NamespacedName{
		Name:      diskName,
		Namespace: constants.NamespaceMigrationSystem,
	}, rdmDisk)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get vmware machine")
	}
	return rdmDisk, nil
}

// atoi is a helper function to convert string to int with a default value of 0
func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func GetVjailbreakSettings(ctx context.Context, k8sClient client.Client) (*VjailbreakSettings, error) {
	vjailbreakSettingsCM := &corev1.ConfigMap{}
	if err := k8sClient.Get(ctx, k8stypes.NamespacedName{Name: constants.VjailbreakSettingsConfigMapName, Namespace: constants.NamespaceMigrationSystem}, vjailbreakSettingsCM); err != nil {
		return nil, errors.Wrap(err, "failed to get vjailbreak settings configmap")
	}

	if vjailbreakSettingsCM.Data == nil {
		return &VjailbreakSettings{
			ChangedBlocksCopyIterationThreshold: constants.ChangedBlocksCopyIterationThreshold,
			PeriodicSyncInterval:                constants.PeriodicSyncInterval,
			VMActiveWaitIntervalSeconds:         constants.VMActiveWaitIntervalSeconds,
			VMActiveWaitRetryLimit:              constants.VMActiveWaitRetryLimit,
			DefaultMigrationMethod:              constants.DefaultMigrationMethod,
			VCenterScanConcurrencyLimit:         constants.VCenterScanConcurrencyLimit,
			CleanupVolumesAfterConvertFailure:   constants.CleanupVolumesAfterConvertFailure,
			CleanupPortsAfterMigrationFailure:   constants.CleanupPortsAfterMigrationFailure,
			PopulateVMwareMachineFlavors:        constants.PopulateVMwareMachineFlavors,
			VolumeAvailableWaitIntervalSeconds:  constants.VolumeAvailableWaitIntervalSeconds,
			VolumeAvailableWaitRetryLimit:       constants.VolumeAvailableWaitRetryLimit,
			VCenterLoginRetryLimit:              constants.VCenterLoginRetryLimit,
			OpenstackCredsRequeueAfterMinutes:   constants.OpenstackCredsRequeueAfterMinutes,
			VMwareCredsRequeueAfterMinutes:      constants.VMwareCredsRequeueAfterMinutes,
			ValidateRDMOwnerVMs:                 constants.ValidateRDMOwnerVMs,
			PeriodicSyncMaxRetries:              constants.PeriodicSyncMaxRetries,
			PeriodicSyncRetryCap:                constants.PeriodicSyncRetryCap,
			AutoFstabUpdate:                     constants.AutoFstabUpdate,
			AutoPXEBootOnConversion:             constants.AutoPXEBootOnConversionDefault,
			V2VHelperPodCPURequest:              constants.V2VHelperPodCPURequest,
			V2VHelperPodMemoryRequest:           constants.V2VHelperPodMemoryRequest,
			V2VHelperPodCPULimit:                constants.V2VHelperPodCPULimit,
			V2VHelperPodMemoryLimit:             constants.V2VHelperPodMemoryLimit,
			V2VHelperPodEphemeralStorageRequest: constants.V2VHelperPodEphemeralStorageRequest,
			V2VHelperPodEphemeralStorageLimit:   constants.V2VHelperPodEphemeralStorageLimit,
		}, nil
	}

	if vjailbreakSettingsCM.Data["CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD"] == "" {
		vjailbreakSettingsCM.Data["CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD"] = strconv.Itoa(constants.ChangedBlocksCopyIterationThreshold)
	}
	if vjailbreakSettingsCM.Data["PERIODIC_SYNC_MAX_RETRIES"] == "" {
		vjailbreakSettingsCM.Data["PERIODIC_SYNC_MAX_RETRIES"] = strconv.Itoa(constants.PeriodicSyncMaxRetries)
	}
	if vjailbreakSettingsCM.Data["PERIODIC_SYNC_RETRY_CAP"] == "" {
		vjailbreakSettingsCM.Data["PERIODIC_SYNC_RETRY_CAP"] = constants.PeriodicSyncRetryCap
	}
	if vjailbreakSettingsCM.Data["VM_ACTIVE_WAIT_INTERVAL_SECONDS"] == "" {
		vjailbreakSettingsCM.Data["VM_ACTIVE_WAIT_INTERVAL_SECONDS"] = strconv.Itoa(constants.VMActiveWaitIntervalSeconds)
	}
	if vjailbreakSettingsCM.Data["PERIODIC_SYNC_INTERVAL"] == "" {
		vjailbreakSettingsCM.Data["PERIODIC_SYNC_INTERVAL"] = constants.PeriodicSyncInterval
	}
	if vjailbreakSettingsCM.Data["VM_ACTIVE_WAIT_RETRY_LIMIT"] == "" {
		vjailbreakSettingsCM.Data["VM_ACTIVE_WAIT_RETRY_LIMIT"] = strconv.Itoa(constants.VMActiveWaitRetryLimit)
	}

	if vjailbreakSettingsCM.Data["DEFAULT_MIGRATION_METHOD"] == "" {
		vjailbreakSettingsCM.Data["DEFAULT_MIGRATION_METHOD"] = constants.DefaultMigrationMethod
	}

	if vjailbreakSettingsCM.Data["VCENTER_SCAN_CONCURRENCY_LIMIT"] == "" {
		vjailbreakSettingsCM.Data["VCENTER_SCAN_CONCURRENCY_LIMIT"] = strconv.Itoa(constants.VCenterScanConcurrencyLimit)
	}

	if vjailbreakSettingsCM.Data["CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE"] == "" {
		vjailbreakSettingsCM.Data["CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE"] = strconv.FormatBool(constants.CleanupVolumesAfterConvertFailure)
	}

	if vjailbreakSettingsCM.Data["CLEANUP_PORTS_AFTER_MIGRATION_FAILURE"] == "" {
		vjailbreakSettingsCM.Data["CLEANUP_PORTS_AFTER_MIGRATION_FAILURE"] = strconv.FormatBool(constants.CleanupPortsAfterMigrationFailure)
	}

	if vjailbreakSettingsCM.Data["POPULATE_VMWARE_MACHINE_FLAVORS"] == "" {
		vjailbreakSettingsCM.Data["POPULATE_VMWARE_MACHINE_FLAVORS"] = strconv.FormatBool(constants.PopulateVMwareMachineFlavors)
	}

	if vjailbreakSettingsCM.Data["VOLUME_AVAILABLE_WAIT_INTERVAL_SECONDS"] == "" {
		vjailbreakSettingsCM.Data["VOLUME_AVAILABLE_WAIT_INTERVAL_SECONDS"] = strconv.Itoa(constants.VolumeAvailableWaitIntervalSeconds)
	}

	if vjailbreakSettingsCM.Data["VOLUME_AVAILABLE_WAIT_RETRY_LIMIT"] == "" {
		vjailbreakSettingsCM.Data["VOLUME_AVAILABLE_WAIT_RETRY_LIMIT"] = strconv.Itoa(constants.VolumeAvailableWaitRetryLimit)
	}

	if vjailbreakSettingsCM.Data["VCENTER_LOGIN_RETRY_LIMIT"] == "" {
		vjailbreakSettingsCM.Data["VCENTER_LOGIN_RETRY_LIMIT"] = strconv.Itoa(constants.VCenterLoginRetryLimit)
	}

	if vjailbreakSettingsCM.Data["OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES"] == "" {
		vjailbreakSettingsCM.Data["OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES"] = strconv.Itoa(constants.OpenstackCredsRequeueAfterMinutes)
	}

	if vjailbreakSettingsCM.Data["VMWARE_CREDS_REQUEUE_AFTER_MINUTES"] == "" {
		vjailbreakSettingsCM.Data["VMWARE_CREDS_REQUEUE_AFTER_MINUTES"] = strconv.Itoa(constants.VMwareCredsRequeueAfterMinutes)
	}

	if vjailbreakSettingsCM.Data[constants.ValidateRDMOwnerVMsKey] == "" {
		vjailbreakSettingsCM.Data[constants.ValidateRDMOwnerVMsKey] = strconv.FormatBool(constants.ValidateRDMOwnerVMs)
	}

	if vjailbreakSettingsCM.Data[constants.AutoFstabUpdateKey] == "" {
		vjailbreakSettingsCM.Data[constants.AutoFstabUpdateKey] = strconv.FormatBool(constants.AutoFstabUpdate)
	}

	if vjailbreakSettingsCM.Data[constants.AutoPXEBootOnConversionKey] == "" {
		vjailbreakSettingsCM.Data[constants.AutoPXEBootOnConversionKey] = strconv.FormatBool(constants.AutoPXEBootOnConversionDefault)
	}

	if vjailbreakSettingsCM.Data[constants.V2VHelperPodCPURequestKey] == "" {
		vjailbreakSettingsCM.Data[constants.V2VHelperPodCPURequestKey] = constants.V2VHelperPodCPURequest
	}

	if vjailbreakSettingsCM.Data[constants.V2VHelperPodMemoryRequestKey] == "" {
		vjailbreakSettingsCM.Data[constants.V2VHelperPodMemoryRequestKey] = constants.V2VHelperPodMemoryRequest
	}

	if vjailbreakSettingsCM.Data[constants.V2VHelperPodCPULimitKey] == "" {
		vjailbreakSettingsCM.Data[constants.V2VHelperPodCPULimitKey] = constants.V2VHelperPodCPULimit
	}

	if vjailbreakSettingsCM.Data[constants.V2VHelperPodMemoryLimitKey] == "" {
		vjailbreakSettingsCM.Data[constants.V2VHelperPodMemoryLimitKey] = constants.V2VHelperPodMemoryLimit
	}

	if vjailbreakSettingsCM.Data[constants.V2VHelperPodEphemeralStorageRequestKey] == "" {
		vjailbreakSettingsCM.Data[constants.V2VHelperPodEphemeralStorageRequestKey] = constants.V2VHelperPodEphemeralStorageRequest
	}

	if vjailbreakSettingsCM.Data[constants.V2VHelperPodEphemeralStorageLimitKey] == "" {
		vjailbreakSettingsCM.Data[constants.V2VHelperPodEphemeralStorageLimitKey] = constants.V2VHelperPodEphemeralStorageLimit
	}

	return &VjailbreakSettings{
		ChangedBlocksCopyIterationThreshold: atoi(vjailbreakSettingsCM.Data["CHANGED_BLOCKS_COPY_ITERATION_THRESHOLD"]),
		PeriodicSyncInterval:                vjailbreakSettingsCM.Data["PERIODIC_SYNC_INTERVAL"],
		VMActiveWaitIntervalSeconds:         atoi(vjailbreakSettingsCM.Data["VM_ACTIVE_WAIT_INTERVAL_SECONDS"]),
		VMActiveWaitRetryLimit:              atoi(vjailbreakSettingsCM.Data["VM_ACTIVE_WAIT_RETRY_LIMIT"]),
		DefaultMigrationMethod:              vjailbreakSettingsCM.Data["DEFAULT_MIGRATION_METHOD"],
		VCenterScanConcurrencyLimit:         atoi(vjailbreakSettingsCM.Data["VCENTER_SCAN_CONCURRENCY_LIMIT"]),
		CleanupVolumesAfterConvertFailure:   vjailbreakSettingsCM.Data["CLEANUP_VOLUMES_AFTER_CONVERT_FAILURE"] == "true",
		CleanupPortsAfterMigrationFailure:   vjailbreakSettingsCM.Data["CLEANUP_PORTS_AFTER_MIGRATION_FAILURE"] == "true",
		PopulateVMwareMachineFlavors:        vjailbreakSettingsCM.Data["POPULATE_VMWARE_MACHINE_FLAVORS"] == "true",
		VolumeAvailableWaitIntervalSeconds:  atoi(vjailbreakSettingsCM.Data["VOLUME_AVAILABLE_WAIT_INTERVAL_SECONDS"]),
		VolumeAvailableWaitRetryLimit:       atoi(vjailbreakSettingsCM.Data["VOLUME_AVAILABLE_WAIT_RETRY_LIMIT"]),
		VCenterLoginRetryLimit:              atoi(vjailbreakSettingsCM.Data["VCENTER_LOGIN_RETRY_LIMIT"]),
		OpenstackCredsRequeueAfterMinutes:   atoi(vjailbreakSettingsCM.Data["OPENSTACK_CREDS_REQUEUE_AFTER_MINUTES"]),
		VMwareCredsRequeueAfterMinutes:      atoi(vjailbreakSettingsCM.Data["VMWARE_CREDS_REQUEUE_AFTER_MINUTES"]),
		ValidateRDMOwnerVMs:                 strings.ToLower(strings.TrimSpace(vjailbreakSettingsCM.Data[constants.ValidateRDMOwnerVMsKey])) == "true",
		PeriodicSyncMaxRetries:              uint64(atoi(vjailbreakSettingsCM.Data["PERIODIC_SYNC_MAX_RETRIES"])),
		PeriodicSyncRetryCap:                vjailbreakSettingsCM.Data["PERIODIC_SYNC_RETRY_CAP"],
		AutoFstabUpdate:                     strings.ToLower(strings.TrimSpace(vjailbreakSettingsCM.Data[constants.AutoFstabUpdateKey])) == "true",
		AutoPXEBootOnConversion:             strings.ToLower(strings.TrimSpace(vjailbreakSettingsCM.Data[constants.AutoPXEBootOnConversionKey])) == "true",
		V2VHelperPodCPURequest:              vjailbreakSettingsCM.Data[constants.V2VHelperPodCPURequestKey],
		V2VHelperPodMemoryRequest:           vjailbreakSettingsCM.Data[constants.V2VHelperPodMemoryRequestKey],
		V2VHelperPodCPULimit:                vjailbreakSettingsCM.Data[constants.V2VHelperPodCPULimitKey],
		V2VHelperPodMemoryLimit:             vjailbreakSettingsCM.Data[constants.V2VHelperPodMemoryLimitKey],
		V2VHelperPodEphemeralStorageRequest: vjailbreakSettingsCM.Data[constants.V2VHelperPodEphemeralStorageRequestKey],
		V2VHelperPodEphemeralStorageLimit:   vjailbreakSettingsCM.Data[constants.V2VHelperPodEphemeralStorageLimitKey],
	}, nil
}

func GetArrayCredsMapping(ctx context.Context, k8sClient client.Client, arrayCredsMappingName string) (vjailbreakv1alpha1.ArrayCredsMapping, error) {
	arrayCredsMapping := vjailbreakv1alpha1.ArrayCredsMapping{}
	if err := k8sClient.Get(ctx, k8stypes.NamespacedName{Name: arrayCredsMappingName, Namespace: constants.NamespaceMigrationSystem}, &arrayCredsMapping); err != nil {
		return vjailbreakv1alpha1.ArrayCredsMapping{}, errors.Wrap(err, "failed to get array creds mapping configmap")
	}
	return arrayCredsMapping, nil
}

func GetArrayCreds(ctx context.Context, k8sClient client.Client, arrayCredsName string) (vjailbreakv1alpha1.ArrayCreds, error) {
	arrayCreds := vjailbreakv1alpha1.ArrayCreds{}
	if err := k8sClient.Get(ctx, k8stypes.NamespacedName{Name: arrayCredsName, Namespace: constants.NamespaceMigrationSystem}, &arrayCreds); err != nil {
		return vjailbreakv1alpha1.ArrayCreds{}, errors.Wrap(err, "failed to get array creds configmap")
	}
	return arrayCreds, nil
}

// GetESXiSSHPrivateKey retrieves the ESXi SSH private key from a Kubernetes secret
func GetESXiSSHPrivateKey(ctx context.Context, k8sClient client.Client, secretName string) ([]byte, error) {
	secret := &corev1.Secret{}
	if err := k8sClient.Get(ctx, k8stypes.NamespacedName{
		Name:      secretName,
		Namespace: constants.NamespaceMigrationSystem,
	}, secret); err != nil {
		return nil, errors.Wrapf(err, "failed to get ESXi SSH secret %s", secretName)
	}

	// The secret should contain a key named "ssh-privatekey"
	privateKey, ok := secret.Data["ssh-privatekey"]
	if !ok {
		return nil, fmt.Errorf("secret %s does not contain 'ssh-privatekey' key", secretName)
	}

	if len(privateKey) == 0 {
		return nil, fmt.Errorf("ESXi SSH private key in secret %s is empty", secretName)
	}

	return privateKey, nil
}
