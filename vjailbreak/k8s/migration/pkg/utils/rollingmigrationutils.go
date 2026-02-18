package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/constants"
	scope "github.com/platform9/vjailbreak/k8s/migration/pkg/scope"
	providers "github.com/platform9/vjailbreak/pkg/vpwned/sdk/providers"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CreateClusterMigration creates a new ClusterMigration object from the given cluster info and rolling migration plan
func CreateClusterMigration(ctx context.Context, k8sClient client.Client, cluster vjailbreakv1alpha1.ClusterMigrationInfo, rollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan) (*vjailbreakv1alpha1.ClusterMigration, error) {
	ESXiSequence := GetESXiSequenceFromVMSequence(ctx, cluster.VMSequence)
	if len(ESXiSequence) == 0 {
		return nil, errors.New("ESXi host sequence cannot be empty")
	}
	vmwarecreds, err := GetVMwareCredsFromRollingMigrationPlan(ctx, k8sClient, rollingMigrationPlan)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get vmware credentials")
	}
	clusterK8sName, err := GetK8sCompatibleVMWareObjectName(cluster.ClusterName, vmwarecreds.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert cluster name to k8s name")
	}
	migrationTemplate := vjailbreakv1alpha1.MigrationTemplate{}
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      rollingMigrationPlan.Spec.MigrationTemplate,
		Namespace: constants.NamespaceMigrationSystem},
		&migrationTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to get migration template")
	}
	clusterMigration := &vjailbreakv1alpha1.ClusterMigration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GenerateRollingMigrationObjectName(clusterK8sName, rollingMigrationPlan),
			Namespace: constants.NamespaceMigrationSystem,
		},
		Spec: vjailbreakv1alpha1.ClusterMigrationSpec{
			ClusterName:           cluster.ClusterName,
			ESXIMigrationSequence: ESXiSequence,
			RollingMigrationPlanRef: corev1.LocalObjectReference{
				Name: rollingMigrationPlan.Name,
			},
			OpenstackCredsRef: corev1.LocalObjectReference{
				Name: migrationTemplate.Spec.Destination.OpenstackRef,
			},
			VMwareCredsRef: corev1.LocalObjectReference{
				Name: migrationTemplate.Spec.Source.VMwareRef,
			},
		},
	}
	if err := controllerutil.SetOwnerReference(rollingMigrationPlan, clusterMigration, k8sClient.Scheme()); err != nil {
		return nil, fmt.Errorf("failed to set owner reference on ClusterMigration: %w", err)
	}

	if err := k8sClient.Create(ctx, clusterMigration); err != nil {
		return nil, err
	}
	return clusterMigration, nil
}

// getMigrationObject is a helper function that retrieves a migration object for a given VMware object name and rolling migration plan
func getMigrationObject(ctx context.Context, k8sClient client.Client, vmwareObjectName string, rollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan, obj client.Object, errorMsg string) error {
	vmwarecreds, err := GetVMwareCredsFromRollingMigrationPlan(ctx, k8sClient, rollingMigrationPlan)
	if err != nil {
		return errors.Wrap(err, "failed to get vmware credentials")
	}

	k8sName, err := GetK8sCompatibleVMWareObjectName(vmwareObjectName, vmwarecreds.Name)
	if err != nil {
		return errors.Wrap(err, errorMsg)
	}

	return k8sClient.Get(ctx, types.NamespacedName{
		Name:      GenerateRollingMigrationObjectName(k8sName, rollingMigrationPlan),
		Namespace: constants.NamespaceMigrationSystem,
	}, obj)
}

// GetClusterMigration retrieves a ClusterMigration object for the given cluster name and rolling migration plan
func GetClusterMigration(ctx context.Context, k8sClient client.Client, clusterName string, rollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan) (*vjailbreakv1alpha1.ClusterMigration, error) {
	clusterMigration := &vjailbreakv1alpha1.ClusterMigration{}
	err := getMigrationObject(ctx, k8sClient, clusterName, rollingMigrationPlan, clusterMigration, "failed to convert cluster name to k8s name")
	if err != nil {
		return nil, err
	}
	return clusterMigration, nil
}

// GetESXIMigration retrieves an ESXIMigration object for the given ESXi host name and rolling migration plan
func GetESXIMigration(ctx context.Context, k8sClient client.Client, esxi string, rollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan) (*vjailbreakv1alpha1.ESXIMigration, error) {
	esxiMigration := &vjailbreakv1alpha1.ESXIMigration{}
	err := getMigrationObject(ctx, k8sClient, esxi, rollingMigrationPlan, esxiMigration, "failed to convert ESXi name to k8s name")
	if err != nil {
		return nil, err
	}
	return esxiMigration, nil
}

// GetMigrationPlan retrieves a MigrationPlan object by name
func GetMigrationPlan(ctx context.Context, k8sClient client.Client, migrationPlanName string) (*vjailbreakv1alpha1.MigrationPlan, error) {
	migrationPlan := &vjailbreakv1alpha1.MigrationPlan{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: migrationPlanName, Namespace: constants.NamespaceMigrationSystem}, migrationPlan); err != nil {
		return nil, err
	}
	return migrationPlan, nil
}

// GetMigrationTemplate retrieves a MigrationTemplate object for the given VM
func GetMigrationTemplate(ctx context.Context, k8sClient client.Client, vm string, _ *vjailbreakv1alpha1.RollingMigrationPlan) (*vjailbreakv1alpha1.MigrationTemplate, error) {
	migrationTemplate := &vjailbreakv1alpha1.MigrationTemplate{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: vm, Namespace: constants.NamespaceMigrationSystem}, migrationTemplate); err != nil {
		return nil, err
	}
	return migrationTemplate, nil
}

