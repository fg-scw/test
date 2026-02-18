package utils

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/constants"
	scope "github.com/platform9/vjailbreak/k8s/migration/pkg/scope"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/sdk/keystone"
	pcd "github.com/platform9/vjailbreak/k8s/migration/pkg/sdk/pcd"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/sdk/resmgr"
	netutils "github.com/platform9/vjailbreak/pkg/common/utils"
	"github.com/platform9/vjailbreak/pkg/vpwned/api/proto/v1/service"
	providers "github.com/platform9/vjailbreak/pkg/vpwned/sdk/providers"
	"github.com/platform9/vjailbreak/v2v-helper/pkg/k8sutils"

	// Import for side effects - registers the base provider implementation
	_ "github.com/platform9/vjailbreak/pkg/vpwned/sdk/providers/base"
	// Import for side effects - registers the maas provider implementation
	_ "github.com/platform9/vjailbreak/pkg/vpwned/sdk/providers/maas"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ConvertESXiToPCDHost converts an ESXi host to a PCD host by reclaiming the hardware
func ConvertESXiToPCDHost(ctx context.Context,
	scope *scope.ESXIMigrationScope,
	bmProvider providers.BMCProvider) error {
	ctxlog := log.FromContext(ctx).WithName(constants.ESXIMigrationControllerName)

	vmwarecreds, err := GetVMwareCredsFromRollingMigrationPlan(ctx, scope.Client, scope.RollingMigrationPlan)
	if err != nil {
		return errors.Wrap(err, "failed to get vmware credentials")
	}

	// list Mass machines
	resources, err := bmProvider.ListResources(ctx)
	if err != nil {
		return err
	}

	if len(resources) == 0 {
		return errors.New("no resources available from BM provisioner")
	}

	hs, err := GetESXiSummary(ctx, scope.Client, scope.ESXIMigration.Spec.ESXiName, vmwarecreds, "")
	if err != nil {
		return errors.Wrap(err, "failed to get ESXi summary")
	}

	// First, try to match by hardware UUID
	var matchedResourceID string
	var matchedResourceIdx = -1

	if hs.Hardware.SystemInfo.Uuid != "" {
		for i := 0; i < len(resources); i++ {
			if resources[i].HardwareUuid != "" && resources[i].HardwareUuid == hs.Hardware.SystemInfo.Uuid {
				ctxlog.Info("Found a matching resource by hardware UUID",
					"hardwareUuid", resources[i].HardwareUuid,
					"name", resources[i].Hostname,
					"id", resources[i].Id)
				matchedResourceID = resources[i].Id
				matchedResourceIdx = i
				break
			}
		}
	}

	// If no UUID match found, fall back to MAC address matching
	if matchedResourceIdx == -1 {
		ctxlog.Info("esxi has empty hardware UUID or no hardware UUID match found, trying MAC address matching")

		// Extract MAC addresses from ESXi host's physical network adapters
		var hostMacAddresses []string
		if hs.Config != nil && hs.Config.Network != nil && hs.Config.Network.Pnic != nil && len(hs.Config.Network.Pnic) > 0 {
			for _, pnic := range hs.Config.Network.Pnic {
				if pnic.Mac != "" {
					// Normalize MAC address to lowercase for comparison
					hostMacAddresses = append(hostMacAddresses, strings.ToLower(pnic.Mac))
				}
			}
		}

		if len(hostMacAddresses) == 0 {
			return errors.New("no hardware UUID or MAC addresses found for matching")
		}

		ctxlog.Info("ESXi host MAC addresses", "macs", hostMacAddresses)

		// Match resources based on MAC addresses (many-to-many)
		for i := 0; i < len(resources); i++ {
			if resources[i].MacAddress == "" {
				continue
			}

			resourceMac := strings.ToLower(resources[i].MacAddress)

			// Check if any host MAC address matches the resource MAC address
			matched := false
			for _, hostMac := range hostMacAddresses {
				if hostMac != "" && hostMac == resourceMac {
					matched = true
					break
				}
			}

			if matched {
				ctxlog.Info("Found a matching resource by MAC address",
					"resourceMAC", resourceMac,
					"name", resources[i].Hostname,
					"id", resources[i].Id,
					"hardwareUuid", resources[i].HardwareUuid)
				matchedResourceID = resources[i].Id
				matchedResourceIdx = i
				break
			}
		}
	}

	// If a match was found (either by UUID or MAC), reclaim the ESXi host
	if matchedResourceIdx != -1 {
		err := ReclaimESXi(ctx, scope, bmProvider, matchedResourceID, hs.Hardware.SystemInfo.Uuid)
		if err != nil {
			scope.ESXIMigration.Status.Phase = vjailbreakv1alpha1.ESXIMigrationPhaseFailed
			updateErr := scope.Client.Status().Update(ctx, scope.ESXIMigration)
			if updateErr != nil {
				return errors.Wrap(updateErr, "failed to update ESXi migration status")
			}
			return errors.Wrap(err, "failed to reclaim ESXi")
		}
	} else {
		return errors.New("no matching resource found by hardware UUID or MAC address")
	}

	scope.ESXIMigration.Status.Phase = vjailbreakv1alpha1.ESXIMigrationPhaseSucceeded
	err = scope.Client.Status().Update(ctx, scope.ESXIMigration)
	if err != nil {
		return errors.Wrap(err, "failed to update ESXi migration status")
	}

	return nil
}

