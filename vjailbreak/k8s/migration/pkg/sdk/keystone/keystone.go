// Copyright © 2020 The Platform9 Systems Inc.

package keystone

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	netutils "github.com/platform9/vjailbreak/pkg/common/utils"
	"go.uber.org/zap"
)

var (
	// ErrUserNotFound is returned when a requested user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrUserAlreadyExists is returned when attempting to create a user that already exists
	ErrUserAlreadyExists = errors.New("user already exists")
	// ErrProjectAlreadyExists is returned when attempting to create a project that already exists
	ErrProjectAlreadyExists = errors.New("project already exists")
)

// Client defines the interface for interacting with the OpenStack Keystone API
type Client interface {
	Auth(ctx context.Context, credentials Credentials) (AuthInfo, error)
	GetTokenInfo(ctx context.Context, token string) (AuthResponse, error)
	GetProjects(ctx context.Context, token string) ([]Project, error)
	CreateUser(ctx context.Context, token string, input CreateUserRequest) (*CreateUserResponse, error)
	DeleteUser(ctx context.Context, token string, userID string) error
	ListUser(ctx context.Context, token, filter string) ([]User, error)
	ListRoles(ctx context.Context, token, filter string) ([]Role, error)
	AssignRoleToUserOnProject(ctx context.Context, token string, projectID string, userID string, roleID string) error
	ListProjects(ctx context.Context, token, filter string) ([]Project, error)
}

// Credentials contains the authentication information needed to access the OpenStack Keystone API
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Tenant   string `json:"tenant,omitempty"`
	Region   string `json:"region,omitempty"`
}

// AuthInfo contains the authentication information returned after a successful login
type AuthInfo struct {
	Token     string
	UserID    string
	ProjectID string
	ExpiresAt time.Time
}

// AuthRequest represents the request structure for authenticating with Keystone
type AuthRequest struct {
	Auth AuthRequestAuth `json:"auth"`
}

// AuthRequestAuth defines the auth field structure in an authentication request
type AuthRequestAuth struct {
	Identity AuthRequestAuthIdentity `json:"identity"`
	Scope    AuthRequestAuthScope    `json:"scope"`
}

// AuthRequestAuthIdentity defines the identity field structure in an authentication request
type AuthRequestAuthIdentity struct {
	Password AuthRequestAuthIdentityPassword `json:"password"`
	Methods  []string                        `json:"methods"`
}

// AuthRequestAuthIdentityPassword defines the password field structure in an authentication identity
type AuthRequestAuthIdentityPassword struct {
	User AuthRequestAuthIdentityPasswordUser `json:"user"`
}

// AuthRequestAuthIdentityPasswordUser defines the user field structure for password authentication
type AuthRequestAuthIdentityPasswordUser struct {
	Domain   map[string]string `json:"domain"`
	Password string            `json:"password"`
	Name     string            `json:"name"`
}

// AuthRequestAuthScope defines the scope field structure in an authentication request
type AuthRequestAuthScope struct {
	Project AuthRequestAuthScopeProject `json:"project"`
}

// AuthRequestAuthScopeProject defines the project field structure in an authentication scope
type AuthRequestAuthScopeProject struct {
	Name   string                            `json:"name"`
	Domain AuthRequestAuthScopeProjectDomain `json:"domain"`
}

// AuthRequestAuthScopeProjectDomain defines the domain field structure in a project scope
type AuthRequestAuthScopeProjectDomain struct {
	ID string `json:"id"`
}

// AuthResponse represents the response structure received after authentication
type AuthResponse struct {
	Token AuthResponseToken `json:"token"`
}

// AuthResponseToken represents the token information in an authentication response
type AuthResponseToken struct {
	// Note: this is just a partial implementation
	ExpiresAt time.Time                `json:"expires_at"`
	IssuedAt  time.Time                `json:"issued_at"`
	User      AuthResponseTokenUser    `json:"user"`
	Project   AuthResponseTokenProject `json:"project"`
	Roles     []Role                   `json:"roles"`
	Methods   []string                 `json:"methods"`
}