// CreateESXIMigration creates a new ESXIMigration object for the given ESXi host using the cluster migration scope
func CreateESXIMigration(ctx context.Context, scope *scope.ClusterMigrationScope, esxi string) (*vjailbreakv1alpha1.ESXIMigration, error) {
	vmwarecreds, err := GetVMwareCredsFromRollingMigrationPlan(ctx, scope.Client, scope.RollingMigrationPlan)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get vmware credentials")
	}
	esxiK8sName, err := GetK8sCompatibleVMWareObjectName(esxi, vmwarecreds.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert ESXi name to k8s name")
	}
	esxiMigration := &vjailbreakv1alpha1.ESXIMigration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GenerateRollingMigrationObjectName(esxiK8sName, scope.RollingMigrationPlan),
			Namespace: constants.NamespaceMigrationSystem,
			Labels: map[string]string{
				constants.ESXiNameLabel:             esxiK8sName,
				constants.VMwareCredsLabel:          scope.ClusterMigration.Spec.VMwareCredsRef.Name,
				constants.RollingMigrationPlanLabel: scope.RollingMigrationPlan.Name,
				constants.ClusterMigrationLabel:     scope.ClusterMigration.Name,
			},
		},
		Spec: vjailbreakv1alpha1.ESXIMigrationSpec{
			ESXiName: esxi,
			RollingMigrationPlanRef: corev1.LocalObjectReference{
				Name: scope.RollingMigrationPlan.Name,
			},
			OpenstackCredsRef: scope.ClusterMigration.Spec.OpenstackCredsRef,
			VMwareCredsRef:    scope.ClusterMigration.Spec.VMwareCredsRef,
		},
	}
	if err := controllerutil.SetOwnerReference(scope.RollingMigrationPlan, esxiMigration, scope.Client.Scheme()); err != nil {
		return nil, fmt.Errorf("failed to set owner reference on ESXiMigration: %w", err)
	}
	if err := scope.Client.Create(ctx, esxiMigration); err != nil {
		return nil, err
	}
	return esxiMigration, nil
}

// GetESXiSequenceFromVMSequence extracts a unique sequence of ESXi host names from the VM sequence information
func GetESXiSequenceFromVMSequence(_ context.Context, vmSequence []vjailbreakv1alpha1.VMSequenceInfo) []string {
	esxiSequence := []string{}
	for _, vm := range vmSequence {
		esxiSequence = AppendUnique(esxiSequence, vm.ESXiName)
	}

	return esxiSequence
}

// AddVMsToESXIMigrationStatus adds the list of VM names to the ESXIMigration status for tracking
func AddVMsToESXIMigrationStatus(ctx context.Context, scope *scope.ClusterMigrationScope, esxi string) error {
	vmwarecreds, err := GetVMwareCredsFromRollingMigrationPlan(ctx, scope.Client, scope.RollingMigrationPlan)
	if err != nil {
		return errors.Wrap(err, "failed to get vmware credentials")
	}
	esxiK8sName, err := GetK8sCompatibleVMWareObjectName(esxi, vmwarecreds.Name)
	if err != nil {
		return errors.Wrap(err, "failed to convert ESXi name to k8s name")
	}
	esxiMigration := &vjailbreakv1alpha1.ESXIMigration{}
	if err := scope.Client.Get(ctx, types.NamespacedName{Name: GenerateRollingMigrationObjectName(esxiK8sName, scope.RollingMigrationPlan), Namespace: constants.NamespaceMigrationSystem}, esxiMigration); err != nil {
		return errors.Wrap(err, "failed to get ESXi migration status")
	}

	vmList := vjailbreakv1alpha1.VMwareMachineList{}

	if err := scope.Client.List(ctx, &vmList, client.InNamespace(constants.NamespaceMigrationSystem), client.MatchingLabels{constants.ESXiNameLabel: esxiK8sName, constants.VMwareCredsLabel: esxiMigration.Spec.VMwareCredsRef.Name}); err != nil {
		return errors.Wrap(err, "failed to get ESXi migration status")
	}

	for _, vmName := range vmList.Items {
		esxiMigration.Status.VMs = append(esxiMigration.Status.VMs, vmName.Name)
	}

	if err := scope.Client.Status().Update(ctx, esxiMigration); err != nil {
		return errors.Wrap(err, "failed to update ESXi migration status")
	}
	return nil
}

// GetESXiHostSystem returns a reference to an ESXi host system using VMware credentials
func GetESXiHostSystem(ctx context.Context, k8sClient client.Client, esxiName string, vmwareCredsRef corev1.LocalObjectReference, datacenter string) (*object.HostSystem, *vim25.Client, error) {
	vmwarecreds := &vjailbreakv1alpha1.VMwareCreds{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: constants.NamespaceMigrationSystem, Name: vmwareCredsRef.Name}, vmwarecreds)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get vmware credentials")
	}

	c, err := ValidateVMwareCreds(ctx, k8sClient, vmwarecreds)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to validate vCenter connection")
	}
	if c != nil {
		defer c.CloseIdleConnections()
		defer func() {
			if err := LogoutVMwareClient(ctx, k8sClient, vmwarecreds, c); err != nil {
				log.FromContext(ctx).Error(err, "Failed to logout VMware client")
			}
		}()
	}
	finder := find.NewFinder(c, false)
	dcName := datacenter
	if dcName == "" {
		dcName = vmwarecreds.Spec.DataCenter
	}
	dc, err := finder.Datacenter(ctx, dcName)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to find datacenter")
	}
	finder.SetDatacenter(dc)

	hostSystem, err := finder.HostSystem(ctx, esxiName)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to find host %s", esxiName)
	}

	return hostSystem, c, nil
}

