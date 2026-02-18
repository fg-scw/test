package scope

import (
	"context"
	"reflect"

	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MigrationScopeParams defines the input parameters used to create a new Scope.
type MigrationScopeParams struct {
	Logger    logr.Logger
	Client    client.Client
	Migration *vjailbreakv1alpha1.Migration
}

// NewMigrationScope creates a new MigrationScope from the supplied parameters.
// This is meant to be called for each reconcile iteration only on MigrationReconciler.
func NewMigrationScope(params MigrationScopeParams) (*MigrationScope, error) {
	if reflect.DeepEqual(params.Logger, logr.Logger{}) {
		params.Logger = ctrl.Log
	}

	return &MigrationScope{
		Logger:    params.Logger,
		Client:    params.Client,
		Migration: params.Migration,
	}, nil
}

// MigrationScope defines the basic context for an actuator to operate upon.
type MigrationScope struct {
	logr.Logger
	Client    client.Client
	Migration *vjailbreakv1alpha1.Migration
}

// Close closes the current scope persisting the Migration configuration and status.
func (s *MigrationScope) Close() error {
	err := s.Client.Update(context.TODO(), s.Migration, &client.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// Name returns the Migration name.
func (s *MigrationScope) Name() string {
	return s.Migration.GetName()
}

// Namespace returns the Migration namespace.
func (s *MigrationScope) Namespace() string {
	return s.Migration.GetNamespace()
}
