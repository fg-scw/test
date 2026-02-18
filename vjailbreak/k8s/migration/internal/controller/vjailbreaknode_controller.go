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
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/pkg/errors"
	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/constants"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/scope"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/utils"
)

// VjailbreakNodeReconciler reconciles a VjailbreakNode object
type VjailbreakNodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Local  bool
}

// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=vjailbreaknodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=vjailbreaknodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=vjailbreaknodes/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;delete

// Reconcile handles the reconciliation of VjailbreakNode resources
func (r *VjailbreakNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := log.FromContext(ctx).WithName(constants.VjailbreakNodeControllerName)

	// Fetch the VjailbreakNode instance.
	vjailbreakNode := vjailbreakv1alpha1.VjailbreakNode{}
	client := r.Client
	err := client.Get(ctx, req.NamespacedName, &vjailbreakNode)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	vjailbreakNodeScope, err := scope.NewVjailbreakNodeScope(scope.VjailbreakNodeScopeParams{
		Logger:         log,
		Client:         r.Client,
		VjailbreakNode: &vjailbreakNode,
	})
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to create vjailbreak node scope")
	}

	// Always close the scope when exiting this function such that we can persist any VjailbreakNode changes.
	defer func() {
		if err := vjailbreakNodeScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Handle deleted VjailbreakNode first (before checking phase)
	if !vjailbreakNode.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, vjailbreakNodeScope)
	}

	// Quick path for just updating ActiveMigrations if node is ready
	if vjailbreakNode.Status.Phase == constants.VjailbreakNodePhaseNodeReady {
		return r.updateActiveMigrations(ctx, vjailbreakNodeScope)
	}

	// Handle regular VjailbreakNode reconcile
	return r.reconcileNormal(ctx, vjailbreakNodeScope)
}