// PutESXiInMaintenanceMode places the ESXi host into maintenance mode to prepare for migration
func PutESXiInMaintenanceMode(ctx context.Context, k8sClient client.Client, scope *scope.ESXIMigrationScope) error {
	hostSystem, _, err := GetESXiHostSystem(ctx, k8sClient, scope.ESXIMigration.Spec.ESXiName, scope.ESXIMigration.Spec.VMwareCredsRef, "")
	if err != nil {
		return errors.Wrap(err, "failed to get ESXi host system")
	}

	// Check host state
	var hs mo.HostSystem
	err = hostSystem.Properties(ctx, hostSystem.Reference(), []string{"runtime.connectionState"}, &hs)
	if err != nil {
		return errors.Wrap(err, "failed to get host properties")
	}

	// Put host into maintenance mode
	task, err := hostSystem.EnterMaintenanceMode(ctx, 300, true, nil)
	if err != nil {
		return errors.Wrap(err, "failed to initiate maintenance mode")
	}

	esxiK8sName, err := GetK8sCompatibleVMWareObjectName(scope.ESXIMigration.Spec.ESXiName, scope.ESXIMigration.Spec.VMwareCredsRef.Name)
	if err != nil {
		return errors.Wrap(err, "failed to convert ESXi name to k8s name")
	}

	esxiMigration := &vjailbreakv1alpha1.ESXIMigration{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: GenerateRollingMigrationObjectName(esxiK8sName, scope.RollingMigrationPlan), Namespace: constants.NamespaceMigrationSystem}, esxiMigration); err != nil {
		return err
	}

	esxiMigration.Status.Phase = vjailbreakv1alpha1.ESXIMigrationPhaseInMaintenanceMode
	err = k8sClient.Status().Update(ctx, esxiMigration)
	if err != nil {
		return errors.Wrap(err, "failed to update ESXi migration status")
	}

	err = task.Wait(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to wait for host to enter maintenance mode")
	}

	return nil
}

// CheckESXiInMaintenanceMode checks if the ESXi host is currently in maintenance mode
func CheckESXiInMaintenanceMode(ctx context.Context, k8sClient client.Client, scope *scope.ESXIMigrationScope) (bool, error) {
	vmwarecreds := &vjailbreakv1alpha1.VMwareCreds{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: constants.NamespaceMigrationSystem, Name: scope.ESXIMigration.Spec.VMwareCredsRef.Name}, vmwarecreds)
	if err != nil {
		return false, errors.Wrap(err, "failed to get vmware credentials")
	}

	hs, err := GetESXiSummary(ctx, k8sClient, scope.ESXIMigration.Spec.ESXiName, vmwarecreds, "")
	if err != nil {
		return false, errors.Wrap(err, "failed to get ESXi summary")
	}

	if hs.Summary.Runtime.InMaintenanceMode {
		return true, nil
	}

	return false, nil
}

// GetESXiSummary retrieves detailed host system information for the given ESXi host
func GetESXiSummary(ctx context.Context, k8sClient client.Client, esxiName string, vmwareCreds *vjailbreakv1alpha1.VMwareCreds, datacenter string) (mo.HostSystem, error) {
	// Create a temporary reference to use with our common function
	vmwareCredsRef := corev1.LocalObjectReference{
		Name: vmwareCreds.Name,
	}

	hostSystem, c, err := GetESXiHostSystem(ctx, k8sClient, esxiName, vmwareCredsRef, datacenter)
	if err != nil {
		return mo.HostSystem{}, errors.Wrap(err, "failed to get ESXi host system")
	}

	pc := property.DefaultCollector(c)

	var hs mo.HostSystem
	err = pc.RetrieveOne(ctx, hostSystem.Reference(), []string{"summary", "config", "hardware"}, &hs)
	if err != nil {
		return mo.HostSystem{}, errors.Wrap(err, "failed to get host properties")
	}
	return hs, nil
}

// RemoveESXiFromVCenter removes an ESXi host from vCenter inventory
// This should only be called after the host is in maintenance mode and has no VMs
func RemoveESXiFromVCenter(ctx context.Context, k8sClient client.Client, scope *scope.ESXIMigrationScope) error {
	hostSystem, _, err := GetESXiHostSystem(ctx, k8sClient, scope.ESXIMigration.Spec.ESXiName, scope.ESXIMigration.Spec.VMwareCredsRef, "")
	if err != nil {
		return errors.Wrap(err, "failed to get ESXi host system")
	}

	// Verify the host is in maintenance mode before removing
	inMaintenance, err := CheckESXiInMaintenanceMode(ctx, k8sClient, scope)
	if err != nil {
		return errors.Wrap(err, "failed to check maintenance mode status")
	}

	if !inMaintenance {
		return errors.New("cannot remove ESXi host that is not in maintenance mode")
	}

	// Verify no VMs exist on the host
	vmCount, err := CountVMsOnESXi(ctx, k8sClient, scope)
	if err != nil {
		return errors.Wrap(err, "failed to count VMs on host")
	}

	if vmCount > 0 {
		return errors.New("cannot remove ESXi host with existing VMs")
	}

	// Remove the host from vCenter inventory
	task, err := hostSystem.Destroy(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to destroy host from vCenter")
	}

	err = task.Wait(ctx)
	if err != nil {
		return errors.Wrap(err, "failed while waiting for host removal task to complete")
	}

	return nil
}

