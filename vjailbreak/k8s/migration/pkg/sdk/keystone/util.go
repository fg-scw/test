package keystone

import (
	"errors"
	"fmt"
	"os"
	"strings"

	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	pcd "github.com/platform9/vjailbreak/k8s/migration/pkg/sdk/pcd"
	"go.uber.org/zap"
)

// ParseCredentialsFromEnv creates a Credentials struct from environment variables.
// It reads KEYSTONE_USER, KEYSTONE_PASSWORD, KEYSTONE_TENANT, and KEYSTONE_REGION.
// Returns an error if any required environment variable is missing.
func ParseCredentialsFromEnv() (Credentials, error) {
	creds := Credentials{
		Username: strings.TrimSpace(os.Getenv("KEYSTONE_USER")),
		Password: strings.TrimSpace(os.Getenv("KEYSTONE_PASSWORD")),
		Tenant:   strings.TrimSpace(os.Getenv("KEYSTONE_TENANT")),
		Region:   strings.TrimSpace(os.Getenv("KEYSTONE_REGION")),
	}
	if len(creds.Username) == 0 {
		return creds, errors.New("environment variable KEYSTONE_USER is required")
	}
	if len(creds.Password) == 0 {
		return creds, errors.New("environment variable KEYSTONE_PASSWORD is required")
	}
	if len(creds.Tenant) == 0 {
		return creds, errors.New("environment variable KEYSTONE_TENANT is required")
	}
	if len(creds.Region) == 0 {
		return creds, errors.New("environment variable KEYSTONE_REGION is required")
	}
	return creds, nil
}

// ParseCredentialsFromOpenstackCreds converts OpenStackCredsInfo to keystone Credentials.
// This allows using OpenStack credentials stored in a Kubernetes CRD to authenticate with Keystone.
func ParseCredentialsFromOpenstackCreds(openstackCreds vjailbreakv1alpha1.OpenStackCredsInfo) (Credentials, error) {
	return Credentials{
		Username: openstackCreds.Username,
		Password: openstackCreds.Password,
		Tenant:   openstackCreds.TenantName,
		Region:   openstackCreds.RegionName,
	}, nil
}

// CreateFromEnv creates a new Keystone client using environment variables.
// It uses PCD information from environment variables to construct the Keystone endpoint.
func CreateFromEnv() (Client, error) {
	// TODO support overriding it
	pcdInfo, err := pcd.ParseInfoFromEnv()
	if err != nil {
		return nil, err
	}
	return CreateFromDuInfo(pcdInfo), nil
}

// CreateFromOpenstackCreds creates a new Keystone client using OpenStackCredsInfo.
// It uses the OpenStack credentials to construct the Keystone endpoint and client.
func CreateFromOpenstackCreds(openstackCreds vjailbreakv1alpha1.OpenStackCredsInfo) (Client, error) {
	// TODO support overriding it
	pcdInfo, err := pcd.ParseInfoFromOpenstackCreds(openstackCreds)
	if err != nil {
		return nil, err
	}
	return CreateFromDuInfo(pcdInfo), nil
}

// CreateFromDuInfo creates a new Keystone client using PCD information.
// It constructs the Keystone endpoint URL and creates a client with appropriate security settings.
func CreateFromDuInfo(pcdInfo pcd.Info) Client {
	keystoneEndpoint := fmt.Sprintf("%s/keystone", pcdInfo.URL)
	client, err := NewClient(keystoneEndpoint, pcdInfo.Insecure)
	if err != nil {
		zap.L().Error("failed to create keystone client", zap.Error(err))
		return nil
	}
	return client
}
