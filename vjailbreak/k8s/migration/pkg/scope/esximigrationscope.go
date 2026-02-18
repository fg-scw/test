package scope

import (
	"context"
	"reflect"

	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ESXIMigrationScopeParams defines the input parameters used to create a new Scope.
type ESXIMigrationScopeParams struct {
	Logger               logr.Logger
	Client               client.Client
	ESXIMigration        *vjailbreakv1alpha1.ESXIMigration
	RollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan
}

// NewESXIMigrationScope creates a new ESXIMigrationScope from the supplied parameters.
// This is meant to be called for each reconcile iteration only on ESXIMigrationReconciler.
func NewESXIMigrationScope(params ESXIMigrationScopeParams) (*ESXIMigrationScope, error) {
	if reflect.DeepEqual(params.Logger, logr.Logger{}) {
		params.Logger = ctrl.Log
	}

	return &ESXIMigrationScope{
		Logger:               params.Logger,
		Client:               params.Client,
		ESXIMigration:        params.ESXIMigration,
		RollingMigrationPlan: params.RollingMigrationPlan,
	}, nil
}

// ESXIMigrationScope defines the basic context for an actuator to operate upon.
type ESXIMigrationScope struct {
	logr.Logger
	Client               client.Client
	ESXIMigration        *vjailbreakv1alpha1.ESXIMigration
	RollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan
}

// Close closes the current scope persisting the ESXIMigration configuration and status.
func (s *ESXIMigrationScope) Close() error {
	err := s.Client.Update(context.TODO(), s.ESXIMigration, &client.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// Name returns the ESXIMigration name.
func (s *ESXIMigrationScope) Name() string {
	return s.ESXIMigration.GetName()
}

// Namespace returns the ESXIMigration namespace.
func (s *ESXIMigrationScope) Namespace() string {
	return s.ESXIMigration.GetNamespace()
}