// AuthResponseTokenUser contains user information included in the authentication token
type AuthResponseTokenUser struct {
	ID                string                            `json:"id"`
	Name              string                            `json:"name"`
	Domain            AuthRequestAuthScopeProjectDomain `json:"domain"`
	PasswordExpiresAt time.Time                         `json:"password_expires_at"`
}

// AuthResponseTokenProject contains project information included in the authentication token
type AuthResponseTokenProject struct {
	ID     string                            `json:"id"`
	Name   string                            `json:"service"`
	Domain AuthRequestAuthScopeProjectDomain `json:"domain"`
}

// AuthResponseTokenProjectDomain contains domain information for a project in the authentication token
type AuthResponseTokenProjectDomain struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetProjectsResponse represents the response structure when retrieving projects
type GetProjectsResponse struct {
	// Note: this is just a partial implementation
	Projects []Project `json:"projects"`
}

// Project represents an OpenStack project (tenant) entity
type Project struct {
	// Note: this is just a partial implementation
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// GetAllTenantsAllUsersResponse represents the response structure when retrieving all tenants and users
type GetAllTenantsAllUsersResponse struct {
	Tenants []TenantUsersMap `json:"tenants"`
}

// TenantUsersMap represents the mapping between a tenant and its users
type TenantUsersMap struct {
	TenantID    string       `json:"id"`
	Name        string       `json:"name"`
	Enabled     bool         `json:"enabled"`
	Description string       `json:"description,omitempty"`
	Users       []UserConfig `json:"users"`
}

// UserConfig contains configuration information for a user
type UserConfig struct {
	UserName    string `json:"username"`
	Name        string `json:"name"`
	UserID      string `json:"id"`
	DisplayName string `json:"displayname"`
	Email       string `json:"email,omitempty"`
}

// CreateUserRequest represents the request structure for creating a new user
type CreateUserRequest struct {
	User CreateUserRequestUser `json:"user"`
}

// CreateUserRequestUser contains the user details for creating a new user
type CreateUserRequestUser struct {
	Name             string `json:"name"`
	Password         string `json:"password"`
	DisplayName      string `json:"displayname"`
	DefaultProjectID string `json:"default_project_id,omitempty"`
	Email            string `json:"email,omitempty"`
	Domain           string `json:"domain,omitempty"`
	Description      string `json:"description,omitempty"`
}

// CreateUserResponse represents the response structure after creating a new user
type CreateUserResponse struct {
	User CreateUserResponseUser `json:"user"`
}

// CreateUserResponseUser contains the user details returned after creating a new user
type CreateUserResponseUser struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	PasswordExpiresAt string `json:"password_expires_at"`
	Enabled           bool   `json:"enabled"`
	Email             string `json:"email"`
	DefaultProjectID  string `json:"default_project_id"`
	Description       string `json:"description"`
}

// CreateProjectRequest represents the request structure for creating a new project
type CreateProjectRequest struct {
	Project CreateProjectRequestProject `json:"project"`
}

// CreateProjectRequestProject contains the project details for creating a new project
type CreateProjectRequestProject struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// Default value - true
	Enabled bool `json:"enabled,omitempty"`
	// Default DomainID to be - "default"
	DomainID string `json:"domain_id,omitempty"`
	// Default value - false
	IsDomain bool `json:"is_domain,omitempty"`
}

// CreateProjectResponse represents the response structure after creating a new project
type CreateProjectResponse struct {
	Project CreateProjectResponseProject `json:"project"`
}

// CreateProjectResponseProject contains the project details returned after creating a new project
type CreateProjectResponseProject struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	DomainID    string   `json:"domain_id"`
	Description string   `json:"description"`
	Enabled     bool     `json:"enabled"`
	ParentID    string   `json:"parent_id"`
	IsDomain    bool     `json:"is_domain"`
	Tags        []string `json:"tags"`
	Links       struct {
		Self string `json:"self"`
	} `json:"links"`
}

// ListUsersResponse represents the response structure when listing users
type ListUsersResponse struct {
	Users []User `json:"users"`
}

// User represents an OpenStack user entity
type User struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	PasswordExpiresAt string `json:"password_expires_at"`
	Enabled           bool   `json:"enabled"`
}

// ListRolesResponse represents the response structure when listing roles
type ListRolesResponse struct {
	Roles []Role `json:"roles"`
}

