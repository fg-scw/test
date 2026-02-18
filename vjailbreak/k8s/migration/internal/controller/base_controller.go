// Package controller provides controllers for managing migrations and related resources
package controller

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// BaseReconciler provides common functionality for mapping controllers
type BaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// ReconcileMapping performs common reconciliation logic for mapping resources
func (r *BaseReconciler) ReconcileMapping(ctx context.Context, obj client.Object, updateStatus func() error) (ctrl.Result, error) {
	ctxlog := log.FromContext(ctx)

	if err := r.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if apierrors.IsNotFound(err) {
			ctxlog.Info("Received ignorable event for a recently deleted resource.")
			return ctrl.Result{}, nil
		}
		ctxlog.Error(err, fmt.Sprintf("Unexpected error reading resource '%s' object", obj.GetName()))
		return ctrl.Result{}, err
	}

	if obj.GetDeletionTimestamp().IsZero() {
		if err := updateStatus(); err != nil {
			ctxlog.Error(err, fmt.Sprintf("Failed to update resource '%s' object", obj.GetName()))
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