// CountVMsOnESXi counts the number of virtual machines currently hosted on the ESXi host
func CountVMsOnESXi(ctx context.Context, k8sClient client.Client, scope *scope.ESXIMigrationScope) (int, error) {
	hostSystem, _, err := GetESXiHostSystem(ctx, k8sClient, scope.ESXIMigration.Spec.ESXiName, scope.ESXIMigration.Spec.VMwareCredsRef, "")
	if err != nil {
		return 0, errors.Wrap(err, "failed to get ESXi host system")
	}

	// Get the VMs on the host
	var host mo.HostSystem
	err = hostSystem.Properties(ctx, hostSystem.Reference(), []string{"vm"}, &host)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get VM properties")
	}

	return len(host.Vm), nil
}

// deepMerge performs a deep merge of src into dst
// src values override dst values when there's a conflict
func boolPtr(b bool) *bool {
	return &b
}

func deepMerge(dst, src map[string]interface{}) map[string]interface{} {
	for key, srcVal := range src {
		dstVal, exists := dst[key]

		// If the key doesn't exist in dst, just set it
		if !exists {
			dst[key] = srcVal
			continue
		}

		// If both values are maps, recursively merge them
		srcMap, srcIsMap := srcVal.(map[string]interface{})
		dstMap, dstIsMap := dstVal.(map[string]interface{})

		if srcIsMap && dstIsMap {
			dst[key] = deepMerge(dstMap, srcMap)
			continue
		}

		// For lists/arrays, we need special handling
		srcSlice, srcIsSlice := srcVal.([]interface{})
		dstSlice, dstIsSlice := dstVal.([]interface{})

		if srcIsSlice && dstIsSlice {
			// Simple merge: just append elements from src to dst
			// Another option could be to replace, or do a merge based on specific rules
			dstSlice = append(dstSlice, srcSlice...)
			dst[key] = dstSlice
			continue
		}

		// In all other cases (primitive types, or type mismatch), src overwrites dst
		dst[key] = srcVal
	}

	return dst
}

// GenerateRollingMigrationObjectName creates a unique name for migration objects based on the rolling migration plan
func GenerateRollingMigrationObjectName(objectName string, rollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan) string {
	return fmt.Sprintf("%s-%s", objectName, rollingMigrationPlan.Name)
}

// GenerateVMwareCredsDependantObjectName creates a unique name for objects that depend on VMware credentials
func GenerateVMwareCredsDependantObjectName(objectName string, vmwareCredsName string) string {
	return fmt.Sprintf("%s-%s", objectName, vmwareCredsName)
}

// UpdateESXiNamesInRollingMigrationPlan updates the ESXi host names in the rolling migration plan
func UpdateESXiNamesInRollingMigrationPlan(ctx context.Context, scope *scope.RollingMigrationPlanScope) error {
	vmwarecreds, err := GetVMwareCredsFromRollingMigrationPlan(ctx, scope.Client, scope.RollingMigrationPlan)
	if err != nil {
		return errors.Wrap(err, "failed to get vmware credentials")
	}
	// Update ESXi Name in RollingMigrationPlan for each VM in VM Sequence
	for i, cluster := range scope.RollingMigrationPlan.Spec.ClusterSequence {
		for j := range cluster.VMSequence {
			k8sVMName, err := GetK8sCompatibleVMWareObjectName(cluster.VMSequence[j].VMName, vmwarecreds.Name)
			if err != nil {
				return errors.Wrap(err, "failed to get vm name")
			}
			vm := &vjailbreakv1alpha1.VMwareMachine{}
			err = scope.Client.Get(ctx, client.ObjectKey{
				Name:      k8sVMName,
				Namespace: scope.Namespace(),
			}, vm)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Error getting VMInfo for VM '%s'", cluster.VMSequence[j].VMName))
			}
			scope.RollingMigrationPlan.Spec.ClusterSequence[i].VMSequence[j].ESXiName = vm.Spec.VMInfo.ESXiName
		}
	}
	return nil
}

// ConvertVMSequenceToMigrationPlans converts a VM sequence into multiple migration plans based on batch size
func ConvertVMSequenceToMigrationPlans(ctx context.Context, scope *scope.ClusterMigrationScope, batchSize int) error {
	if batchSize <= 0 {
		return fmt.Errorf("batch size must be greater than 0")
	}

	rollingMigrationPlan := scope.RollingMigrationPlan

	if len(rollingMigrationPlan.Spec.VMMigrationPlans) != 0 {
		return nil
	}

	// Collect all VM names from all clusters
	batches := convertVMSequenceToBatches(scope, batchSize)

	// Create a MigrationPlan for each batch
	for i, batch := range batches {
		err := convertBatchToMigrationPlan(ctx, scope, batch, i)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				continue
			}
			return errors.Wrap(err, "failed to convert batch to migration plan")
		}
	}

	// Log summary of batches created
	klog.Infof("Created %d batches with a maximum of %d VMs per batch", len(batches), batchSize)

	return nil
}

