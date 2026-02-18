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

	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// NetworkMappingReconciler reconciles a NetworkMapping object
type NetworkMappingReconciler struct {
	BaseReconciler
}

// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=networkmappings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=networkmappings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=networkmappings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NetworkMappingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	networkmapping := &vjailbreakv1alpha1.NetworkMapping{}
	networkmapping.Name = req.Name
	networkmapping.Namespace = req.Namespace

	return r.ReconcileMapping(ctx, networkmapping, func() error {
		networkmapping.Status = vjailbreakv1alpha1.NetworkMappingStatus{}
		return r.Status().Update(ctx, networkmapping)
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkMappingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	return ctrl.NewControllerManagedBy(mgr).
		For(&vjailbreakv1alpha1.NetworkMapping{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
