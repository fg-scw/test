package scope

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ArrayCredsScopeParams defines the input parameters for creating an ArrayCredsScope
type ArrayCredsScopeParams struct {
	Logger     logr.Logger
	Client     client.Client
	ArrayCreds *vjailbreakv1alpha1.ArrayCreds
}

// ArrayCredsScope defines the scope for ArrayCreds reconciliation
type ArrayCredsScope struct {
	logr.Logger
	Client     client.Client
	ArrayCreds *vjailbreakv1alpha1.ArrayCreds
}

// NewArrayCredsScope creates a new ArrayCredsScope from the supplied parameters
func NewArrayCredsScope(params ArrayCredsScopeParams) (*ArrayCredsScope, error) {
	if params.ArrayCreds == nil {
		return nil, errors.New("failed to generate new scope from nil ArrayCreds")
	}

	return &ArrayCredsScope{
		Logger:     params.Logger,
		Client:     params.Client,
		ArrayCreds: params.ArrayCreds,
	}, nil
}

// Close closes the ArrayCredsScope by updating the ArrayCreds resource
func (s *ArrayCredsScope) Close() error {
	return s.patchArrayCreds()
}

// patchArrayCreds patches the ArrayCreds resource
func (s *ArrayCredsScope) patchArrayCreds() error {
	return s.Client.Update(context.TODO(), s.ArrayCreds)
}