// PrettyPrint outputs a JSON-formatted representation of the provided value
func PrettyPrint(v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(string(b))
}

// ReclaimESXi reclaims an ESXi host as a baremetal resource
func ReclaimESXi(ctx context.Context, scope *scope.ESXIMigrationScope, bmProvider providers.BMCProvider, resourceID string, hostID string) error {
	rollingMigrationPlan := scope.RollingMigrationPlan
	// Get BMConfig for the rolling migration plan
	bmConfig, err := GetBMConfigForRollingMigrationPlan(ctx, scope.Client, scope.RollingMigrationPlan)
	if err != nil {
		return errors.Wrap(err, "failed to get BMConfig for rolling migration plan")
	}
	// Get cloud init secret from rolling migration plan
	secret, err := GetCloudInitSecretFromRollingMigrationPlan(ctx, scope.Client, rollingMigrationPlan)
	if err != nil {
		return errors.Wrap(err, "failed to get cloud init secret from rolling migration plan")
	}
	if _, ok := secret.Data[constants.CloudInitConfigKey]; !ok {
		return errors.New("cloud init secret is empty")
	}

	cloudInit := string(secret.Data[constants.CloudInitConfigKey])
	cloudInit = strings.ReplaceAll(cloudInit, "HOST_ID", hostID)

	// Get vjailbreak settings to check if automatic PXE boot is enabled
	vjailbreakSettings, err := k8sutils.GetVjailbreakSettings(ctx, scope.Client)
	if err != nil {
		return errors.Wrap(err, "failed to get vjailbreak settings")
	}

	// Create ReclaimBM request
	reclaimRequest := service.ReclaimBMRequest{
		AccessInfo: &service.BMProvisionerAccessInfo{
			BaseUrl:     bmConfig.Spec.APIUrl,
			ApiKey:      bmConfig.Spec.APIKey,
			UseInsecure: bmConfig.Spec.Insecure,
		},
		UserData:           cloudInit,
		ResourceId:         resourceID,
		PowerCycle:         true,
		ManualPowerControl: !vjailbreakSettings.AutoPXEBootOnConversion,
		BootSource: &service.BootsourceSelections{
			Release: bmConfig.Spec.BootSource.Release,
		},
	}

	// Retry logic: attempt up to 3 times with exponential backoff
	maxRetries := 3
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Create a new request with same fields to avoid copying mutex
		newRequest := service.ReclaimBMRequest{
			AccessInfo:         reclaimRequest.AccessInfo,
			ResourceId:         reclaimRequest.ResourceId,
			UserData:           reclaimRequest.UserData,
			EraseDisk:          reclaimRequest.EraseDisk,
			BootSource:         reclaimRequest.BootSource,
			PowerCycle:         reclaimRequest.PowerCycle,
			ManualPowerControl: reclaimRequest.ManualPowerControl,
			IpmiInterface:      reclaimRequest.IpmiInterface,
		}
		err = bmProvider.ReclaimBM(ctx, newRequest) //nolint:govet // Safe - creating a new struct avoids copying mutex
		if err == nil {
			// Success, no need to retry
			break
		}

		lastErr = err
		if attempt < maxRetries {
			// Calculate backoff delay: 2^attempt * 500ms (0.5s, 1s, 2s)
			backoffTime := time.Duration(math.Pow(2, float64(attempt-1))*5) * time.Second
			logger := log.FromContext(ctx)
			logger.Info(
				"ReclaimBM attempt failed, retrying",
				"attempt", attempt,
				"maxRetries", maxRetries,
				"backoffTime", backoffTime,
				"error", err.Error(),
			)
			// Wait before retrying
			select {
			case <-ctx.Done():
				return errors.Wrap(ctx.Err(), "context canceled during ReclaimBM retry")
			case <-time.After(backoffTime):
				// Continue to next retry
			}
		}
	}

	if lastErr != nil {
		return errors.Wrap(lastErr, fmt.Sprintf("failed to reclaim BM after %d attempts", maxRetries))
	}
	return nil
}