// reconcileNormal handles regular VjailbreakNode reconcile
//
//nolint:unparam //future use
func (r *VjailbreakNodeReconciler) reconcileNormal(ctx context.Context,
	scope *scope.VjailbreakNodeScope) (ctrl.Result, error) {
	log := scope.Logger
	log.Info("Reconciling VjailbreakNode")

	vjNode := scope.VjailbreakNode
	controllerutil.AddFinalizer(vjNode, constants.VjailbreakNodeFinalizer)

	if vjNode.Spec.NodeRole == constants.NodeRoleMaster {
		err := utils.UpdateMasterNodeImageID(ctx, r.Client, r.Local)
		if err != nil {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, errors.Wrap(err, "failed to update master node image id")
		}
		log.Info("Skipping master node, updating flavor", "name", vjNode.Name)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// Skip reconciliation if node is in error state - wait for manual intervention or deletion
	if vjNode.Status.Phase == constants.VjailbreakNodePhaseError {
		log.Info("Node is in error state, skipping reconciliation. Delete the node to clean up.", "name", vjNode.Name)
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	uuid, err := utils.GetOpenstackVMByName(ctx, vjNode.Name, r.Client, vjNode)
	if err != nil {
		log.Error(err, "Failed to get OpenStack VM by name, setting node to error state")
		vjNode.Status.Phase = constants.VjailbreakNodePhaseError
		if updateErr := r.Client.Status().Update(ctx, vjNode); updateErr != nil {
			log.Error(updateErr, "Failed to update node status to error")
		}
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, errors.Wrap(err, "failed to get openstack vm by name")
	}

	if uuid != "" {
		// VM exists, check if we need to update UUID and/or IP
		if vjNode.Status.OpenstackUUID == "" || vjNode.Status.VMIP == "" {
			ipSet, err := utils.ReconcileVMStatusAndIP(ctx, r.Client, vjNode, uuid)
			if err != nil {
				log.Error(err, "Failed to reconcile VM status and IP")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, err
			}
			if !ipSet {
				// IP not yet available or VM still building, retry
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
			// IP was set, requeue quickly to check K8s node status
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		// VM and UUID already set, check Kubernetes node status
		nodeReady, err := utils.ReconcileK8sNodeStatus(ctx, r.Client, vjNode)
		if err != nil {
			log.Error(err, "Failed to reconcile K8s node status")
			return ctrl.Result{}, err
		}
		if nodeReady {
			// Node is ready - no need to reconcile until something changes
			return ctrl.Result{}, nil
		}
		// Node not ready yet - check again soon
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// VM doesn't exist yet, create it
	vjNode.Status.Phase = constants.VjailbreakNodePhaseVMCreating
	if updateErr := r.Client.Status().Update(ctx, vjNode); updateErr != nil {
		log.Error(updateErr, "Failed to update node status to CreatingVM")
	}

	// Create Openstack VM for worker node
	vmid, err := utils.CreateOpenstackVMForWorkerNode(ctx, r.Client, scope)
	if err != nil {
		log.Error(err, "Failed to create OpenStack VM for worker node, setting node to error state")
		vjNode.Status.Phase = constants.VjailbreakNodePhaseError
		if updateErr := r.Client.Status().Update(ctx, vjNode); updateErr != nil {
			log.Error(updateErr, "Failed to update node status to error")
		}
		return ctrl.Result{RequeueAfter: 2 * time.Minute}, errors.Wrap(err, "failed to create openstack vm for worker node")
	}

	vjNode.Status.OpenstackUUID = vmid
	vjNode.Status.Phase = constants.VjailbreakNodePhaseVMCreated
	// Note: VMIP will be populated on next reconciliation when we can query it from OpenStack

	// Update the VjailbreakNode status
	err = r.Client.Status().Update(ctx, vjNode)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to update vjailbreak node status")
	}

	log.Info("Successfully created openstack vm for worker node", "vmid", vmid)
	// Requeue to fetch the VM IP (give VM time to get IP assigned)
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// reconcileDelete handles deleted VjailbreakNode
//
//nolint:unparam //future use
func (r *VjailbreakNodeReconciler) reconcileDelete(ctx context.Context,
	scope *scope.VjailbreakNodeScope) (ctrl.Result, error) {
	log := scope.Logger
	log.Info("Reconciling VjailbreakNode Delete")

	if scope.VjailbreakNode.Spec.NodeRole == constants.NodeRoleMaster {
		controllerutil.RemoveFinalizer(scope.VjailbreakNode, constants.VjailbreakNodeFinalizer)
		return ctrl.Result{}, nil
	}

	scope.VjailbreakNode.Status.Phase = constants.VjailbreakNodePhaseDeleting
	// Update the VjailbreakNode status
	err := r.Client.Status().Update(ctx, scope.VjailbreakNode)
	if err != nil {
		log.Error(err, "Failed to update node status to deleting, continuing with deletion")
	}

	// Try to get the VM UUID - use stored UUID if lookup fails
	uuid, err := utils.GetOpenstackVMByName(ctx, scope.VjailbreakNode.Name, r.Client, scope.VjailbreakNode)
	if err != nil {
		log.Info("Failed to lookup VM by name, using stored UUID if available", "error", err)
		uuid = scope.VjailbreakNode.Status.OpenstackUUID
	}

	// If we have a UUID (either from lookup or stored), try to delete the VM
	if uuid != "" {
		err = utils.DeleteOpenstackVM(ctx, uuid, r.Client, scope.VjailbreakNode)
		if err != nil {
			log.Error(err, "Failed to delete OpenStack VM, continuing with cleanup", "uuid", uuid)
			// Don't return error - continue with cleanup even if VM deletion fails
		} else {
			log.Info("Successfully deleted OpenStack VM", "uuid", uuid)
		}
	} else {
		log.Info("No VM UUID found, VM may have been manually deleted or never created")
	}

	// Try to delete the Kubernetes node
	if scope.VjailbreakNode.Name != "" {
		err = utils.DeleteNodeByName(ctx, r.Client, scope.VjailbreakNode.Name)
		if err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to delete Kubernetes node, continuing with finalizer removal", "nodeName", scope.VjailbreakNode.Name)
			// Don't return error - allow finalizer removal even if K8s node deletion fails
		} else if err == nil {
			log.Info("Successfully deleted Kubernetes node", "nodeName", scope.VjailbreakNode.Name)
		}
	} else {
		log.Info("VjailbreakNode name is empty, skipping Kubernetes node deletion")
	}

	// Always remove finalizer to allow the resource to be deleted
	// This ensures nodes in error states can be cleaned up
	controllerutil.RemoveFinalizer(scope.VjailbreakNode, constants.VjailbreakNodeFinalizer)
	log.Info("Finalizer removed, VjailbreakNode will be deleted", "name", scope.VjailbreakNode.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VjailbreakNodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vjailbreakv1alpha1.VjailbreakNode{}).
		Complete(r)
}

// updateActiveMigrations efficiently updates just the ActiveMigrations field
func (r *VjailbreakNodeReconciler) updateActiveMigrations(ctx context.Context,
	scope *scope.VjailbreakNodeScope) (ctrl.Result, error) {
	vjNode := scope.VjailbreakNode

	// Get active migrations happening on the node
	activeMigrations, err := utils.GetActiveMigrations(ctx, vjNode.Name, r.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to get active migrations")
	}
	// Create a patch to update only the ActiveMigrations field
	patch := client.MergeFrom(vjNode.DeepCopy())
	vjNode.Status.ActiveMigrations = activeMigrations

	err = r.Client.Status().Patch(ctx, vjNode, patch)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to patch vjailbreak node status")
	}

	// Only requeue if there are active migrations - otherwise wait for changes
	if len(activeMigrations) > 0 {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}