// Role represents an OpenStack role entity used for permissions
type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// HTTPClient implements the Client interface for interacting with Keystone API
type HTTPClient struct {
	endpoint   string
	httpClient *http.Client
	log        *zap.Logger
}

// NewClient creates a new HTTPClient for Keystone API communication with the specified endpoint and TLS settings.
func NewClient(endpoint string, insecure bool) (*HTTPClient, error) {
	client := netutils.NewVjbNet()
	returnClient := &HTTPClient{}
	// Turn off cert verification if running in airgapped mode
	pmkEnv := os.Getenv("PMK_ENVIRONMENT")
	if pmkEnv == "airgap" || insecure {
		zap.L().Debug("running in airgapped mode - disabling cert verification")
		client.Insecure = true
	}
	if err := client.CreateSecureHTTPClient(); err != nil {
		zap.L().Error("failed to create secure HTTP client", zap.Error(err))
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	returnClient.httpClient = client.GetClient()
	returnClient.endpoint = strings.TrimRight(endpoint, "/")
	returnClient.log = zap.L()
	return returnClient, nil
}

// Auth authenticates the user with the provided credentials.
/*
[Example Request]
curl -H 'Content-Type: application/json' -d '{"auth":{"identity":{"password":{"user":{"domain":{"id":"default"},"password":"$KEYSTONE_PASSWORD","name":"$KEYSTONE_USER"}},"methods":["password"]}}}' \
	$KEYSTONE_ENDPOINT/v3/auth/tokens

[Example Response]
Status: 200 OK
Headers:
- x-subject-token: $TOKEN
Body:
{
  "token": {
    "methods": [
      "password"
    ],
    "user": {
      "domain": {
        "id": "default",
        "name": "Default"
      },
      "id": "ebebd5d4aa6140f4b833e5d573efd583",
      "name": "$USERNAME",
      "password_expires_at": <date>
    },
    "audit_ids": [
      "62pEVKubTA6AuCoGbeVCSw"
    ],
    "expires_at": <date>,
    "issued_at": <date>,
    "project": {
      "domain": {
        "id": "default",
        "name": "Default"
      },
      "id": "f683d6132e764c798247d5bdb9542d6a",
      "name": "service"
    },
    "is_domain": false,
    "roles": [
      {
        "id": "b82b6dfe32784ab0a27ed6e32361a861",
        "name": "reader"
      }
    ]
  }
}
*/
func (c *HTTPClient) Auth(ctx context.Context, credentials Credentials) (AuthInfo, error) {
	reqBody, err := json.Marshal(credentialsToKeystoneAuthRequest(credentials))
	if err != nil {
		return AuthInfo{}, err
	}

	keystoneAuthEndpoint := c.endpoint + "/v3/auth/tokens?nocatalog"
	c.log.Debug("Requesting a new Keystone token from " + keystoneAuthEndpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, keystoneAuthEndpoint, bytes.NewReader(reqBody))
	if err != nil {
		return AuthInfo{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return AuthInfo{}, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return AuthInfo{}, err
	}

	if resp.StatusCode >= 400 {
		return AuthInfo{}, fmt.Errorf("failed to authenticate: received a %d from keystone: %s", resp.StatusCode, string(respBody))
	}

	tokenResp := &AuthResponse{}
	err = json.Unmarshal(respBody, tokenResp)
	if err != nil {
		return AuthInfo{}, err
	}

	token := resp.Header.Get("x-subject-token")
	if token == "" {
		return AuthInfo{}, errors.New("keystone authentication response did not contain the 'x-subject-token' header")
	}

	tokenInfo := AuthInfo{
		Token:     token,
		ProjectID: tokenResp.Token.Project.ID,
		ExpiresAt: tokenResp.Token.ExpiresAt,
		UserID:    tokenResp.Token.User.ID,
	}

	return tokenInfo, nil
}

// GetTokenInfo retrieves detailed information about a Keystone token.
func (c *HTTPClient) GetTokenInfo(ctx context.Context, token string) (AuthResponse, error) {
	keystoneAuthEndpoint := c.endpoint + "/v3/auth/tokens?nocatalog"
	c.log.Debug("Looking up Keystone token info from " + keystoneAuthEndpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, keystoneAuthEndpoint, nil)
	if err != nil {
		return AuthResponse{}, err
	}
	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("X-Subject-Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return AuthResponse{}, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return AuthResponse{}, err
	}
	if resp.StatusCode >= 400 {
		return AuthResponse{}, fmt.Errorf("failed to get token info: received a %d from keystone: %s", resp.StatusCode, string(respBody))
	}

	tokenResp := &AuthResponse{}
	err = json.Unmarshal(respBody, tokenResp)
	if err != nil {
		return AuthResponse{}, err
	}
	return *tokenResp, nil
}

// GetProjects lists projects available to the user.
/*
[Example request]
curl --silent --header "X-Auth-Token: $TOKEN" $KEYSTONE_ENDPOINT/v3/auth/projects

[Example response]
Status: 200 OK
Body:
{
    "links": {
        "self": "$KEYSTONE_ENDPOINT/v3/auth/projects",
        "previous": null,
        "next": null
    },
    "projects": [
        {
            "is_domain": false,
            "description": "",
            "links": {
                "self": "$KEYSTONE_ENDPOINT/v3/projects/314f9d9339824d978733b4b83662ae3a"
            },
            "enabled": true,
            "id": "314f9d9339824d978733b4b83662ae3a",
            "parent_id": "default",
            "domain_id": "default",
            "name": "service"
        }
    ]
}
*/
func (c *HTTPClient) GetProjects(ctx context.Context, token string) ([]Project, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/v3/auth/projects", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to fetch projects: received a %d from keystone: %s", resp.StatusCode, string(respBody))
	}

	projectsResp := &GetProjectsResponse{}
	err = json.Unmarshal(respBody, projectsResp)
	if err != nil {
		return nil, err
	}

	return projectsResp.Projects, nil
}