func convertBatchToMigrationPlan(ctx context.Context, scope *scope.ClusterMigrationScope, batch []string, i int) error {
	rollingMigrationPlan := scope.RollingMigrationPlan

	// Create a migration plan for this batch
	migrationPlan := vjailbreakv1alpha1.MigrationPlan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-batch-%d", rollingMigrationPlan.Name, i),
			Namespace: rollingMigrationPlan.Namespace,
			// Set owner reference to the rolling migration plan
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: rollingMigrationPlan.APIVersion,
					Kind:       rollingMigrationPlan.Kind,
					Name:       rollingMigrationPlan.Name,
					UID:        rollingMigrationPlan.UID,
					Controller: boolPtr(true),
				},
			},
			Labels: map[string]string{
				constants.RollingMigrationPlanLabel: rollingMigrationPlan.Name,
			},
		},
		Spec: vjailbreakv1alpha1.MigrationPlanSpec{
			// Use the template name from existing plans or default
			MigrationPlanSpecPerVM: vjailbreakv1alpha1.MigrationPlanSpecPerVM{
				MigrationTemplate: rollingMigrationPlan.Spec.MigrationTemplate,
				MigrationStrategy: rollingMigrationPlan.Spec.MigrationStrategy,
				// Copy advanced options if needed
				AdvancedOptions: rollingMigrationPlan.Spec.AdvancedOptions,
			},
			// Include VM batch as a single group for migration
			VirtualMachines: [][]string{batch},
		},
		Status: vjailbreakv1alpha1.MigrationPlanStatus{
			MigrationStatus:  "Pending",
			MigrationMessage: fmt.Sprintf("Created by RollingMigrationPlan %s", rollingMigrationPlan.Name),
		},
	}

	// Create the migration plan
	err := scope.Client.Create(ctx, &migrationPlan)
	if err != nil {
		return errors.Wrap(err, "failed to create migration plan")
	}
	// Add the migration plan to the rolling migration plan
	rollingMigrationPlan.Spec.VMMigrationPlans = append(rollingMigrationPlan.Spec.VMMigrationPlans, migrationPlan.Name)
	return nil
}

func convertVMSequenceToBatches(scope *scope.ClusterMigrationScope, batchSize int) [][]string {
	var batches [][]string
	rollingMigrationPlan := scope.RollingMigrationPlan

	for _, cluster := range rollingMigrationPlan.Spec.ClusterSequence {
		allVMs := make([]string, 0, len(cluster.VMSequence))
		for _, vm := range cluster.VMSequence {
			allVMs = append(allVMs, vm.VMName)
		}

		// Create batches of VMs
		for i := 0; i < len(allVMs); i += batchSize {
			end := i + batchSize
			if end > len(allVMs) {
				end = len(allVMs)
			}
			batches = append(batches, allVMs[i:end])
		}
	}
	return batches
}

// IsRollingMigrationPlanPaused checks if a rolling migration plan is currently paused
func IsRollingMigrationPlanPaused(ctx context.Context, name string, client client.Client) bool {
	rollingMigrationPlan := &vjailbreakv1alpha1.RollingMigrationPlan{}
	if err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: constants.NamespaceMigrationSystem}, rollingMigrationPlan); err != nil {
		return false
	}
	if rollingMigrationPlan.Labels == nil {
		return false
	}
	return rollingMigrationPlan.Labels[constants.PauseMigrationLabel] == trueString
}

// IsClusterMigrationPaused checks if a cluster migration is currently paused
func IsClusterMigrationPaused(ctx context.Context, name string, client client.Client) bool {
	clusterMigration := &vjailbreakv1alpha1.ClusterMigration{}
	if err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: constants.NamespaceMigrationSystem}, clusterMigration); err != nil {
		return false
	}
	if clusterMigration.Labels == nil {
		return false
	}
	return clusterMigration.Labels[constants.PauseMigrationLabel] == trueString
}

// IsESXIMigrationPaused checks if an ESXi migration is currently paused
func IsESXIMigrationPaused(ctx context.Context, name string, client client.Client) bool {
	esxiMigration := &vjailbreakv1alpha1.ESXIMigration{}
	if err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: constants.NamespaceMigrationSystem}, esxiMigration); err != nil {
		return false
	}
	if esxiMigration.Labels == nil {
		return false
	}
	return esxiMigration.Labels[constants.PauseMigrationLabel] == trueString
}

// IsMigrationPlanPaused checks if a migration plan is currently paused
func IsMigrationPlanPaused(ctx context.Context, name string, client client.Client) bool {
	migrationPlan := &vjailbreakv1alpha1.MigrationPlan{}
	if err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: constants.NamespaceMigrationSystem}, migrationPlan); err != nil {
		return false
	}
	if migrationPlan.Labels == nil {
		return false
	}
	return migrationPlan.Labels[constants.PauseMigrationLabel] == trueString
}

