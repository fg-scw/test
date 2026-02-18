package openstack

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/pkg/errors"
	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/utils"
	netutils "github.com/platform9/vjailbreak/pkg/common/utils"
	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	namespaceMigrationSystem = "migration-system"
)

// ValidationResult holds the outcome of credential validation
type ValidationResult struct {
	Valid   bool
	Message string
	Error   error
}

func authErrorMessage(err error, isTokenAuth bool) string {
	if err == nil {
		return ""
	}
	var message string
	switch {
	case strings.Contains(err.Error(), "401"):
		if isTokenAuth {
			message = "Authentication failed: invalid or expired token. Please generate a fresh token and try again"
		} else {
			message = "Authentication failed: invalid username, password, or project/domain. Please verify your credentials"
		}
	case strings.Contains(err.Error(), "404"):
		message = "Authentication failed: the authentication URL or tenant/project name is incorrect"
	case strings.Contains(err.Error(), "timeout"):
		message = "Connection timeout: unable to reach the OpenStack authentication service. Please check your network connection and Auth URL"
	default:
		message = fmt.Sprintf("Authentication failed: %s. Please verify your OpenStack credentials", err.Error())
	}
	return message
}

// Validate performs complete OpenStack credential validation
func Validate(ctx context.Context, k8sClient client.Client, openstackcreds *vjailbreakv1alpha1.OpenstackCreds) ValidationResult {
	// Ensure logger exists
	ctx = ensureLogger(ctx)
	// Get credentials from secret
	openstackCredential, err := getCredentialsFromSecret(ctx, k8sClient, openstackcreds.Spec.SecretRef.Name)
	if err != nil {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("Failed to get credentials from secret: %s", err.Error()),
			Error:   err,
		}
	}

	// Create provider client
	providerClient, err := openstack.NewClient(openstackCredential.AuthURL)
	if err != nil {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("Failed to create OpenStack client: %s", err.Error()),
			Error:   err,
		}
	}

	vjbNet := netutils.NewVjbNet()
	if openstackCredential.Insecure {
		vjbNet.Insecure = true
	} else {
		fmt.Printf("Warning: TLS verification is enforced by default. If you encounter certificate errors, set OS_INSECURE=true to skip verification.\n")
	}
	vjbNet.SetTimeout(60 * time.Second)
	if err := vjbNet.CreateSecureHTTPClient(); err != nil {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("failed to create secure HTTP client"),
			Error:   fmt.Errorf("failed to create secure HTTP client %v", err),
		}
	}
	providerClient.HTTPClient = *vjbNet.GetClient()

	// Authenticate based on available credentials
	if openstackCredential.AuthToken != "" {
		authOpts := gophercloud.AuthOptions{
			IdentityEndpoint: openstackCredential.AuthURL,
			TokenID:          openstackCredential.AuthToken,
			TenantName:       openstackCredential.TenantName,
		}
		if openstackCredential.DomainName != "" {
			authOpts.DomainName = openstackCredential.DomainName
		}

		if err := openstack.Authenticate(ctx, providerClient, authOpts); err != nil {
			message := authErrorMessage(err, true)
			return ValidationResult{
				Valid:   false,
				Message: message,
				Error:   err,
			}
		}
	} else {
		// Password-based authentication: Use standard authentication flow
		authOpts := gophercloud.AuthOptions{
			IdentityEndpoint: openstackCredential.AuthURL,
			Username:         openstackCredential.Username,
			Password:         openstackCredential.Password,
			DomainName:       openstackCredential.DomainName,
			TenantName:       openstackCredential.TenantName,
		}

		if err := openstack.Authenticate(ctx, providerClient, authOpts); err != nil {
			message := authErrorMessage(err, false)
			return ValidationResult{
				Valid:   false,
				Message: message,
				Error:   err,
			}
		}
	}
	if providerClient.EndpointLocator == nil {
		return ValidationResult{
			Valid:   false,
			Message: "Authentication failed: client endpoint catalog was not initialized. Please verify your OpenStack credentials and try again",
			Error:   fmt.Errorf("authentication returned no error but ProviderClient.EndpointLocator is nil"),
		}
	}

	// Verify credentials match current environment
	_, err = verifyCredentialsMatchCurrentEnvironment(providerClient, openstackCredential.RegionName)
	if err != nil {
		if strings.Contains(err.Error(), "Creds are valid but for a different OpenStack environment") {
			return ValidationResult{
				Valid:   false,
				Message: "Creds are valid but for a different OpenStack environment",
				Error:   err,
			}
		}
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("Failed to verify credentials against current environment: %s", err.Error()),
			Error:   err,
		}
	}

	return ValidationResult{
		Valid:   true,
		Message: "Successfully authenticated to Openstack",
		Error:   nil,
	}
}