// GetAllTenantsAllUsers retrieves a mapping of all tenants to their associated users.
func (c *HTTPClient) GetAllTenantsAllUsers(ctx context.Context, token string) ([]TenantUsersMap, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/v3/PF9-KSADM/all_tenants_all_users", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to fetch all tenants to all users mapping: received a %d from keystone: %s", resp.StatusCode, string(respBody))
	}

	allTenantsMapResp := &GetAllTenantsAllUsersResponse{}
	err = json.Unmarshal(respBody, allTenantsMapResp)
	if err != nil {
		return nil, err
	}

	return allTenantsMapResp.Tenants, nil
}

// ListProjects retrieves a list of projects with optional filtering criteria.
func (c *HTTPClient) ListProjects(ctx context.Context, token, filter string) ([]Project, error) {
	var projectsEndpoint string

	if filter != "" {
		projectsEndpoint = fmt.Sprintf(c.endpoint+"/v3/projects?%s", filter)
	} else {
		projectsEndpoint = c.endpoint + "/v3/projects"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, projectsEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to fetch projects: received a %d from keystone: %s", resp.StatusCode, string(respBody))
	}

	projectsResp := &GetProjectsResponse{}
	err = json.Unmarshal(respBody, projectsResp)
	if err != nil {
		return nil, err
	}

	return projectsResp.Projects, nil
}

// CreateProject creates a new project (tenant) in Keystone.
// It makes an API call to the OpenStack Identity service to create a project resource.
// Ref: https://docs.openstack.org/api-ref/identity/v3/?expanded=create-user-detail,create-project-detail#projects
func (c *HTTPClient) CreateProject(ctx context.Context, token string, input CreateProjectRequest) (*CreateProjectResponse, error) {
	reqBody, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v3/projects", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// When project with same name is tried to be created.
	if resp.StatusCode == 409 {
		return nil, ErrProjectAlreadyExists
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to create project: received a %d from keystone: %s", resp.StatusCode, string(respBody))
	}

	createProjectResp := &CreateProjectResponse{}
	err = json.Unmarshal(respBody, createProjectResp)
	if err != nil {
		return nil, err
	}

	return createProjectResp, nil
}