// PauseRollingMigrationPlan pauses all migration operations for a rolling migration plan
func PauseRollingMigrationPlan(ctx context.Context, scope *scope.RollingMigrationPlanScope) error {
	log := scope.Logger
	rollingMigrationPlan := scope.RollingMigrationPlan
	if rollingMigrationPlan.Labels == nil {
		rollingMigrationPlan.Labels = make(map[string]string)
	}

	// Set label on rolling migration plan
	rollingMigrationPlan.Labels[constants.PauseMigrationLabel] = trueString

	// Update all child ClusterMigrations
	for _, cluster := range rollingMigrationPlan.Spec.ClusterSequence {
		// Get the cluster migration
		clusterMigration, err := GetClusterMigration(ctx, scope.Client, cluster.ClusterName, rollingMigrationPlan)
		if err != nil {
			log.Error(err, "failed to get cluster migration", "cluster", cluster.ClusterName)
			continue
		}

		// Add label to cluster migration
		if clusterMigration.Labels == nil {
			clusterMigration.Labels = make(map[string]string)
		}
		clusterMigration.Labels[constants.PauseMigrationLabel] = trueString
		err = scope.Client.Update(ctx, clusterMigration)
		if err != nil {
			log.Error(err, "failed to update cluster migration with pause label", "cluster", cluster.ClusterName)
		}

		// Get unique ESXi hosts from the VM sequence
		esxiHosts := make(map[string]bool)
		for _, vm := range cluster.VMSequence {
			if vm.ESXiName != "" {
				esxiHosts[vm.ESXiName] = true
			}
		}

		// Find and update all ESXi migrations for this cluster
		for esxi := range esxiHosts {
			esxiMigration, err := GetESXIMigration(ctx, scope.Client, esxi, rollingMigrationPlan)
			if err != nil {
				log.Error(err, "failed to get ESXi migration", "esxi", esxi)
				continue
			}

			// Add label to ESXi migration
			if esxiMigration.Labels == nil {
				esxiMigration.Labels = make(map[string]string)
			}
			esxiMigration.Labels[constants.PauseMigrationLabel] = trueString
			err = scope.Client.Update(ctx, esxiMigration)
			if err != nil {
				log.Error(err, "failed to update ESXi migration with pause label", "esxi", esxi)
			}
		}
	}

	// Update all MigrationPlans
	for _, planName := range rollingMigrationPlan.Spec.VMMigrationPlans {
		migrationPlan := &vjailbreakv1alpha1.MigrationPlan{}
		err := scope.Client.Get(ctx, types.NamespacedName{Name: planName, Namespace: rollingMigrationPlan.Namespace}, migrationPlan)
		if err != nil {
			log.Error(err, "failed to get migration plan", "plan", planName)
			continue
		}

		// Add label to migration plan
		if migrationPlan.Labels == nil {
			migrationPlan.Labels = make(map[string]string)
		}
		migrationPlan.Labels[constants.PauseMigrationLabel] = trueString
		err = scope.Client.Update(ctx, migrationPlan)
		if err != nil {
			log.Error(err, "failed to update migration plan with pause label", "plan", planName)
		}
	}

	// Update the rolling migration plan itself
	return scope.Client.Update(ctx, rollingMigrationPlan)
}

// StringSlicesEqual compares two string slices and returns true if they contain the same elements (order sensitive)
func StringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// ResumeRollingMigrationPlan resumes all migration operations for a previously paused rolling migration plan
func ResumeRollingMigrationPlan(ctx context.Context, scope *scope.RollingMigrationPlanScope) error {
	log := scope.Logger
	log.Info("Resuming rolling migration plan", "rollingMigrationPlan", scope.RollingMigrationPlan.Name)
	rollingMigrationPlan := scope.RollingMigrationPlan
	if rollingMigrationPlan.Labels == nil {
		return nil
	}

	if _, ok := rollingMigrationPlan.Labels[constants.PauseMigrationLabel]; !ok {
		return nil
	}

	// Update all child ClusterMigrations
	for _, cluster := range rollingMigrationPlan.Spec.ClusterSequence {
		// Get the cluster migration
		clusterMigration, err := GetClusterMigration(ctx, scope.Client, cluster.ClusterName, rollingMigrationPlan)
		if err != nil {
			log.Error(err, "failed to get cluster migration", "cluster", cluster.ClusterName)
			continue
		}

		// Remove label from cluster migration
		if clusterMigration.Labels != nil {
			delete(clusterMigration.Labels, constants.PauseMigrationLabel)
			err = scope.Client.Update(ctx, clusterMigration)
			if err != nil {
				log.Error(err, "failed to update cluster migration to remove pause label", "cluster", cluster.ClusterName)
			}
		}

		// Get unique ESXi hosts from the VM sequence
		esxiHosts := make(map[string]bool)
		for _, vm := range cluster.VMSequence {
			if vm.ESXiName != "" {
				esxiHosts[vm.ESXiName] = true
			}
		}

		// Find and update all ESXi migrations for this cluster
		for esxi := range esxiHosts {
			esxiMigration, err := GetESXIMigration(ctx, scope.Client, esxi, rollingMigrationPlan)
			if err != nil {
				log.Error(err, "failed to get ESXi migration", "esxi", esxi)
				continue
			}

			// Remove label from ESXi migration
			if esxiMigration.Labels != nil {
				delete(esxiMigration.Labels, constants.PauseMigrationLabel)
				err = scope.Client.Update(ctx, esxiMigration)
				if err != nil {
					log.Error(err, "failed to update ESXi migration to remove pause label", "esxi", esxi)
				}
			}
		}
	}

	// Update all MigrationPlans
	for _, planName := range rollingMigrationPlan.Spec.VMMigrationPlans {
		migrationPlan := &vjailbreakv1alpha1.MigrationPlan{}
		err := scope.Client.Get(ctx, types.NamespacedName{Name: planName, Namespace: rollingMigrationPlan.Namespace}, migrationPlan)
		if err != nil {
			log.Error(err, "failed to get migration plan", "plan", planName)
			continue
		}

		// Remove label from migration plan
		if migrationPlan.Labels != nil {
			delete(migrationPlan.Labels, constants.PauseMigrationLabel)
			err = scope.Client.Update(ctx, migrationPlan)
			if err != nil {
				log.Error(err, "failed to update migration plan to remove pause label", "plan", planName)
			}
		}
	}

	// Remove pause label from rolling migration plan
	delete(rollingMigrationPlan.Labels, constants.PauseMigrationLabel)

	// Update the rolling migration plan itself
	return scope.Client.Update(ctx, rollingMigrationPlan)
}

