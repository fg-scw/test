package keystone

import (
	"context"
	"time"
)

// FakeClient provides a test implementation of the Client interface
// that returns predefined values without making actual API calls to Keystone.
type FakeClient struct {
}

// NewFakeClient creates a new instance of FakeClient for testing purposes.
func NewFakeClient() *FakeClient {
	return &FakeClient{}
}

var _ Client = (*FakeClient)(nil)

// Auth implements the Client interface's Auth method for testing purposes.
// It returns a fake authentication response without making actual API calls.
func (f *FakeClient) Auth(_ context.Context, _ Credentials) (AuthInfo, error) {
	return AuthInfo{
		Token:     "faketoken",
		UserID:    "fakeuserid",
		ProjectID: "fakeprojectid",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil
}

// GetTokenInfo implements the Client interface's GetTokenInfo method for testing.
// This test implementation is not fully implemented and will panic when called.
func (f *FakeClient) GetTokenInfo(_ context.Context, _ string) (AuthResponse, error) {
	panic("implement me")
}

// GetProjects implements the Client interface's GetProjects method for testing.
// This test implementation is not fully implemented and will panic when called.
func (f *FakeClient) GetProjects(_ context.Context, _ string) ([]Project, error) {
	panic("implement me")
}

// CreateUser implements the Client interface's CreateUser method for testing.
// This test implementation is not fully implemented and will panic when called.
func (f *FakeClient) CreateUser(_ context.Context, _ string, _ CreateUserRequest) (*CreateUserResponse, error) {
	panic("implement me")
}

// DeleteUser implements the Client interface's DeleteUser method for testing.
// This test implementation is not fully implemented and will panic when called.
func (f *FakeClient) DeleteUser(_ context.Context, _ string, _ string) error {
	panic("implement me")
}

// ListUser implements the Client interface's ListUser method for testing.
// This test implementation is not fully implemented and will panic when called.
func (f *FakeClient) ListUser(_ context.Context, _ string, _ string) ([]User, error) {
	panic("implement me")
}

// ListRoles implements the Client interface's ListRoles method for testing.
// This test implementation is not fully implemented and will panic when called.
func (f *FakeClient) ListRoles(_ context.Context, _ string, _ string) ([]Role, error) {
	panic("implement me")
}

// AssignRoleToUserOnProject implements the Client interface's AssignRoleToUserOnProject method for testing.
// This test implementation is not fully implemented and will panic when called.
func (f *FakeClient) AssignRoleToUserOnProject(_ context.Context, _ string, _ string, _ string, _ string) error {
	panic("implement me")
}

// ListProjects implements the Client interface's ListProjects method for testing.
// This test implementation is not fully implemented and will panic when called.
func (f *FakeClient) ListProjects(_ context.Context, _ string, _ string) ([]Project, error) {
	panic("implement me")
}
