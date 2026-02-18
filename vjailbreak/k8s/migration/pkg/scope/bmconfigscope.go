package scope

import (
	"context"
	"reflect"

	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BMConfigScopeParams defines the input parameters used to create a new Scope.
type BMConfigScopeParams struct {
	Logger   logr.Logger
	Client   client.Client
	BMConfig *vjailbreakv1alpha1.BMConfig
}

// NewBMConfigScope creates a new BMConfigScope from the supplied parameters.
// This is meant to be called for each reconcile iteration only on BMConfigReconciler.
func NewBMConfigScope(params BMConfigScopeParams) (*BMConfigScope, error) {
	if reflect.DeepEqual(params.Logger, logr.Logger{}) {
		params.Logger = ctrl.Log
	}

	return &BMConfigScope{
		Logger:   params.Logger,
		Client:   params.Client,
		BMConfig: params.BMConfig,
	}, nil
}

// BMConfigScope defines the basic context for an actuator to operate upon.
type BMConfigScope struct {
	logr.Logger
	Client   client.Client
	BMConfig *vjailbreakv1alpha1.BMConfig
}

// Close closes the current scope persisting the BMConfig configuration and status.
func (s *BMConfigScope) Close() error {
	err := s.Client.Update(context.TODO(), s.BMConfig, &client.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// Name returns the BMConfig name.
func (s *BMConfigScope) Name() string {
	return s.BMConfig.GetName()
}

// Namespace returns the BMConfig namespace.
func (s *BMConfigScope) Namespace() string {
	return s.BMConfig.GetNamespace()
}