// ValidateRollingMigrationPlan validates that a rolling migration plan meets all the prerequisites
// It checks VMware credentials, Bare Metal configurations, and ESXi host readiness
func ValidateRollingMigrationPlan(ctx context.Context, scope *scope.RollingMigrationPlanScope, configMap *corev1.ConfigMap) (bool, string, error) {
	config := GetRollingMigrationPlanValidationConfigFromConfigMap(configMap)
	if config == nil {
		return false, "", errors.New("failed to get rolling migration plan validation config")
	}

	// TODO(vpwned): validate vmwarecreds have enough permissions

	// TODO(vpwned): validate there is enough space on underlying storage array

	if !isBMConfigValid(ctx, scope.Client, scope.RollingMigrationPlan.Spec.BMConfigRef.Name) {
		return false, "", errors.New("BMConfig is not valid")
	}

	vmwareCreds, err := GetVMwareCredsFromRollingMigrationPlan(ctx, scope.Client, scope.RollingMigrationPlan)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to get vmware credentials")
	}

	vmwareCredsInfo, err := GetVMwareCredentialsFromSecret(ctx, scope.Client, vmwareCreds.Spec.SecretRef.Name)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to get vmware credentials from secret")
	}

	clusterK8sID := GetClusterK8sID(scope.RollingMigrationPlan.Spec.ClusterSequence[0].ClusterName, vmwareCredsInfo.Datacenter)

	vmwareHosts, err := FilterVMwareHostsForCluster(ctx, scope.Client, clusterK8sID)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to filter vmware hosts for cluster")
	}

	for _, vmwareHost := range vmwareHosts {
		// This checks if the ESXi host can enter maintenance mode
		// with all VMs automatically migrating off to other hosts.
		// It checks host, VM, and cluster settings that could block automatic VM migration.
		canEnterMaintenanceMode, reason, err := CanEnterMaintenanceMode(ctx, scope, vmwareCreds, vmwareHost.Spec.Name, *config)
		if err != nil {
			return false, "", errors.Wrap(err, "failed to check if ESXi can enter maintenance mode")
		}
		if !canEnterMaintenanceMode {
			return false, "", fmt.Errorf("esxi %s cannot be put in maintenance mode, due to %v", vmwareHost.Spec.Name, reason)
		}

		if config.CheckESXiInMAAS {
			// Ensure the ESXi host is in MAAS
			inMAAS, message, err := EnsureESXiInMass(ctx, scope, vmwareHost)
			if err != nil {
				return false, "", errors.Wrap(err, "failed to ensure ESXi is in MAAS")
			}
			if !inMAAS {
				return false, "", errors.New(message)
			}
		}
	}

	if config.CheckPCDHasClusterConfigured {
		// Ensure PCD has at-least one Cluster configured
		inPCD, message, err := EnsurePCDHasClusterConfigured(ctx, scope)
		if err != nil {
			return false, "", errors.Wrap(err, "failed to ensure PCD has at-least one Cluster configured")
		}
		if !inPCD {
			return false, "", errors.New(message)
		}
	}

	return true, "", nil
}

func isBMConfigValid(ctx context.Context, client client.Client, name string) bool {
	bmConfig := &vjailbreakv1alpha1.BMConfig{}
	err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: constants.NamespaceMigrationSystem}, bmConfig)
	if err != nil {
		return false
	}
	if bmConfig.Status.ValidationStatus != string(corev1.PodSucceeded) {
		return false
	}

	return true
}

