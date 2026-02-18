package scope

import (
	"context"
	"reflect"

	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OpenstackCredsScopeParams defines the input parameters used to create a new Scope.
type OpenstackCredsScopeParams struct {
	Logger         logr.Logger
	Client         client.Client
	OpenstackCreds *vjailbreakv1alpha1.OpenstackCreds
}

// NewOpenstackCredsScope creates a new OpenstackCredsScope from the supplied parameters.
// This is meant to be called for each reconcile iteration only on OpenstackCredsReconciler.
func NewOpenstackCredsScope(params OpenstackCredsScopeParams) (*OpenstackCredsScope, error) {
	if reflect.DeepEqual(params.Logger, logr.Logger{}) {
		params.Logger = ctrl.Log
	}

	return &OpenstackCredsScope{
		Logger:         params.Logger,
		Client:         params.Client,
		OpenstackCreds: params.OpenstackCreds,
	}, nil
}

// OpenstackCredsScope defines the basic context for an actuator to operate upon.
type OpenstackCredsScope struct {
	logr.Logger
	Client         client.Client
	OpenstackCreds *vjailbreakv1alpha1.OpenstackCreds
}

// Close closes the current scope persisting the OpenstackCreds configuration and status.
func (s *OpenstackCredsScope) Close() error {
	err := s.Client.Update(context.TODO(), s.OpenstackCreds, &client.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// Name returns the OpenstackCreds name.
func (s *OpenstackCredsScope) Name() string {
	return s.OpenstackCreds.GetName()
}

// Namespace returns the OpenstackCreds namespace.
func (s *OpenstackCredsScope) Namespace() string {
	return s.OpenstackCreds.GetNamespace()
}