// MergeCloudInit merges two cloud-init YAML configurations.
// userData and cloudInit are expected to be valid YAML strings in cloud-init format.
// The function returns the merged configuration as a YAML string with the #cloud-config directive.
func MergeCloudInit(userData, cloudInit string) (string, error) {
	// Parse both YAML strings into maps
	userDataMap := make(map[string]interface{})
	cloudInitMap := make(map[string]interface{})

	// Unmarshal the YAML strings
	if err := yaml.Unmarshal([]byte(userData), &userDataMap); err != nil {
		return "", fmt.Errorf("failed to parse userData: %v", err)
	}

	if err := yaml.Unmarshal([]byte(cloudInit), &cloudInitMap); err != nil {
		return "", fmt.Errorf("failed to parse cloudInit: %v", err)
	}

	// Deep merge the maps (cloudInit takes precedence over userData)
	mergedMap := deepMerge(userDataMap, cloudInitMap)

	// Marshal the result back to YAML
	result, err := yaml.Marshal(mergedMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal merged config: %v", err)
	}

	// Ensure the #cloud-config directive is at the beginning of the YAML
	return "#cloud-config\n" + string(result), nil
}

// appendScriptToRunCmd appends the pf9-setup.sh script execution to the runcmd section
func appendScriptToRunCmd(cloudInit string) (string, error) {
	// Parse the cloud-init YAML
	cloudInitMap := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(cloudInit), &cloudInitMap); err != nil {
		return "", fmt.Errorf("failed to parse cloudInit: %v", err)
	}

	// Get or create the runcmd section
	var runcmd []interface{}
	if existingRuncmd, ok := cloudInitMap["runcmd"]; ok {
		if runcmdSlice, ok := existingRuncmd.([]interface{}); ok {
			runcmd = runcmdSlice
		} else {
			return "", fmt.Errorf("runcmd exists in cloud-init but is not a valid array/slice type")
		}
	}

	// Append the script execution command
	runcmd = append(runcmd, "/root/pf9-setup.sh")
	cloudInitMap["runcmd"] = runcmd

	// Marshal back to YAML
	result, err := yaml.Marshal(cloudInitMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cloud-init: %v", err)
	}

	// Ensure the #cloud-config directive is at the beginning
	return "#cloud-config\n" + string(result), nil
}

