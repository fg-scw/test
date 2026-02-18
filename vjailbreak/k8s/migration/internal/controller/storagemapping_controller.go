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

// Package controller provides controllers for managing migrations and related resources
package controller

import (
	"context"
	"fmt"

	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// StorageMappingReconciler reconciles a StorageMapping object
type StorageMappingReconciler struct {
	BaseReconciler
}

// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=storagemappings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=storagemappings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=storagemappings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *StorageMappingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctxlog := log.FromContext(ctx)
	storagemapping := &vjailbreakv1alpha1.StorageMapping{}
	storagemapping.Name = req.Name
	storagemapping.Namespace = req.Namespace

	if err := r.Get(ctx, req.NamespacedName, storagemapping); err != nil {
		if apierrors.IsNotFound(err) {
			ctxlog.Info("Received ignorable event for a recently deleted storagemapping.")
			return ctrl.Result{}, nil
		}
		ctxlog.Error(err, fmt.Sprintf("Unexpected error reading storagemapping '%s' object", storagemapping.Name))
		return ctrl.Result{}, err
	}
	if storagemapping.DeletionTimestamp.IsZero() {
		ctxlog.Info(fmt.Sprintf("Reconciling storagemapping '%s'", storagemapping.Name))
		return r.ReconcileMapping(ctx, storagemapping, func() error {
			storagemapping.Status = vjailbreakv1alpha1.StorageMappingStatus{}
			return r.Status().Update(ctx, storagemapping)
		})
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StorageMappingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	return ctrl.NewControllerManagedBy(mgr).
		For(&vjailbreakv1alpha1.StorageMapping{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
