package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	"github.com/platform9/vjailbreak/v2v-helper/pkg/constants"
	"github.com/platform9/vjailbreak/v2v-helper/pkg/k8sutils"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

func RemoveEmptyStrings(slice []string) []string {
	var result []string
	for _, str := range slice {
		if str != "" {
			result = append(result, str)
		}
	}
	return result
}

func GetInclusterClient() (client.Client, error) {
	// Create a direct Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get in-cluster config")
	}
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(vjailbreakv1alpha1.AddToScheme(scheme))
	clientset, err := client.New(config, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get in-cluster config")
	}

	return clientset, err
}

func PrintLog(logMessage string) error {
	log.Println(logMessage)
	return WriteToLogFile(logMessage)
}

func GetMigrationObjectName() (string, error) {
	vmK8sName, err := GetVMwareMachineName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("migration-%s", vmK8sName), nil
}

// GetMigrationConfigMapName is function that returns the name of the secret
func GetMigrationConfigMapName() (string, error) {
	vmK8sName, err := GetVMwareMachineName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("migration-config-%s", vmK8sName), nil
}

// GetFirstbootConfigMapName is function that returns the name of the secret
func GetFirstbootConfigMapName() (string, error) {
	vmK8sName, err := GetVMwareMachineName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("firstboot-config-%s", vmK8sName), nil
}

func GetVMwareMachineName() (string, error) {
	vmK8sName := os.Getenv("VMWARE_MACHINE_OBJECT_NAME")
	if vmK8sName == "" {
		return "", errors.New("VMWARE_MACHINE_OBJECT_NAME environment variable is not set")
	}
	return vmK8sName, nil
}

func WriteToLogFile(message string) error {
	// Get migration object name from environment variable
	migrationName, err := GetMigrationObjectName()
	if err != nil {
		return errors.Wrap(err, "failed to get migration object name")
	}

	// Ensure the logs directory exists
	if err := os.MkdirAll(constants.LogsDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create logs directory")
	}

	// Create log file with the migration object name
	logFilePath := fmt.Sprintf("%s/%s.log", constants.LogsDir, migrationName)
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to create log file")
	}
	defer logFile.Close()

	logMessage := fmt.Sprintf("[%s] %s\n", time.Now().Format(time.RFC3339), message)

	// Write the log message
	if _, err := logFile.WriteString(logMessage); err != nil {
		return errors.Wrap(err, "failed to write log message to log file")
	}
	return nil
}

func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func GetRetryLimits() (uint64, time.Duration) {
	const defaultMaxRetries = 3
	const defaultInterval = 3 * time.Hour
	client, err := GetInclusterClient()
	if err != nil {
		PrintLog(fmt.Sprintf("WARNING: Failed to get in-cluster client: %v, using default max retries (%d)",
			err, defaultMaxRetries))
		return defaultMaxRetries, defaultInterval
	}
	vjailbreakSettings, err := k8sutils.GetVjailbreakSettings(context.Background(), client)
	if err != nil {
		PrintLog(fmt.Sprintf("WARNING: Failed to get vjailbreak settings: %v, using default max retries (%d)",
			err, defaultMaxRetries))
		return defaultMaxRetries, defaultInterval
	}
	retryCap, err := time.ParseDuration(vjailbreakSettings.PeriodicSyncRetryCap)
	if err != nil {
		PrintLog(fmt.Sprintf("WARNING: Failed to parse retry cap: %v, using default retry cap (%s)",
			err, defaultInterval))
		retryCap = defaultInterval
	}
	return vjailbreakSettings.PeriodicSyncMaxRetries, retryCap
}

func DoRetryWithExponentialBackoff(ctx context.Context, task func() error, maxRetries uint64, capInterval time.Duration) error {
	retries := uint64(0)
	var err error
	waitTime := 1 * time.Minute
	for retries < maxRetries {
		err = task()
		if err == nil {
			return nil
		}
		PrintLog(fmt.Sprintf("Attempt %d failed with error %v", retries, err))
		retries++
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			waitTime *= 2
			if waitTime > capInterval {
				waitTime = capInterval
			}
		}
	}
	return err
}
func GetNetworkPersistance(ctx context.Context, client client.Client) bool {
	migrationParams, err := GetMigrationParams(ctx, client)
	if err != nil {
		return false
	}
	PrintLog(fmt.Sprintf("Network persistence value from ConfigMap: %t", migrationParams.NetworkPersistance))
	return migrationParams.NetworkPersistance
}