// getCredentialsFromSecret retrieves OpenStack credentials from a Kubernetes secret
func getCredentialsFromSecret(ctx context.Context, k8sClient client.Client, secretName string) (vjailbreakv1alpha1.OpenStackCredsInfo, error) {
	var openstackCredsInfo vjailbreakv1alpha1.OpenStackCredsInfo
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, k8stypes.NamespacedName{Name: secretName, Namespace: namespaceMigrationSystem}, secret)
	if err != nil {
		return openstackCredsInfo, errors.Wrap(err, "failed to get secret")
	}

	// Check which authentication method is being used
	authToken := string(secret.Data["OS_AUTH_TOKEN"])
	username := string(secret.Data["OS_USERNAME"])
	password := string(secret.Data["OS_PASSWORD"])

	// Common required fields for both auth methods
	authURL := string(secret.Data["OS_AUTH_URL"])
	tenantName := string(secret.Data["OS_TENANT_NAME"])
	if tenantName == "" {
		tenantName = string(secret.Data["OS_PROJECT_NAME"])
	}
	regionName := string(secret.Data["OS_REGION_NAME"])

	// Validate common required fields
	if authURL == "" {
		return openstackCredsInfo, fmt.Errorf("field OS_AUTH_URL is empty or missing in secret")
	}
	if tenantName == "" {
		return openstackCredsInfo, fmt.Errorf("field OS_TENANT_NAME is empty or missing in secret")
	}
	if regionName == "" {
		return openstackCredsInfo, fmt.Errorf("field OS_REGION_NAME is empty or missing in secret")
	}

	// Determine authentication method and validate accordingly
	if authToken != "" {
		// Token-based authentication
		openstackCredsInfo.AuthToken = authToken
		openstackCredsInfo.AuthURL = authURL
		openstackCredsInfo.TenantName = tenantName
		openstackCredsInfo.RegionName = regionName
		// DomainName is optional for token-based auth
		openstackCredsInfo.DomainName = string(secret.Data["OS_DOMAIN_NAME"])
	} else if username != "" && password != "" {
		// Password-based authentication
		domainName := string(secret.Data["OS_DOMAIN_NAME"])
		if domainName == "" {
			return openstackCredsInfo, fmt.Errorf("field OS_DOMAIN_NAME is empty or missing in secret for password-based auth")
		}

		openstackCredsInfo.AuthURL = authURL
		openstackCredsInfo.Username = username
		openstackCredsInfo.Password = password
		openstackCredsInfo.DomainName = domainName
		openstackCredsInfo.TenantName = tenantName
		openstackCredsInfo.RegionName = regionName
	} else {
		// Neither authentication method has complete credentials
		return openstackCredsInfo, fmt.Errorf("missing required fields: either OS_AUTH_TOKEN or (OS_USERNAME and OS_PASSWORD) must be provided")
	}

	// Parse insecure flag
	insecureStr := string(secret.Data["OS_INSECURE"])
	openstackCredsInfo.Insecure = strings.EqualFold(strings.TrimSpace(insecureStr), "true")

	return openstackCredsInfo, nil
}

