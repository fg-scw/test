package scope

import (
	"context"
	"reflect"

	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RollingMigrationPlanScopeParams defines the input parameters used to create a new Scope.
type RollingMigrationPlanScopeParams struct {
	Logger               logr.Logger
	Client               client.Client
	RollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan
}

// NewRollingMigrationPlanScope creates a new RollingMigrationPlanScope from the supplied parameters.
// This is meant to be called for each reconcile iteration only on RollingMigrationPlanReconciler.
func NewRollingMigrationPlanScope(params RollingMigrationPlanScopeParams) (*RollingMigrationPlanScope, error) {
	if reflect.DeepEqual(params.Logger, logr.Logger{}) {
		params.Logger = ctrl.Log
	}

	return &RollingMigrationPlanScope{
		Logger:               params.Logger,
		Client:               params.Client,
		RollingMigrationPlan: params.RollingMigrationPlan,
	}, nil
}

// RollingMigrationPlanScope defines the basic context for an actuator to operate upon.
type RollingMigrationPlanScope struct {
	logr.Logger
	Client               client.Client
	RollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan
}

// Close closes the current scope persisting the RollingMigrationPlan configuration and status.
func (s *RollingMigrationPlanScope) Close() error {
	err := s.Client.Update(context.TODO(), s.RollingMigrationPlan, &client.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// Name returns the RollingMigrationPlan name.
func (s *RollingMigrationPlanScope) Name() string {
	return s.RollingMigrationPlan.GetName()
}

// Namespace returns the RollingMigrationPlan namespace.
func (s *RollingMigrationPlanScope) Namespace() string {
	return s.RollingMigrationPlan.GetNamespace()
}
