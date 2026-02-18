// Package scope provides scoped client functionality for controllers
package scope

import (
	"context"
	"reflect"

	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MigrationPlanScopeParams defines the input parameters used to create a new Scope.
type MigrationPlanScopeParams struct {
	Logger        logr.Logger
	Client        client.Client
	MigrationPlan *vjailbreakv1alpha1.MigrationPlan
}

// NewMigrationPlanScope creates a new MigrationPlanScope from the supplied parameters.
// This is meant to be called for each reconcile iteration only on MigrationPlanReconciler.
func NewMigrationPlanScope(params MigrationPlanScopeParams) (*MigrationPlanScope, error) {
	if reflect.DeepEqual(params.Logger, logr.Logger{}) {
		params.Logger = ctrl.Log
	}

	return &MigrationPlanScope{
		Logger:        params.Logger,
		Client:        params.Client,
		MigrationPlan: params.MigrationPlan,
	}, nil
}

// MigrationPlanScope defines the basic context for an actuator to operate upon.
type MigrationPlanScope struct {
	logr.Logger
	Client        client.Client
	MigrationPlan *vjailbreakv1alpha1.MigrationPlan
}

// Close closes the current scope persisting the MigrationPlan configuration and status.
func (s *MigrationPlanScope) Close() error {
	err := s.Client.Update(context.TODO(), s.MigrationPlan, &client.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// Name returns the MigrationPlan name.
func (s *MigrationPlanScope) Name() string {
	return s.MigrationPlan.GetName()
}

// Namespace returns the MigrationPlan namespace.
func (s *MigrationPlanScope) Namespace() string {
	return s.MigrationPlan.GetNamespace()
}