// verifyCredentialsMatchCurrentEnvironment checks if the provided credentials can access the current instance
func verifyCredentialsMatchCurrentEnvironment(providerClient *gophercloud.ProviderClient, regionName string) (ok bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
			err = fmt.Errorf("panic while verifying OpenStack environment: %v", r)
		}
	}()

	if providerClient == nil {
		return false, fmt.Errorf("provider client is nil")
	}
	if providerClient.EndpointLocator == nil {
		return false, fmt.Errorf("OpenStack client is not authenticated (endpoint locator is not initialized)")
	}
	if strings.TrimSpace(regionName) == "" {
		return false, fmt.Errorf("region name is empty")
	}

	// Get current instance metadata
	metadata, err := utils.GetCurrentInstanceMetadata()
	if err != nil {
		return false, fmt.Errorf("unable to get current instance metadata: %w. "+
			"Please ensure this is running on an OpenStack instance with metadata service enabled", err)
	}

	computeClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{
		Region: regionName,
	})

	if err != nil {
		return false, fmt.Errorf("failed to create OpenStack compute client: %w", err)
	}
	_, err = servers.Get(context.TODO(), computeClient, metadata.UUID).Extract()
	if err != nil {
		if strings.Contains(err.Error(), "Resource not found") ||
			strings.Contains(err.Error(), "No server with a name or ID") {
			return false, errors.New("Creds are valid but for a different OpenStack environment")
		}
		return false, fmt.Errorf("failed to verify instance access: %w. "+
			"Please check if the provided credentials have compute:get_server permission", err)
	}
	return true, nil
}

// PostValidationResources holds resources fetched after successful validation
type PostValidationResources struct {
	Flavors       []flavors.Flavor
	OpenstackInfo *vjailbreakv1alpha1.OpenstackInfo
	ProjectName   string
}

// FetchResourcesPostValidation fetches OpenStack resources after successful credential validation
func FetchResourcesPostValidation(ctx context.Context, k8sClient client.Client, openstackcreds *vjailbreakv1alpha1.OpenstackCreds) (*PostValidationResources, error) {
	if openstackcreds == nil {
		return nil, fmt.Errorf("openstackcreds cannot be nil")
	}

	ctx = ensureLogger(ctx)

	log.Printf("Updating Master Node Image ID")
	err := utils.UpdateMasterNodeImageID(ctx, k8sClient, false)
	if err != nil {
		log.Printf("Warning: Failed to update master node image ID: %v", err)
	}

	log.Printf("Creating Dummy PCD Cluster if needed")
	err = utils.CreateDummyPCDClusterForStandAlonePCDHosts(ctx, k8sClient, openstackcreds)
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return nil, errors.Wrap(err, "failed to create dummy PCD cluster")
		}
		log.Printf("Dummy PCD Cluster already exists, continuing")
	}

	log.Printf("Listing Flavors")
	flavorsList, err := utils.ListAllFlavors(ctx, k8sClient, openstackcreds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list flavors")
	}

	log.Printf("Getting OpenStack Info")
	openstackInfo, err := utils.GetOpenstackInfo(ctx, k8sClient, openstackcreds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get OpenStack info")
	}

	openstackCredential, err := getCredentialsFromSecret(ctx, k8sClient, openstackcreds.Spec.SecretRef.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get credentials for project name")
	}

	return &PostValidationResources{
		Flavors:       flavorsList,
		OpenstackInfo: openstackInfo,
		ProjectName:   openstackCredential.TenantName,
	}, nil
}

func ensureLogger(ctx context.Context) context.Context {
	l := ctrllog.FromContext(ctx)
	if l.GetSink() == nil {
		return ctrllog.IntoContext(ctx, zap.New(zap.UseDevMode(true)))
	}
	return ctx
}
