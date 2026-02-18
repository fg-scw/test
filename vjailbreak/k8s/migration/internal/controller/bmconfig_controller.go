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
	"time"

	"github.com/pkg/errors"
	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	constants "github.com/platform9/vjailbreak/k8s/migration/pkg/constants"
	scope "github.com/platform9/vjailbreak/k8s/migration/pkg/scope"
	providers "github.com/platform9/vjailbreak/pkg/vpwned/sdk/providers"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// BMConfigReconciler reconciles a BMConfig object
type BMConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=bmconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=bmconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vjailbreak.k8s.pf9.io,resources=bmconfigs/finalizers,verbs=update

// Reconcile processes a BMConfig resource and reconciles the desired state with the actual state.
// It handles both normal reconciliation and deletion reconciliation.
func (r *BMConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctxlog := log.FromContext(ctx).WithName(constants.BMConfigControllerName)
	ctxlog.Info(fmt.Sprintf("Reconciling BMConfig '%s'", req.Name))

	bmConfig := &vjailbreakv1alpha1.BMConfig{}
	if err := r.Get(ctx, req.NamespacedName, bmConfig); err != nil {
		if apierrors.IsNotFound(err) {
			ctxlog.Info("Received ignorable event for a recently deleted bmconfig.")
			return ctrl.Result{}, nil
		}
		ctxlog.Error(err, fmt.Sprintf("Unexpected error reading bmconfig '%s' object", bmConfig.Name))
		return ctrl.Result{}, err
	}

	scope, err := scope.NewBMConfigScope(scope.BMConfigScopeParams{
		Logger:   ctxlog,
		Client:   r.Client,
		BMConfig: bmConfig,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	// Always close the scope when exiting this function such that we can persist any BMConfig changes.
	defer func() {
		if err := scope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	if !bmConfig.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, scope)
	}

	return r.reconcileNormal(ctx, scope)
}

// nolint:unparam
func (r *BMConfigReconciler) reconcileDelete(_ context.Context, scope *scope.BMConfigScope) (ctrl.Result, error) {
	bmConfig := scope.BMConfig
	controllerutil.RemoveFinalizer(bmConfig, constants.BMConfigFinalizer)

	return ctrl.Result{}, nil
}

func (r *BMConfigReconciler) reconcileNormal(ctx context.Context, scope *scope.BMConfigScope) (ctrl.Result, error) {
	bmConfig := scope.BMConfig
	controllerutil.AddFinalizer(bmConfig, constants.BMConfigFinalizer)

	provider, err := providers.GetProvider(string(bmConfig.Spec.ProviderType))
	if err != nil {
		return ctrl.Result{}, err
	}

	err = provider.Connect(providers.BMAccessInfo{
		Username:    bmConfig.Spec.UserName,
		Password:    bmConfig.Spec.Password,
		APIKey:      bmConfig.Spec.APIKey,
		BaseURL:     bmConfig.Spec.APIUrl,
		UseInsecure: bmConfig.Spec.Insecure,
	})
	if err != nil {
		bmConfig.Status.ValidationStatus = string(corev1.PodFailed)
		bmConfig.Status.ValidationMessage = fmt.Sprintf("Error connecting to MAAS: %s", err)
		if updateErr := r.Status().Update(ctx, bmConfig); updateErr != nil {
			return ctrl.Result{}, errors.Wrap(
				errors.Wrap(updateErr, fmt.Sprintf("Error updating status of BMConfig '%s'", bmConfig.Name)),
				err.Error())
		}
		return ctrl.Result{RequeueAfter: constants.CredsRequeueAfter}, err
	}
	defer func() {
		if err := provider.Disconnect(); err != nil {
			scope.Error(err, "Error disconnecting from MAAS provider")
		}
	}()

	bmConfig.Status.ValidationStatus = string(corev1.PodSucceeded)
	bmConfig.Status.ValidationMessage = "Successfully connected to MAAS"
	if updateErr := r.Status().Update(ctx, bmConfig); updateErr != nil {
		return ctrl.Result{}, errors.Wrap(
			updateErr, fmt.Sprintf("Error updating status of BMConfig '%s'", bmConfig.Name))
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BMConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vjailbreakv1alpha1.BMConfig{}).
		Complete(r)
}
