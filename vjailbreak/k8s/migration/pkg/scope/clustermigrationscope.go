package scope

import (
	"context"
	"reflect"

	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterMigrationScopeParams defines the input parameters used to create a new Scope.
type ClusterMigrationScopeParams struct {
	Logger               logr.Logger
	Client               client.Client
	ClusterMigration     *vjailbreakv1alpha1.ClusterMigration
	RollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan
}

// NewClusterMigrationScope creates a new ClusterMigrationScope from the supplied parameters.
// This is meant to be called for each reconcile iteration only on ClusterMigrationReconciler.
func NewClusterMigrationScope(params ClusterMigrationScopeParams) (*ClusterMigrationScope, error) {
	if reflect.DeepEqual(params.Logger, logr.Logger{}) {
		params.Logger = ctrl.Log
	}

	return &ClusterMigrationScope{
		Logger:               params.Logger,
		Client:               params.Client,
		ClusterMigration:     params.ClusterMigration,
		RollingMigrationPlan: params.RollingMigrationPlan,
	}, nil
}

// ClusterMigrationScope defines the basic context for an actuator to operate upon.
type ClusterMigrationScope struct {
	logr.Logger
	Client               client.Client
	ClusterMigration     *vjailbreakv1alpha1.ClusterMigration
	RollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan
}

// Close closes the current scope persisting the ClusterMigration configuration and status.
func (s *ClusterMigrationScope) Close() error {
	err := s.Client.Update(context.TODO(), s.ClusterMigration, &client.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// Name returns the ClusterMigration name.
func (s *ClusterMigrationScope) Name() string {
	return s.ClusterMigration.GetName()
}

// Namespace returns the ClusterMigration namespace.
func (s *ClusterMigrationScope) Namespace() string {
	return s.ClusterMigration.GetNamespace()
}
