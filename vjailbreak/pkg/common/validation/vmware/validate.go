package vmware

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/scope"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/utils"
	commonutils "github.com/platform9/vjailbreak/pkg/common/utils"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/session/cache"
	"github.com/vmware/govmomi/vim25"
	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	sdkPath                  = "/sdk"
	namespaceMigrationSystem = "migration-system"
	defaultMaxRetries        = 5
	vjailbreakSettingsName   = "vjailbreak-settings"
	retryLimitKey            = "VCENTER_LOGIN_RETRY_LIMIT"
)

var vmwareClientMap *sync.Map

// ValidationResult holds the outcome of credential validation
type ValidationResult struct {
	Valid   bool
	Message string
	Error   error
}

// getRetryLimitFromSettings fetches VCENTER_LOGIN_RETRY_LIMIT from vjailbreak-settings
func getRetryLimitFromSettings(ctx context.Context, k8sClient client.Client) int {
	configMap := &corev1.ConfigMap{}
	err := k8sClient.Get(ctx, k8stypes.NamespacedName{
		Name:      vjailbreakSettingsName,
		Namespace: namespaceMigrationSystem,
	}, configMap)

	if err != nil {
		return defaultMaxRetries
	}

	if retryLimitStr, ok := configMap.Data[retryLimitKey]; ok {
		var retryLimit int
		if _, err := fmt.Sscanf(retryLimitStr, "%d", &retryLimit); err == nil && retryLimit > 0 {
			return retryLimit
		}
	}

	return defaultMaxRetries
}

// Validate performs complete VMware credential validation with all edge cases
func Validate(ctx context.Context, k8sClient client.Client, vmwcreds *vjailbreakv1alpha1.VMwareCreds, maxRetries ...int) ValidationResult {
	// Determine retry limit: use passed value, else fetch from ConfigMap, else use default
	retryLimit := defaultMaxRetries
	if len(maxRetries) > 0 && maxRetries[0] > 0 {
		retryLimit = maxRetries[0]
	} else {
		retryLimit = getRetryLimitFromSettings(ctx, k8sClient)
	}

	// Get credentials from secret
	vmwareCredsinfo, err := getCredentialsFromSecret(ctx, k8sClient, vmwcreds.Spec.SecretRef.Name)
	if err != nil {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("Failed to get vCenter credentials from secret: %s", err.Error()),
			Error:   err,
		}
	}

	host := vmwareCredsinfo.Host
	username := vmwareCredsinfo.Username
	password := vmwareCredsinfo.Password
	disableSSLVerification := vmwareCredsinfo.Insecure
	datacenter := vmwareCredsinfo.Datacenter

	// Normalize URL
	u, err := commonutils.NormalizeVCenterURL(host)
	if err != nil {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("Failed to normalize URL: %s", err.Error()),
			Error:   err,
		}
	}
	u.User = url.UserPassword(username, password)

	// Connect and log in to ESX or vCenter
	s := &cache.Session{
		URL:      u,
		Insecure: disableSSLVerification,
		Reauth:   true,
	}

	mapKey := string(vmwcreds.UID)
	var c *vim25.Client

	// Initialize map if needed
	if vmwareClientMap == nil {
		vmwareClientMap = &sync.Map{}
	}

	// Check cache for existing authenticated client
	if val, ok := vmwareClientMap.Load(mapKey); ok {
		cachedClient, valid := val.(*vim25.Client)
		if valid && cachedClient != nil && cachedClient.Client != nil {
			c = cachedClient
			sessMgr := session.NewManager(c)
			userSession, err := sessMgr.UserSession(ctx)
			if err == nil && userSession != nil {
				// Cached client is still valid
				return ValidationResult{
					Valid:   true,
					Message: "Successfully authenticated to VMware",
					Error:   nil,
				}
			}
			// Cached client is no longer valid, remove it
			vmwareClientMap.Delete(mapKey)
			// Will create fresh client in the retry loop
		}
	}

	// Exponential retry logic with retry limit from ConfigMap or passed parameter
	var lastErr error
	ctx = ensureLogger(ctx)
	ctxlog := ctrllog.FromContext(ctx)

	for attempt := 1; attempt <= retryLimit; attempt++ {
		// Create a new empty client struct for Login to populate
		c = &vim25.Client{}
		err = s.Login(ctx, c, nil)
		if err == nil {
			// Login successful
			ctxlog.Info("Login successful", "attempt", attempt)
			break
		} else if strings.Contains(err.Error(), "incorrect user name or password") {
			return ValidationResult{
				Valid:   false,
				Message: "Authentication failed: invalid username or password. Please verify your credentials",
				Error:   err,
			}
		}
		// Save the error and log it
		lastErr = err
		ctxlog.Info("Login attempt failed", "attempt", attempt, "error", err)
		// Retry with exponential backoff
		if attempt < retryLimit {
			delayNum := math.Pow(2, float64(attempt)) * 500
			ctxlog.Info("Retrying login after delay", "delayMs", delayNum)
			time.Sleep(time.Duration(delayNum) * time.Millisecond)
			// Client will be recreated in next iteration
		}
	}

	// Check if all login attempts failed
	if lastErr != nil {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("Failed to login to vCenter after %d attempts: %s", retryLimit, lastErr.Error()),
			Error:   lastErr,
		}
	}

	// Check if the datacenter exists (only if datacenter is provided)
	if datacenter != "" {
		finder := find.NewFinder(c, false)
		_, err = finder.Datacenter(ctx, datacenter)
		if err != nil {
			return ValidationResult{
				Valid:   false,
				Message: fmt.Sprintf("Failed to find datacenter: %s", err.Error()),
				Error:   err,
			}
		}
	}

	// All validations passed - cache the fully validated client
	vmwareClientMap.Store(mapKey, c)

	return ValidationResult{
		Valid:   true,
		Message: "Successfully authenticated to VMware",
		Error:   nil,
	}
}

