package scope

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VjailbreakNodeScopeParams defines the input parameters used to create a new Scope.
type VjailbreakNodeScopeParams struct {
	Logger         logr.Logger
	Client         client.Client
	VjailbreakNode *vjailbreakv1alpha1.VjailbreakNode
}

// NewVjailbreakNodeScope creates a new VjailbreakNodeScope from the supplied parameters.
// This is meant to be called for each reconcile iteration only on VjailbreakNodeReconciler.
func NewVjailbreakNodeScope(params VjailbreakNodeScopeParams) (*VjailbreakNodeScope, error) {
	if reflect.DeepEqual(params.Logger, logr.Logger{}) {
		params.Logger = ctrl.Log
	}

	return &VjailbreakNodeScope{
		Logger:         params.Logger,
		Client:         params.Client,
		VjailbreakNode: params.VjailbreakNode,
	}, nil
}

// VjailbreakNodeScope defines the basic context for an actuator to operate upon.
type VjailbreakNodeScope struct {
	logr.Logger
	Client         client.Client
	VjailbreakNode *vjailbreakv1alpha1.VjailbreakNode
}

// Close closes the current scope persisting the VjailbreakNode configuration and status.
func (s *VjailbreakNodeScope) Close() error {
	err := s.Client.Update(context.TODO(), s.VjailbreakNode, &client.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// Name returns the VjailbreakNode name.
func (s *VjailbreakNodeScope) Name() string {
	return s.VjailbreakNode.GetName()
}

// Namespace returns the VjailbreakNode namespace.
func (s *VjailbreakNodeScope) Namespace() string {
	return s.VjailbreakNode.GetNamespace()
}
