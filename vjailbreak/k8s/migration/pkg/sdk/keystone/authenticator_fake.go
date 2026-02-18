package keystone

import (
	"context"
	"time"
)

// FakeAuthenticator provides a test implementation of the Authenticator interface
// that returns predefined values without making actual API calls.
type FakeAuthenticator struct {
}

// NewFakeAuthenticator creates a new instance of FakeAuthenticator for testing purposes.
func NewFakeAuthenticator() *FakeAuthenticator {
	return &FakeAuthenticator{}
}

// Auth provides a fake implementation of the Authenticator interface's Auth method.
// It returns a predefined authentication response without making actual API calls.
func (f *FakeAuthenticator) Auth(_ context.Context, _ ...AuthOptions) (AuthInfo, error) {
	return AuthInfo{
		Token:     "token-qwertyuiop",
		UserID:    "user-id-123",
		ProjectID: "project-id-123",
		ExpiresAt: time.Now(),
	}, nil
}

// ResetCache implements the Authenticator interface's ResetCache method for testing purposes.
// This test implementation simply returns nil without performing any operations.
func (f *FakeAuthenticator) ResetCache() error {
	return nil
}