// getCredentialsFromSecret retrieves VMware credentials from a Kubernetes secret
func getCredentialsFromSecret(ctx context.Context, k8sClient client.Client, secretName string) (vjailbreakv1alpha1.VMwareCredsInfo, error) {
	var vmwareCredsInfo vjailbreakv1alpha1.VMwareCredsInfo
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, k8stypes.NamespacedName{Name: secretName, Namespace: namespaceMigrationSystem}, secret)
	if err != nil {
		return vmwareCredsInfo, errors.Wrap(err, "failed to get secret")
	}

	if secret.Data == nil {
		return vmwareCredsInfo, fmt.Errorf("no data in secret '%s'", secretName)
	}

	vmwareCredsInfo.Host = string(secret.Data["VCENTER_HOST"])
	vmwareCredsInfo.Username = string(secret.Data["VCENTER_USERNAME"])
	vmwareCredsInfo.Password = string(secret.Data["VCENTER_PASSWORD"])
	vmwareCredsInfo.Datacenter = string(secret.Data["VCENTER_DATACENTER"])

	if vmwareCredsInfo.Host == "" {
		return vmwareCredsInfo, fmt.Errorf("VCENTER_HOST is missing in secret '%s'", secretName)
	}

	insecureStr := string(secret.Data["VCENTER_INSECURE"])
	vmwareCredsInfo.Insecure = strings.EqualFold(strings.TrimSpace(insecureStr), "true")

	return vmwareCredsInfo, nil
}

// PostValidationResources holds resources fetched after successful validation
type PostValidationResources struct {
	VMInfo     []vjailbreakv1alpha1.VMInfo
	RDMDiskMap *sync.Map
}

// FetchResourcesPostValidation fetches VMware resources after successful credential validation
func FetchResourcesPostValidation(ctx context.Context, k8sClient client.Client, vmwcreds *vjailbreakv1alpha1.VMwareCreds) (*PostValidationResources, error) {
	if vmwcreds == nil {
		return nil, fmt.Errorf("vmwcreds cannot be nil")
	}
	if k8sClient == nil {
		return nil, fmt.Errorf("k8sClient cannot be nil")
	}

	ctx = ensureLogger(ctx)
	logger := ctrllog.FromContext(ctx)

	scope, err := scope.NewVMwareCredsScope(scope.VMwareCredsScopeParams{
		Logger:      logger,
		Client:      k8sClient,
		VMwareCreds: vmwcreds,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create VMware scope")
	}

	log.Printf("Creating VMware Clusters and Hosts")
	err = utils.CreateVMwareClustersAndHosts(ctx, scope)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create VMware clusters and hosts")
	}

	log.Printf("Fetching all VMs")
	vminfo, rdmDiskMap, err := utils.GetAndCreateAllVMs(ctx, scope, vmwcreds.Spec.DataCenter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch VMs")
	}

	log.Printf("Syncing RDM Disks")
	err = utils.CreateOrUpdateRDMDisks(ctx, k8sClient, vmwcreds, rdmDiskMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create/update RDM disks")
	}

	log.Printf("Deleting Stale Machines")
	err = utils.DeleteStaleVMwareMachines(ctx, k8sClient, vmwcreds, vminfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to delete stale VMware machines")
	}

	log.Printf("Deleting Stale Clusters and Hosts")
	err = utils.DeleteStaleVMwareClustersAndHosts(ctx, scope)
	if err != nil {
		return nil, errors.Wrap(err, "failed to delete stale clusters and hosts")
	}

	return &PostValidationResources{
		VMInfo:     vminfo,
		RDMDiskMap: rdmDiskMap,
	}, nil
}

// ensureLogger ensures the context has a valid logger to prevent panics in shared packages
func ensureLogger(ctx context.Context) context.Context {
	l := ctrllog.FromContext(ctx)
	if l.GetSink() == nil {
		// Inject a dev logger if none exists
		return ctrllog.IntoContext(ctx, zap.New(zap.UseDevMode(true)))
	}
	return ctx
}