// MergeCloudInitAndCreateSecret merges cloud-init configurations and creates a secret with the result
func MergeCloudInitAndCreateSecret(ctx context.Context, scope *scope.RollingMigrationPlanScope, local bool) error {
	// Get BMConfig for the rolling migration plan
	bmConfig, err := GetBMConfigForRollingMigrationPlan(ctx, scope.Client, scope.RollingMigrationPlan)
	if err != nil {
		return errors.Wrap(err, "failed to get BMConfig for rolling migration plan")
	}

	// Get user data from the secret
	userDataSecret, err := GetUserDataSecretForBMConfig(ctx, scope.Client, bmConfig)
	if err != nil {
		return errors.Wrap(err, "failed to get user data secret for BMConfig")
	}

	// Decode user data
	if _, ok := userDataSecret.Data[constants.UserDataSecretKey]; !ok {
		return errors.New("user data secret is empty")
	}

	userData := string(userDataSecret.Data[constants.UserDataSecretKey])

	cloudInit, err := generatePCDOnboardingCloudInit(ctx, scope, local)
	if err != nil {
		return errors.Wrap(err, "failed to generate cloud init for BMConfig")
	}

	// merge cloud init and user data
	mergedCloudInit, err := MergeCloudInit(userData, cloudInit)
	if err != nil {
		return errors.Wrap(err, "failed to merge cloud init and user data")
	}

	// Append script execution to runcmd
	mergedCloudInit, err = appendScriptToRunCmd(mergedCloudInit)
	if err != nil {
		return errors.Wrap(err, "failed to append script to runcmd")
	}

	finalCloudInitSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-merged", userDataSecret.Name),
			Namespace: constants.NamespaceMigrationSystem,
		},
		Data: map[string][]byte{
			constants.CloudInitConfigKey: []byte(mergedCloudInit),
		},
	}

	// Set the owner reference
	if err := controllerutil.SetOwnerReference(scope.RollingMigrationPlan, finalCloudInitSecret, scope.Client.Scheme()); err != nil {
		return errors.Wrap(err, "failed to set owner reference on cloud-init secret")
	}

	// Create or update the secret
	if err := scope.Client.Create(ctx, finalCloudInitSecret); err != nil {
		// Check if the error is because the resource already exists
		if strings.Contains(err.Error(), "already exists") {
			// Secret already exists, update it
			if err := scope.Client.Update(ctx, finalCloudInitSecret); err != nil {
				return errors.Wrap(err, "failed to update merged cloud init secret")
			}
		} else {
			return errors.Wrap(err, "failed to create merged cloud init secret")
		}
	}

	scope.RollingMigrationPlan.Spec.CloudInitConfigRef = &corev1.SecretReference{
		Name:      finalCloudInitSecret.Name,
		Namespace: constants.NamespaceMigrationSystem,
	}
	err = scope.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close scope")
	}

	return nil
}

func generatePCDOnboardingCloudInit(ctx context.Context, scope *scope.RollingMigrationPlanScope, local bool) (string, error) {
	openstackCreds, err := GetOpenstackCredsInfoFromRollingMigrationPlan(ctx, scope.Client, scope.RollingMigrationPlan)
	if err != nil {
		return "", errors.Wrap(err, "failed to get openstack credentials")
	}
	fqdn := strings.Split(openstackCreds.AuthURL, "/")[2]
	authURL := strings.Split(openstackCreds.AuthURL, "/")[:3]
	cloudInitParams := CloudInitParams{
		AuthURL:     strings.Join(authURL, "/"),
		Username:    openstackCreds.Username,
		Password:    openstackCreds.Password,
		RegionName:  openstackCreds.RegionName,
		TenantName:  openstackCreds.TenantName,
		Insecure:    openstackCreds.Insecure,
		DomainName:  openstackCreds.DomainName,
		FQDN:        fqdn,
		KeystoneURL: openstackCreds.AuthURL,
	}

	// read cloud-init template
	var cloudInitTemplateLocation string
	if local {
		cloudInitTemplateLocation = "./pkg/scripts/cloud-init.tmpl.yaml"
	} else {
		cloudInitTemplateLocation = "/pkg/scripts/cloud-init.tmpl.yaml"
	}
	cloudInitTemplateStr, err := os.ReadFile(cloudInitTemplateLocation) //nolint: gosec
	if err != nil {
		return "", errors.Wrap(err, "failed to read cloud-init template")
	}
	cloudInitTemplate, err := template.New("cloud-init").Parse(string(cloudInitTemplateStr))
	if err != nil {
		return "", errors.Wrap(err, "failed to parse cloud-init template")
	}
	var cloudInitBuffer bytes.Buffer
	err = cloudInitTemplate.Execute(&cloudInitBuffer, cloudInitParams)
	if err != nil {
		return "", errors.Wrap(err, "failed to execute cloud-init template")
	}
	return cloudInitBuffer.String(), nil
}

