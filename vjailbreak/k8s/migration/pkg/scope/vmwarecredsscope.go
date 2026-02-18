package scope

import (
	"context"
	"reflect"

	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VMwareCredsScopeParams defines the input parameters used to create a new Scope.
type VMwareCredsScopeParams struct {
	Logger      logr.Logger
	Client      client.Client
	VMwareCreds *vjailbreakv1alpha1.VMwareCreds
}

// NewVMwareCredsScope creates a new VMwareCredsScope from the supplied parameters.
// This is meant to be called for each reconcile iteration only on VMwareCredsReconciler.
func NewVMwareCredsScope(params VMwareCredsScopeParams) (*VMwareCredsScope, error) {
	if reflect.DeepEqual(params.Logger, logr.Logger{}) {
		params.Logger = ctrl.Log
	}

	return &VMwareCredsScope{
		Logger:      params.Logger,
		Client:      params.Client,
		VMwareCreds: params.VMwareCreds,
	}, nil
}

// VMwareCredsScope defines the basic context for an actuator to operate upon.
type VMwareCredsScope struct {
	logr.Logger
	Client      client.Client
	VMwareCreds *vjailbreakv1alpha1.VMwareCreds
}

// Close closes the current scope persisting the VMwareCreds configuration and status.
func (s *VMwareCredsScope) Close() error {
	err := s.Client.Update(context.TODO(), s.VMwareCreds, &client.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// Name returns the VMwareCreds name.
func (s *VMwareCredsScope) Name() string {
	return s.VMwareCreds.GetName()
}

// Namespace returns the VMwareCreds namespace.
func (s *VMwareCredsScope) Namespace() string {
	return s.VMwareCreds.GetNamespace()
}