// EnsureESXiInMass verifies that an ESXi host is correctly registered in the Metal-as-a-Service system
// and is in the appropriate state (Deployed or Allocated) for migration operations
func EnsureESXiInMass(ctx context.Context, scope *scope.RollingMigrationPlanScope, vmwarehost vjailbreakv1alpha1.VMwareHost) (bool, string, error) {
	ctxlog := log.FromContext(ctx)

	// Get maas provider
	bmConfig, err := GetBMConfigForRollingMigrationPlan(ctx, scope.Client, scope.RollingMigrationPlan)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to get BMConfig for rolling migration plan")
	}
	provider, err := providers.GetProvider(string(bmConfig.Spec.ProviderType))
	if err != nil {
		return false, "", errors.Wrap(err, "failed to get provider")
	}

	// list maas machines
	machines, err := provider.ListResources(ctx)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to list maas machines")
	}

	ctxlog.Info("Retrieved MAAS machines", "count", len(machines), "esxiName", vmwarehost.Spec.Name, "esxiHardwareUUID", vmwarehost.Spec.HardwareUUID)

	// First, try to match by hardware UUID
	var matchedMachineIdx = -1

	if vmwarehost.Spec.HardwareUUID != "" {
		ctxlog.Info("Attempting hardware UUID matching", "esxiHardwareUUID", vmwarehost.Spec.HardwareUUID)
		for i := range machines {
			ctxlog.V(1).Info("Checking MAAS machine",
				"index", i,
				"machineHardwareUuid", machines[i].HardwareUuid,
				"machineName", machines[i].Hostname,
				"machineId", machines[i].Id,
				"machineStatus", machines[i].Status)
			if machines[i].HardwareUuid != "" && machines[i].HardwareUuid == vmwarehost.Spec.HardwareUUID {
				ctxlog.Info("Found a matching machine by hardware UUID",
					"hardwareUuid", machines[i].HardwareUuid,
					"name", machines[i].Hostname,
					"id", machines[i].Id)
				matchedMachineIdx = i
				break
			}
		}
		if matchedMachineIdx == -1 {
			ctxlog.Info("No hardware UUID match found")
		}
	} else {
		ctxlog.Info("ESXi has empty hardware UUID, skipping UUID matching")
	}

	// If no UUID match found, fall back to MAC address matching
	if matchedMachineIdx == -1 {
		ctxlog.Info("Falling back to MAC address matching")

		// Get VMware credentials to fetch ESXi summary
		vmwarecreds, err := GetVMwareCredsFromRollingMigrationPlan(ctx, scope.Client, scope.RollingMigrationPlan)
		if err != nil {
			return false, "", errors.Wrap(err, "failed to get vmware credentials")
		}

		// Get ESXi summary to extract MAC addresses
		hs, err := GetESXiSummary(ctx, scope.Client, vmwarehost.Spec.Name, vmwarecreds, "")
		if err != nil {
			return false, "", errors.Wrap(err, "failed to get ESXi summary")
		}

		// Extract MAC addresses from ESXi host's physical network adapters
		var hostMacAddresses []string
		if hs.Config != nil && hs.Config.Network != nil && hs.Config.Network.Pnic != nil && len(hs.Config.Network.Pnic) > 0 {
			ctxlog.Info("Extracting MAC addresses from ESXi physical NICs", "pnicCount", len(hs.Config.Network.Pnic))
			for idx, pnic := range hs.Config.Network.Pnic {
				if pnic.Mac != "" {
					// Normalize MAC address to lowercase for comparison
					normalizedMac := strings.ToLower(pnic.Mac)
					hostMacAddresses = append(hostMacAddresses, normalizedMac)
					ctxlog.V(1).Info("Found ESXi NIC MAC address",
						"nicIndex", idx,
						"device", pnic.Device,
						"originalMac", pnic.Mac,
						"normalizedMac", normalizedMac)
				} else {
					ctxlog.V(1).Info("Skipping ESXi NIC with empty MAC", "nicIndex", idx, "device", pnic.Device)
				}
			}
		} else {
			ctxlog.Info("ESXi host has no physical NIC configuration available")
		}

		if len(hostMacAddresses) == 0 {
			return false, "", errors.New("no hardware UUID or MAC addresses found for matching")
		}

		ctxlog.Info("ESXi host MAC addresses extracted", "count", len(hostMacAddresses), "macs", hostMacAddresses)

		// Match machines based on MAC addresses (many-to-many)
		ctxlog.Info("Starting MAC address matching against MAAS machines", "machineCount", len(machines))
		for i := range machines {
			if machines[i].MacAddress == "" {
				ctxlog.V(1).Info("Skipping MAAS machine with empty MAC address",
					"index", i,
					"machineName", machines[i].Hostname,
					"machineId", machines[i].Id)
				continue
			}

			machineMac := strings.ToLower(machines[i].MacAddress)
			ctxlog.V(1).Info("Checking MAAS machine MAC",
				"index", i,
				"machineName", machines[i].Hostname,
				"machineId", machines[i].Id,
				"originalMac", machines[i].MacAddress,
				"normalizedMac", machineMac)

			// Check if any host MAC address matches the machine MAC address
			matched := false
			for hostMacIdx, hostMac := range hostMacAddresses {
				if hostMac != "" && hostMac == machineMac {
					ctxlog.Info("MAC address match found!",
						"esxiMac", hostMac,
						"esxiMacIndex", hostMacIdx,
						"machineMac", machineMac,
						"machineName", machines[i].Hostname,
						"machineId", machines[i].Id)
					matched = true
					break
				}
			}

			if matched {
				ctxlog.Info("Found a matching machine by MAC address",
					"machineMAC", machineMac,
					"name", machines[i].Hostname,
					"id", machines[i].Id,
					"hardwareUuid", machines[i].HardwareUuid,
					"status", machines[i].Status)
				matchedMachineIdx = i
				break
			}
		}

		if matchedMachineIdx == -1 {
			ctxlog.Info("No MAC address match found after checking all machines",
				"esxiMacs", hostMacAddresses,
				"machinesChecked", len(machines))
		}
	}

	// If a match was found (either by UUID or MAC), check the status
	if matchedMachineIdx != -1 {
		ctxlog.Info("Checking matched machine status",
			"machineName", machines[matchedMachineIdx].Hostname,
			"machineId", machines[matchedMachineIdx].Id,
			"status", machines[matchedMachineIdx].Status,
			"requiredStatus", "Deployed or Allocated")
		if machines[matchedMachineIdx].Status == "Deployed" || machines[matchedMachineIdx].Status == "Allocated" {
			ctxlog.Info("ESXi host successfully validated in MAAS",
				"esxiName", vmwarehost.Spec.Name,
				"machineName", machines[matchedMachineIdx].Hostname,
				"status", machines[matchedMachineIdx].Status)
			return true, "", nil
		}
		ctxlog.Info("ESXi host found in MAAS but status is invalid",
			"esxiName", vmwarehost.Spec.Name,
			"machineName", machines[matchedMachineIdx].Hostname,
			"currentStatus", machines[matchedMachineIdx].Status,
			"requiredStatus", "Deployed or Allocated")
		return false, fmt.Sprintf("ESXi %s is not in Deployed or Allocated state", vmwarehost.Spec.Name), nil
	}

	ctxlog.Info("ESXi host not found in MAAS after all matching attempts",
		"esxiName", vmwarehost.Spec.Name,
		"esxiHardwareUUID", vmwarehost.Spec.HardwareUUID)
	return false, fmt.Sprintf("ESXi %s is not in MAAS", vmwarehost.Spec.Name), nil
}

// EnsurePCDHasClusterConfigured verifies that at least one cluster is configured in PCD for the OpenStack credentials
func EnsurePCDHasClusterConfigured(ctx context.Context, scope *scope.RollingMigrationPlanScope) (bool, string, error) {
	// Get openstackcreds
	openstackCreds, err := GetOpenstackCredsForRollingMigrationPlan(ctx, scope.Client, scope.RollingMigrationPlan)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to get openstack creds for rolling migration plan")
	}
	// List PCD clusters
	clusters, err := filterPCDClustersOnOpenstackCreds(ctx, scope.Client, *openstackCreds)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to list PCD clusters")
	}
	if len(clusters) == 0 {
		return false, fmt.Sprintf("no PCD clusters configured for openstack creds %s", openstackCreds.Name), nil
	}
	return true, "", nil
}
