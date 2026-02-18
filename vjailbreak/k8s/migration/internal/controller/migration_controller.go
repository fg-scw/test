/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	openstackconst "github.com/platform9/vjailbreak/v2v-helper/pkg/constants"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/pkg/errors"
	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	constants "github.com/platform9/vjailbreak/k8s/migration/pkg/constants"
	migrationmetrics "github.com/platform9/vjailbreak/k8s/migration/pkg/metrics"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/scope"
	utils "github.com/platform9/vjailbreak/k8s/migration/pkg/utils"
)

// MigrationReconciler reconciles a Migration object
type MigrationReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	MaxConcurrentReconciles int
}

const migrationFinalizer = "migration.vjailbreak.k8s.pf9.io/finalizer"

// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch
// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=migrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=migrations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=vmwaremachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=vmwaremachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=bmconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=bmconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=bmconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces;serviceaccounts;services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch;create;update;patch;delete

// Reconcile reconciles a Migration object
// nolint:gocyclo // Reconcile function complexity is inherent to state machine logic
func (r *MigrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctxlog := log.FromContext(ctx).WithName(constants.MigrationControllerName)

	migration := &vjailbreakv1alpha1.Migration{}

	if err := r.Get(ctx, req.NamespacedName, migration); err != nil {
		if apierrors.IsNotFound(err) {
			// Object deleted successfully
			return ctrl.Result{}, nil
		}
		ctxlog.Error(err, fmt.Sprintf("Unexpected error reading Migration '%s' object", migration.Name))
		return ctrl.Result{}, err
	}

	// Handle deletion reconciliation first, even for ValidationFailed migrations.
	if !migration.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(migration, migrationFinalizer) {
			if err := r.reconcileDelete(ctx, migration); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(migration, migrationFinalizer)
			if err := r.Update(ctx, migration); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if migration.Status.Phase == vjailbreakv1alpha1.VMMigrationPhaseValidationFailed {
		ctxlog.Info(
			"Migration is ValidationFailed; skipping reconciliation and requeue",
			"migration", migration.Name,
		)
		return ctrl.Result{}, nil
	}

	// If migration is already marked as Failed and no pod was created, don't keep requeuing
	if migration.Status.Phase == vjailbreakv1alpha1.VMMigrationPhaseFailed {
		ctxlog.Info(
			"Migration is Failed; skipping reconciliation and requeue",
			"migration", migration.Name,
		)
		return ctrl.Result{}, nil
	}

	oldStatus := migration.Status.DeepCopy()

	migrationScope, err := scope.NewMigrationScope(scope.MigrationScopeParams{
		Logger:    ctxlog,
		Client:    r.Client,
		Migration: migration,
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create scope: %w", err)
	}

	defer func() {
		if err := migrationScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Adding finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(migration, migrationFinalizer) {
		controllerutil.AddFinalizer(migration, migrationFinalizer)
		if err := r.Update(ctx, migration); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	ctxlog.Info("Reconciling Migration object")

	// Get the pod phase
	pod, err := r.GetPod(ctx, migrationScope)
	if err != nil {
		if apierrors.IsNotFound(err) {
			ctxlog.Info("Migration pod not found yet, requeuing", "migration", migration.Name)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	if migration.Spec.PodRef != pod.Name {
		ctxlog.Info("Updating migration spec podref", "migration", migration.Name, "podRef", pod.Name)
		migration.Spec.PodRef = pod.Name
		if err := r.Update(ctx, migration); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}

	oldCutoverLabel := pod.Labels["startCutover"]
	newCutoverLabel := utils.SetCutoverLabel(migration.Spec.InitiateCutover, oldCutoverLabel)
	if newCutoverLabel != oldCutoverLabel {
		pod.Labels["startCutover"] = newCutoverLabel
		if err = r.Update(ctx, pod); err != nil {
			ctxlog.Error(err, fmt.Sprintf("Failed to update Pod '%s'", pod.Name))
			return ctrl.Result{}, err
		}
	}

	if constants.VMMigrationStatesEnum[migration.Status.Phase] <= constants.VMMigrationStatesEnum[vjailbreakv1alpha1.VMMigrationPhaseValidating] {
		migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseValidating
	}
	// Check if the pod is in a valid state only then continue
	if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodFailed && pod.Status.Phase != corev1.PodSucceeded {
		ctxlog.Info("Pod is not in a terminal state, requeuing", "migration", migration.Name, "podStatus", pod.Status.Phase)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	filteredEvents, err := r.GetEventsSorted(ctx, migrationScope)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed getting pod events")
	}
	// Create status conditions
	migration.Status.Conditions = utils.CreateValidatedCondition(migration, filteredEvents)
	migration.Status.Conditions = utils.CreateStorageAcceleratedCopyCondition(migration, filteredEvents)
	migration.Status.Conditions = utils.CreateDataCopyCondition(migration, filteredEvents)
	migration.Status.Conditions = utils.CreateMigratingCondition(migration, filteredEvents)
	migration.Status.Conditions = utils.CreateFailedCondition(migration, filteredEvents)
	migration.Status.Conditions = utils.CreateSucceededCondition(migration, filteredEvents)

	migration.Status.AgentName = pod.Spec.NodeName

	// Extract current disk being copied from events
	r.ExtractCurrentDisk(migration, filteredEvents)

	if migration.Status.TotalDisks == 0 {
		if v, ok := migration.Labels[constants.NumberOfDisksLabel]; ok {
			if n, err := strconv.Atoi(v); err == nil {
				migration.Status.TotalDisks = n
			} else {
				log.FromContext(ctx).Error(err, "Failed to parse total disks value",
					"label", constants.NumberOfDisksLabel,
					"value", v)
			}
		}
	}

	err = r.SetupMigrationPhase(ctx, migrationScope)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "error setting migration phase")
	}

	// Record migration start if this is a new migration (hasn't been started in metrics yet)
	// Check if migration was just created (within last minute) and oldStatus phase is empty or Pending
	isNewMigration := time.Since(migration.CreationTimestamp.Time) < time.Minute &&
		(oldStatus.Phase == "" || oldStatus.Phase == vjailbreakv1alpha1.VMMigrationPhasePending)

	if isNewMigration && oldStatus.Phase == "" {
		migrationmetrics.RecordMigrationStarted(
			migration.Name,
			migration.Spec.VMName,
			migration.Namespace,
			migration.Spec.MigrationPlan,
			migration.Status.AgentName,
		)
	}

	// Update phase metrics on every reconcile for active migrations
	if migration.Status.Phase != "" {
		migrationmetrics.UpdateMigrationPhase(migration.Name, migration.Spec.VMName, migration.Namespace, migration.Status.AgentName, migration.Status.Phase)
	}

	// Update duration for active migrations
	if migration.Status.Phase != vjailbreakv1alpha1.VMMigrationPhaseSucceeded &&
		migration.Status.Phase != vjailbreakv1alpha1.VMMigrationPhaseFailed &&
		migration.Status.Phase != vjailbreakv1alpha1.VMMigrationPhaseValidationFailed &&
		migration.Status.Phase != "" {
		migrationmetrics.RecordMigrationProgress(migration.Name, migration.Spec.VMName, migration.Namespace, migration.CreationTimestamp.Time)
	}

	// Record completion when transitioning to a terminal state from a non-terminal state
	isNowTerminal := migration.Status.Phase == vjailbreakv1alpha1.VMMigrationPhaseSucceeded ||
		migration.Status.Phase == vjailbreakv1alpha1.VMMigrationPhaseFailed
	wasTerminal := oldStatus.Phase == vjailbreakv1alpha1.VMMigrationPhaseSucceeded ||
		oldStatus.Phase == vjailbreakv1alpha1.VMMigrationPhaseFailed

	if isNowTerminal && !wasTerminal {
		migrationmetrics.RecordMigrationCompleted(migration.Name, migration.Spec.VMName, migration.Namespace, migration.Status.AgentName, migration.Status.Phase)
	}

	if !reflect.DeepEqual(&migration.Status, oldStatus) {
		if err := r.Status().Update(ctx, migration); err != nil {
			ctxlog.Error(err, fmt.Sprintf("Failed to update status of Migration '%s'", migration.Name))
			return ctrl.Result{}, err
		}
	}

	if string(migration.Status.Phase) != string(vjailbreakv1alpha1.VMMigrationPhaseFailed) &&
		string(migration.Status.Phase) != string(vjailbreakv1alpha1.VMMigrationPhaseValidationFailed) &&
		string(migration.Status.Phase) != string(vjailbreakv1alpha1.VMMigrationPhaseSucceeded) {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// reconcileDelete handles the cleanup logic when Migration object is deleted.
func (r *MigrationReconciler) reconcileDelete(ctx context.Context, migration *vjailbreakv1alpha1.Migration) error {
	ctxlog := log.FromContext(ctx).WithName(constants.MigrationControllerName)
	ctxlog.Info("Reconciling deletion of Migration, resetting VMwareMachine status", "MigrationName", migration.Name)

	if migration.Spec.VMName == "" {
		ctxlog.Info("VMName is empty in Migration spec, nothing to do.")
		return nil
	}

	// Clean up metrics for this migration
	migrationmetrics.CleanupMigrationMetrics(migration.Name, migration.Spec.VMName, migration.Namespace, migration.Spec.MigrationPlan, migration.Status.AgentName)

	vmwareCredsName, err := utils.GetVMwareCredsNameFromMigration(ctx, r.Client, migration)
	if err != nil {
		ctxlog.Error(err, "Failed to get VMware credentials name for migration")
		return nil
	}

	// Then use it to get the k8s compatible name
	vmwMachineName, err := utils.GetK8sCompatibleVMWareObjectName(migration.Spec.VMName, vmwareCredsName)
	if err != nil {
		ctxlog.Error(err, "Could not determine VMwareMachine name from VM name", "VMName", migration.Spec.VMName)
		return nil
	}

	vmwMachine := &vjailbreakv1alpha1.VMwareMachine{}
	err = r.Get(ctx, types.NamespacedName{Name: vmwMachineName, Namespace: migration.GetNamespace()}, vmwMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			ctxlog.Info("VMwareMachine not found during migration deletion, nothing to do.", "VMwareMachineName", vmwMachineName)
			return nil
		}
		return errors.Wrap(err, "failed to get VMwareMachine for cleanup")
	}

	if vmwMachine.Status.Migrated {
		ctxlog.Info("Setting VMwareMachine status.migrated to false", "VMwareMachineName", vmwMachineName)
		vmwMachine.Status.Migrated = false
		if err := r.Status().Update(ctx, vmwMachine); err != nil {
			return errors.Wrap(err, "failed to update VMwareMachine status")
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MigrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vjailbreakv1alpha1.Migration{}).
		Owns(&corev1.Pod{}, builder.WithPredicates(
			predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldpod, ok := e.ObjectOld.(*corev1.Pod)
					if !ok {
						return false
					}
					newpod, ok := e.ObjectNew.(*corev1.Pod)
					if !ok {
						return false
					}
					for _, condition := range newpod.Status.Conditions {
						// Ignores the disk percentage updates in the pod custom conditions
						if condition.Type == "Progressing" && !strings.Contains(condition.Message, "Progress:") {
							return true
						}
					}
					return oldpod.Status.Phase != newpod.Status.Phase
				},
			},
		)).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.MaxConcurrentReconciles}).
		Complete(r)
}

// SetupMigrationPhase sets up the migration phase based on current state
//
//nolint:gocyclo
func (r *MigrationReconciler) SetupMigrationPhase(ctx context.Context, scope *scope.MigrationScope) error {
	events, err := r.GetEventsSorted(ctx, scope)
	if err != nil {
		return err
	}

	// Get the pod to check startCutover label
	pod, err := r.GetPod(ctx, scope)
	if err != nil {
		return err
	}

	IgnoredPhases := []vjailbreakv1alpha1.VMMigrationPhase{
		vjailbreakv1alpha1.VMMigrationPhaseValidating,
		vjailbreakv1alpha1.VMMigrationPhasePending,
		vjailbreakv1alpha1.VMMigrationPhaseValidationFailed}

loop:
	for i := range events.Items {
		switch {
		// In reverse order, because the events are sorted by timestamp latest to oldest
		case strings.Contains(events.Items[i].Message, openstackconst.EventMessageMigrationSucessful) &&
			constants.VMMigrationStatesEnum[scope.Migration.Status.Phase] <= constants.VMMigrationStatesEnum[vjailbreakv1alpha1.VMMigrationPhaseSucceeded]:
			scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseSucceeded
			if err := r.markMigrationSuccessful(ctx, scope); err != nil {
				return err
			}
			break loop
		case strings.Contains(events.Items[i].Message, openstackconst.EventMessageWaitingForAdminCutOver) &&
			constants.VMMigrationStatesEnum[scope.Migration.Status.Phase] <= constants.VMMigrationStatesEnum[vjailbreakv1alpha1.VMMigrationPhaseAwaitingAdminCutOver]:
			// Only stay in AwaitingAdminCutOver if cutover hasn't been triggered yet
			if pod.Labels["startCutover"] != "yes" {
				scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseAwaitingAdminCutOver
				break loop
			}
			// Admin cutover was triggered, reset to a lower phase so it can progress normally
			scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseCopying

			// If startCutover is "yes", don't set phase here - let it progress to next phase
			// by continuing to check other events
			continue
		case strings.Contains(events.Items[i].Message, openstackconst.EventMessageWaitingForCutOverStart) &&
			constants.VMMigrationStatesEnum[scope.Migration.Status.Phase] <= constants.VMMigrationStatesEnum[vjailbreakv1alpha1.VMMigrationPhaseAwaitingCutOverStartTime]:
			scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseAwaitingCutOverStartTime
			break loop
		case strings.Contains(events.Items[i].Message, openstackconst.EventMessageConvertingDisk) &&
			constants.VMMigrationStatesEnum[scope.Migration.Status.Phase] <= constants.VMMigrationStatesEnum[vjailbreakv1alpha1.VMMigrationPhaseConvertingDisk]:
			scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseConvertingDisk
			break loop
		case strings.Contains(events.Items[i].Message, openstackconst.EventMessageCopyingChangedBlocksWithIteration) &&
			constants.VMMigrationStatesEnum[scope.Migration.Status.Phase] <= constants.VMMigrationStatesEnum[vjailbreakv1alpha1.VMMigrationPhaseCopyingChangedBlocks]:
			scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseCopyingChangedBlocks
			break loop
		case strings.Contains(events.Items[i].Message, openstackconst.EventMessageCopyingDisk) &&
			constants.VMMigrationStatesEnum[scope.Migration.Status.Phase] <= constants.VMMigrationStatesEnum[vjailbreakv1alpha1.VMMigrationPhaseCopying]:
			scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseCopying
			break loop
		case strings.Contains(events.Items[i].Message, openstackconst.EventMessageStorageAcceleratedCopyRescanStorage) &&
			constants.VMMigrationStatesEnum[scope.Migration.Status.Phase] <= constants.VMMigrationStatesEnum[vjailbreakv1alpha1.VMMigrationPhaseRescanningStorage]:
			scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseRescanningStorage
			break loop
		case strings.Contains(events.Items[i].Message, openstackconst.EventMessageStorageAcceleratedCopyMappingVolume) &&
			constants.VMMigrationStatesEnum[scope.Migration.Status.Phase] <= constants.VMMigrationStatesEnum[vjailbreakv1alpha1.VMMigrationPhaseMappingVolume]:
			scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseMappingVolume
			break loop
		case strings.Contains(events.Items[i].Message, openstackconst.EventMessageStorageAcceleratedCopyCinderManage) &&
			constants.VMMigrationStatesEnum[scope.Migration.Status.Phase] <= constants.VMMigrationStatesEnum[vjailbreakv1alpha1.VMMigrationPhaseImportingToCinder]:
			scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseImportingToCinder
			break loop
		case strings.Contains(events.Items[i].Message, openstackconst.EventMessageStorageAcceleratedCopyCreatingVolume) &&
			constants.VMMigrationStatesEnum[scope.Migration.Status.Phase] <= constants.VMMigrationStatesEnum[vjailbreakv1alpha1.VMMigrationPhaseCreatingVolume]:
			scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseCreatingVolume
			break loop
		case strings.Contains(events.Items[i].Message, openstackconst.EventMessageInitiatorGroup) &&
			constants.VMMigrationStatesEnum[scope.Migration.Status.Phase] <= constants.VMMigrationStatesEnum[vjailbreakv1alpha1.VMMigrationPhaseCreatingInitiatorGroup]:
			scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseCreatingInitiatorGroup
			break loop
		case strings.Contains(events.Items[i].Message, openstackconst.EventMessageEsxiSSHConnect) &&
			constants.VMMigrationStatesEnum[scope.Migration.Status.Phase] <= constants.VMMigrationStatesEnum[vjailbreakv1alpha1.VMMigrationPhaseConnectingToESXi]:
			scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseConnectingToESXi
			break loop
		case strings.Contains(events.Items[i].Message, openstackconst.EventMessageWaitingForDataCopyStart) &&
			constants.VMMigrationStatesEnum[scope.Migration.Status.Phase] <= constants.VMMigrationStatesEnum[vjailbreakv1alpha1.VMMigrationPhaseAwaitingDataCopyStart]:
			scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseAwaitingDataCopyStart
			break loop
		case strings.Contains(strings.TrimSpace(events.Items[i].Message), openstackconst.EventMessageMigrationFailed) ||
			strings.Contains(strings.TrimSpace(events.Items[i].Message), openstackconst.EventMessageFailed):
			scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseFailed
			break loop
		case slices.Contains(IgnoredPhases, scope.Migration.Status.Phase):
			break loop
		default:
			continue
		}
	}
	return nil
}

// Extracted function to handle successful migration updates
func (r *MigrationReconciler) markMigrationSuccessful(ctx context.Context, scope *scope.MigrationScope) error {
	scope.Migration.Status.Phase = vjailbreakv1alpha1.VMMigrationPhaseSucceeded
	vmwareCredsName, err := utils.GetVMwareCredsNameFromMigration(ctx, r.Client, scope.Migration)
	if err != nil {
		return errors.Wrap(err, "failed to get vmware credentials name")
	}
	name, err := utils.GetK8sCompatibleVMWareObjectName(scope.Migration.Spec.VMName, vmwareCredsName)
	if err != nil {
		return errors.Wrap(err, "failed to get vmware machine name")
	}

	vmwvm := &vjailbreakv1alpha1.VMwareMachine{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: scope.Migration.Namespace}, vmwvm); err != nil {
		return errors.Wrap(err, "failed to get vmware machine")
	}

	vmwvm.Status.Migrated = true
	return r.Status().Update(ctx, vmwvm)
}

// GetEventsSorted retrieves sorted events for a migration
func (r *MigrationReconciler) GetEventsSorted(ctx context.Context, scope *scope.MigrationScope) (*corev1.EventList, error) {
	migration := scope.Migration
	ctxlog := scope.Logger

	pod, err := r.GetPod(ctx, scope)
	if err != nil {
		return nil, err
	}

	// Get events of this pod
	allevents := &corev1.EventList{}
	if err := r.List(ctx, allevents, client.InNamespace(migration.Namespace)); err != nil {
		ctxlog.Error(err, fmt.Sprintf("Failed to get events for Pod '%s'", migration.Spec.PodRef))
		return nil, err
	}

	filteredEvents := &corev1.EventList{}
	for i := 0; i < len(allevents.Items); i++ {
		if allevents.Items[i].InvolvedObject.Name == migration.Spec.PodRef && string(allevents.Items[i].InvolvedObject.UID) == string(pod.UID) {
			filteredEvents.Items = append(filteredEvents.Items, allevents.Items[i])
		}
	}

	// Sort filteredEvents by creation timestamp
	sort.Slice(filteredEvents.Items, func(i, j int) bool {
		return !filteredEvents.Items[i].CreationTimestamp.Before(&filteredEvents.Items[j].CreationTimestamp)
	})
	return filteredEvents, nil
}

// GetPod retrieves the pod associated with a migration
func (r *MigrationReconciler) GetPod(ctx context.Context, scope *scope.MigrationScope) (*corev1.Pod, error) {
	migration := scope.Migration
	vmwareCredsName, err := utils.GetVMwareCredsNameFromMigration(ctx, r.Client, scope.Migration)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get vmware credentials name")
	}
	vmname, err := utils.GetK8sCompatibleVMWareObjectName(migration.Spec.VMName, vmwareCredsName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get vm name")
	}
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(migration.Namespace),
		client.MatchingLabels(map[string]string{constants.VMNameLabel: vmname})); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to get pod with label '%s=%s'", constants.VMNameLabel, vmname))
	}
	if len(podList.Items) == 0 {
		return nil, apierrors.NewNotFound(corev1.Resource("pods"), fmt.Sprintf("migration pod not found for vm %s", migration.Spec.VMName))
	}
	return &podList.Items[0], nil
}

// ExtractCurrentDisk extracts which disk is currently being copied from pod events
func (r *MigrationReconciler) ExtractCurrentDisk(migration *vjailbreakv1alpha1.Migration, events *corev1.EventList) {
	// Events are sorted by timestamp (newest first)
	parseCurrentDisk := func(msg string) (string, bool) {
		if !strings.Contains(msg, "Copying disk") {
			return "", false
		}
		parts := strings.Split(msg, "Copying disk")
		if len(parts) <= 1 {
			return "", false
		}
		diskPart := strings.TrimSpace(parts[1])
		if len(diskPart) == 0 {
			return "", false
		}
		diskNum := strings.Split(diskPart, ",")[0]
		diskNum = strings.Split(diskNum, " ")[0]
		diskNum = strings.TrimSpace(diskNum)
		if diskNum == "" {
			return "", false
		}
		return diskNum, true
	}

	for i := range events.Items {
		if diskNum, ok := parseCurrentDisk(events.Items[i].Message); ok {
			migration.Status.CurrentDisk = diskNum
			return
		}
	}

	for _, condition := range migration.Status.Conditions {
		if diskNum, ok := parseCurrentDisk(condition.Message); ok {
			migration.Status.CurrentDisk = diskNum
			return
		}
	}
}