// GetBMConfigForRollingMigrationPlan retrieves the BMConfig associated with a RollingMigrationPlan
func GetBMConfigForRollingMigrationPlan(ctx context.Context,
	k8sClient client.Client,
	rollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan) (*vjailbreakv1alpha1.BMConfig, error) {
	bmConfig := &vjailbreakv1alpha1.BMConfig{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: rollingMigrationPlan.Spec.BMConfigRef.Name, Namespace: constants.NamespaceMigrationSystem}, bmConfig); err != nil {
		return nil, errors.Wrap(err, "failed to get BMConfig for rolling migration plan")
	}
	return bmConfig, nil
}

// GetOpenstackCredsForRollingMigrationPlan retrieves the OpenstackCreds associated with a RollingMigrationPlan
func GetOpenstackCredsForRollingMigrationPlan(ctx context.Context,
	k8sClient client.Client,
	rollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan) (*vjailbreakv1alpha1.OpenstackCreds, error) {
	migrationTemplate, err := GetMigrationTemplateFromRollingMigrationPlan(ctx, k8sClient, rollingMigrationPlan)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get migration template")
	}
	openstackCreds := &vjailbreakv1alpha1.OpenstackCreds{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: migrationTemplate.Spec.Destination.OpenstackRef, Namespace: constants.NamespaceMigrationSystem}, openstackCreds); err != nil {
		return nil, errors.Wrap(err, "failed to get OpenstackCreds for rolling migration plan")
	}
	return openstackCreds, nil
}

// GetUserDataForBMConfig retrieves user data configuration for a BMConfig from the RollingMigrationPlan
func GetUserDataForBMConfig(ctx context.Context, scope *scope.RollingMigrationPlanScope) (string, error) {
	bmConfig, err := GetBMConfig(ctx, scope.Client, scope.RollingMigrationPlan.Spec.BMConfigRef)
	if err != nil {
		return "", errors.Wrap(err, "failed to get BMConfig for rolling migration plan")
	}
	secret, err := GetUserDataSecretForBMConfig(ctx, scope.Client, bmConfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to get user data secret for BMConfig")
	}
	if _, ok := secret.Data[constants.UserDataSecretKey]; !ok {
		return "", errors.New("user data secret is empty")
	}
	userData, err := base64.StdEncoding.DecodeString(string(secret.Data[constants.UserDataSecretKey]))
	if err != nil {
		return "", errors.Wrap(err, "failed to decode user data secret for BMConfig")
	}

	return string(userData), nil
}

// GetUserDataSecretForBMConfig retrieves the secret containing user data for a BMConfig
func GetUserDataSecretForBMConfig(ctx context.Context, k8sClient client.Client, bmConfig *vjailbreakv1alpha1.BMConfig) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: bmConfig.Spec.UserDataSecretRef.Name, Namespace: constants.NamespaceMigrationSystem}, secret); err != nil {
		return nil, errors.Wrap(err, "failed to get user data secret for BMConfig")
	}
	return secret, nil
}

// GetCloudInitSecretFromRollingMigrationPlan retrieves the cloud-init secret from a RollingMigrationPlan
func GetCloudInitSecretFromRollingMigrationPlan(ctx context.Context,
	k8sClient client.Client,
	rollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: rollingMigrationPlan.Spec.CloudInitConfigRef.Name, Namespace: constants.NamespaceMigrationSystem}, secret); err != nil {
		return nil, errors.Wrap(err, "failed to get cloud init secret from rolling migration plan")
	}
	return secret, nil
}