// ListRoles retrieves a list of roles with optional filtering criteria.
func (c *HTTPClient) ListRoles(ctx context.Context, token, filter string) ([]Role, error) {
	var rolesEndpoint string

	if filter != "" {
		rolesEndpoint = fmt.Sprintf(c.endpoint+"/v3/roles?%s", filter)
	} else {
		rolesEndpoint = c.endpoint + "/v3/roles"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rolesEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to fetch roles: received a %d from keystone: %s", resp.StatusCode, string(respBody))
	}

	rolesList := &ListRolesResponse{}
	err = json.Unmarshal(respBody, rolesList)
	if err != nil {
		return nil, err
	}

	return rolesList.Roles, nil
}

// CheckRoleAssignForUserOnProject verifies if a user has a specific role assignment on a project.
// It checks if the user identified by userID has the role identified by roleID on the project identified by projectID.
// Ref: https://docs.openstack.org/api-ref/identity/v3/index.html?expanded=list-roles-detail#check-whether-user-has-role-assignment-on-project
func (c *HTTPClient) CheckRoleAssignForUserOnProject(ctx context.Context, token, projectID, userID, roleID string) bool {
	checkRoleEndpoint := fmt.Sprintf(c.endpoint+"/v3/projects/%s/users/%s/roles/%s", projectID, userID, roleID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkRoleEndpoint, nil)
	if err != nil {
		return false
	}

	req.Header.Set("X-Auth-Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	// Status code 204 refers to specified role is assigned to user on given project.
	return resp.StatusCode < 400
}

// AssignRoleToUserOnProject grants a specific role to a user on a project.
func (c *HTTPClient) AssignRoleToUserOnProject(ctx context.Context, token string, projectID string, userID string, roleID string) error {
	url := fmt.Sprintf("%s/v3/projects/%s/users/%s/roles/%s", c.endpoint, projectID, userID, roleID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Auth-Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to assign role to user on given project: received a %d from keystone: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// CreateUser creates a new user in Keystone with the provided details.
func (c *HTTPClient) CreateUser(ctx context.Context, token string, input CreateUserRequest) (*CreateUserResponse, error) {
	reqBody, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v3/users", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 409 {
		return nil, ErrUserAlreadyExists
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to create user: received a %d from keystone: %s", resp.StatusCode, string(respBody))
	}

	createUserResponse := &CreateUserResponse{}
	err = json.Unmarshal(respBody, createUserResponse)
	if err != nil {
		return nil, err
	}

	return createUserResponse, nil
}

// ListUser retrieves a list of users with optional filtering criteria.
func (c *HTTPClient) ListUser(ctx context.Context, token, filter string) ([]User, error) {
	var usersEndpoint string

	if filter != "" {
		usersEndpoint = fmt.Sprintf(c.endpoint+"/v3/users?%s", filter)
	} else {
		usersEndpoint = c.endpoint + "/v3/users"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usersEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to fetch users: received a %d from keystone: %s", resp.StatusCode, string(respBody))
	}

	usersList := &ListUsersResponse{}
	err = json.Unmarshal(respBody, usersList)
	if err != nil {
		return nil, err
	}

	return usersList.Users, nil
}

// DeleteUser removes a user from Keystone using their unique identifier.
func (c *HTTPClient) DeleteUser(ctx context.Context, token string, userID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.endpoint+"/v3/users/"+userID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == 404 {
		return ErrUserNotFound
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to delete user: received a %d from keystone: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// credentialsToKeystoneAuthRequest converts Credentials into an AuthRequest structure
// for use with the Keystone API authentication endpoints.
func credentialsToKeystoneAuthRequest(credentials Credentials) *AuthRequest {
	return &AuthRequest{
		Auth: AuthRequestAuth{
			Identity: AuthRequestAuthIdentity{
				Methods: []string{"password"},
				Password: AuthRequestAuthIdentityPassword{
					User: AuthRequestAuthIdentityPasswordUser{
						Domain: map[string]string{
							"id": "default",
						},
						Password: credentials.Password,
						Name:     credentials.Username,
					},
				},
			},
			Scope: AuthRequestAuthScope{
				Project: AuthRequestAuthScopeProject{
					Name: credentials.Tenant,
					Domain: AuthRequestAuthScopeProjectDomain{
						ID: "default",
					},
				},
			},
		},
	}
}