// GetBMConfig retrieves a BMConfig by its reference
func GetBMConfig(ctx context.Context, k8sClient client.Client, bmConfigRef corev1.LocalObjectReference) (*vjailbreakv1alpha1.BMConfig, error) {
	bmConfig := &vjailbreakv1alpha1.BMConfig{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: bmConfigRef.Name, Namespace: constants.NamespaceMigrationSystem}, bmConfig); err != nil {
		return nil, errors.Wrap(err, "failed to get BMConfig")
	}
	return bmConfig, nil
}

// ValidateOpenstackIsPCD checks if the OpenStack environment is a Platform9 Distributed Cloud (PCD)
func ValidateOpenstackIsPCD(ctx context.Context, k8sClient client.Client, rollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan) (bool, error) {
	migrationTemplate, err := GetMigrationTemplateFromRollingMigrationPlan(ctx, k8sClient, rollingMigrationPlan)
	if err != nil {
		return false, errors.Wrap(err, "failed to get migration template")
	}
	openstackCreds := vjailbreakv1alpha1.OpenstackCreds{}
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      migrationTemplate.Spec.Destination.OpenstackRef,
		Namespace: constants.NamespaceMigrationSystem},
		&openstackCreds); err != nil {
		return false, errors.Wrap(err, "failed to get openstack credentials")
	}
	return IsOpenstackPCD(openstackCreds), nil
}

// IsOpenstackPCD determines if OpenStack credentials belong to a Platform9 Distributed Cloud
func IsOpenstackPCD(openstackCreds vjailbreakv1alpha1.OpenstackCreds) bool {
	if _, ok := openstackCreds.Labels[constants.IsPCDCredsLabel]; !ok {
		return false
	}
	return true
}

func getKeystoneAuthenticator(openstackCreds vjailbreakv1alpha1.OpenStackCredsInfo) (*keystone.CachedAuthenticator, error) {
	var ksClient keystone.Client
	var err error
	// Create the client
	ksClient, err = keystone.CreateFromOpenstackCreds(openstackCreds)

	if err != nil {
		// keystone is needed when its a CAPI enabled setup. However at this point there is no way to determine
		// if controller manager will be used with CAPI or not. Fail to be safe if keystone client cannot be created.
		return nil, fmt.Errorf("unable to create keystone client with insecure=true: %w", err)
	}

	if openstackCreds.AuthToken != "" {
		authenticator := keystone.NewCachedAuthenticator(keystone.NewStaticTokenAuthenticator(ksClient, openstackCreds.AuthToken))
		return authenticator, nil
	}

	creds, err := keystone.ParseCredentialsFromOpenstackCreds(openstackCreds)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch keystone creds from openstack creds: %w", err)
	}
	authenticator := keystone.NewCachedAuthenticator(keystone.NewBasicTokenGenerator(ksClient, creds))
	return authenticator, err
}

// GetResmgrClient creates a resource manager client from OpenStack credentials
func GetResmgrClient(openstackCreds vjailbreakv1alpha1.OpenStackCredsInfo) (resmgr.Resmgr, error) {
	ksAuth, err := getKeystoneAuthenticator(openstackCreds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get keystone authenticator")
	}
	var resmgrHTTPClient *http.Client
	vjbNet := netutils.NewVjbNet()
	if openstackCreds.Insecure {
		vjbNet.Insecure = true
	}
	if vjbNet.CreateSecureHTTPClient() == nil {
		resmgrHTTPClient = vjbNet.GetClient()
	} else {
		return nil, fmt.Errorf("failed to create secure HTTP client")
	}
	return resmgr.NewResmgrClient(
		resmgr.Config{
			DU: pcd.Info{
				URL:      strings.Join(strings.Split(openstackCreds.AuthURL, "/")[:3], "/"),
				Insecure: openstackCreds.Insecure,
			},
			Authenticator: ksAuth,
			HTTPClient:    *resmgrHTTPClient,
		},
	), nil
}

// GetVMwareCredsFromRollingMigrationPlan retrieves the source VMware credentials from a RollingMigrationPlan
func GetVMwareCredsFromRollingMigrationPlan(ctx context.Context, k3sclient client.Client, rollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan) (*vjailbreakv1alpha1.VMwareCreds, error) {
	migrationTemplate, err := GetMigrationTemplateFromRollingMigrationPlan(ctx, k3sclient, rollingMigrationPlan)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get migration template")
	}
	vmwarecreds := &vjailbreakv1alpha1.VMwareCreds{}
	err = k3sclient.Get(ctx, types.NamespacedName{Namespace: constants.NamespaceMigrationSystem, Name: migrationTemplate.Spec.Source.VMwareRef}, vmwarecreds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get vmware credentials")
	}
	return vmwarecreds, nil
}

// GetVMwareCredsInfoFromRollingMigrationPlan retrieves the source VMware credentials info from a RollingMigrationPlan
func GetVMwareCredsInfoFromRollingMigrationPlan(ctx context.Context, k3sclient client.Client, rollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan) (*vjailbreakv1alpha1.VMwareCredsInfo, error) {
	vmwarecreds, err := GetVMwareCredsFromRollingMigrationPlan(ctx, k3sclient, rollingMigrationPlan)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get vmware credentials")
	}
	vmwareCredsInfo, err := GetVMwareCredsInfo(ctx, k3sclient, vmwarecreds.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get vmware credentials")
	}
	return &vmwareCredsInfo, nil
}

// GetOpenstackCredsFromRollingMigrationPlan retrieves the destination OpenStack credentials from a RollingMigrationPlan
func GetOpenstackCredsFromRollingMigrationPlan(ctx context.Context, k3sclient client.Client, rollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan) (*vjailbreakv1alpha1.OpenstackCreds, error) {
	migrationTemplate, err := GetMigrationTemplateFromRollingMigrationPlan(ctx, k3sclient, rollingMigrationPlan)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get migration template")
	}
	openstackcreds := &vjailbreakv1alpha1.OpenstackCreds{}
	err = k3sclient.Get(ctx, types.NamespacedName{Namespace: constants.NamespaceMigrationSystem, Name: migrationTemplate.Spec.Destination.OpenstackRef}, openstackcreds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get openstack credentials")
	}
	return openstackcreds, nil
}

// GetOpenstackCredsInfoFromRollingMigrationPlan retrieves the destination OpenStack credentials info from a RollingMigrationPlan
func GetOpenstackCredsInfoFromRollingMigrationPlan(ctx context.Context, k3sclient client.Client, rollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan) (*vjailbreakv1alpha1.OpenStackCredsInfo, error) {
	openstackcreds, err := GetOpenstackCredsFromRollingMigrationPlan(ctx, k3sclient, rollingMigrationPlan)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get openstack credentials")
	}
	openstackCredsInfo, err := GetOpenstackCredsInfo(ctx, k3sclient, openstackcreds.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get openstack credentials")
	}
	return &openstackCredsInfo, nil
}

// GetMigrationTemplateFromRollingMigrationPlan retrieves the migration template from a RollingMigrationPlan
func GetMigrationTemplateFromRollingMigrationPlan(ctx context.Context, k3sclient client.Client, rollingMigrationPlan *vjailbreakv1alpha1.RollingMigrationPlan) (*vjailbreakv1alpha1.MigrationTemplate, error) {
	migrationTemplate := vjailbreakv1alpha1.MigrationTemplate{}
	if err := k3sclient.Get(ctx, types.NamespacedName{
		Name:      rollingMigrationPlan.Spec.MigrationTemplate,
		Namespace: constants.NamespaceMigrationSystem},
		&migrationTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to get migration template")
	}
	return &migrationTemplate, nil
}

// GetMigrationPlanFromMigration retrieves the migration plan from a Migration
func GetMigrationPlanFromMigration(ctx context.Context, k3sclient client.Client, migration *vjailbreakv1alpha1.Migration) (*vjailbreakv1alpha1.MigrationPlan, error) {
	migrationPlan := vjailbreakv1alpha1.MigrationPlan{}
	if err := k3sclient.Get(ctx, types.NamespacedName{
		Name:      migration.Spec.MigrationPlan,
		Namespace: constants.NamespaceMigrationSystem},
		&migrationPlan); err != nil {
		return nil, errors.Wrap(err, "failed to get migration plan")
	}
	return &migrationPlan, nil
}

// GetMigrationTemplateFromMigrationPlan retrieves the migration template from a MigrationPlan
func GetMigrationTemplateFromMigrationPlan(ctx context.Context, k3sclient client.Client, migrationPlan *vjailbreakv1alpha1.MigrationPlan) (*vjailbreakv1alpha1.MigrationTemplate, error) {
	migrationTemplate := vjailbreakv1alpha1.MigrationTemplate{}
	if err := k3sclient.Get(ctx, types.NamespacedName{
		Name:      migrationPlan.Spec.MigrationTemplate,
		Namespace: constants.NamespaceMigrationSystem},
		&migrationTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to get migration template")
	}
	return &migrationTemplate, nil
}

// GetMigrationTemplateFromMigration retrieves the migration template from a Migration
func GetMigrationTemplateFromMigration(ctx context.Context, k3sclient client.Client, migration *vjailbreakv1alpha1.Migration) (*vjailbreakv1alpha1.MigrationTemplate, error) {
	migrationPlan, err := GetMigrationPlanFromMigration(ctx, k3sclient, migration)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get migration plan")
	}
	return GetMigrationTemplateFromMigrationPlan(ctx, k3sclient, migrationPlan)
}

// GetVMwareCredsNameFromMigration retrieves the source VMware credentials name from a Migration
func GetVMwareCredsNameFromMigration(ctx context.Context, k3sclient client.Client, migration *vjailbreakv1alpha1.Migration) (string, error) {
	migrationTemplate, err := GetMigrationTemplateFromMigration(ctx, k3sclient, migration)
	if err != nil {
		return "", errors.Wrap(err, "failed to get migration template")
	}
	return migrationTemplate.Spec.Source.VMwareRef, nil
}

// GetOpenstackCredsNameFromMigration retrieves the destination OpenStack credentials name from a Migration
func GetOpenstackCredsNameFromMigration(ctx context.Context, k3sclient client.Client, migration *vjailbreakv1alpha1.Migration) (string, error) {
	migrationTemplate, err := GetMigrationTemplateFromMigration(ctx, k3sclient, migration)
	if err != nil {
		return "", errors.Wrap(err, "failed to get migration template")
	}
	return migrationTemplate.Spec.Destination.OpenstackRef, nil
}

// GetVMwareCredsNameFromMigrationPlan retrieves the source VMware credentials name from a MigrationPlan
func GetVMwareCredsNameFromMigrationPlan(ctx context.Context, k3sclient client.Client, migrationPlan *vjailbreakv1alpha1.MigrationPlan) (string, error) {
	migrationTemplate, err := GetMigrationTemplateFromMigrationPlan(ctx, k3sclient, migrationPlan)
	if err != nil {
		return "", errors.Wrap(err, "failed to get migration template")
	}
	return migrationTemplate.Spec.Source.VMwareRef, nil
}

// GetOpenstackCredsNameFromMigrationPlan retrieves the destination OpenStack credentials name from a MigrationPlan
func GetOpenstackCredsNameFromMigrationPlan(ctx context.Context, k3sclient client.Client, migrationPlan *vjailbreakv1alpha1.MigrationPlan) (string, error) {
	migrationTemplate, err := GetMigrationTemplateFromMigrationPlan(ctx, k3sclient, migrationPlan)
	if err != nil {
		return "", errors.Wrap(err, "failed to get migration template")
	}
	return migrationTemplate.Spec.Destination.OpenstackRef, nil
}
