package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	ProgressConfigMapNamePrefix = "vjailbreak-upgrade-progress"
	Namespace                   = "migration-system"
)

type DeploymentConfig struct {
	Namespace     string
	Name          string
	ContainerName string
	ImagePrefix   string
}

var DeploymentConfigs = []DeploymentConfig{
	{
		Namespace:     Namespace,
		Name:          "migration-controller-manager",
		ContainerName: "manager",
		ImagePrefix:   "quay.io/platform9/vjailbreak-controller",
	},
	{
		Namespace:     Namespace,
		Name:          "migration-vpwned-sdk",
		ContainerName: "vpwned",
		ImagePrefix:   "quay.io/platform9/vjailbreak-vpwned",
	},
	{
		Namespace:     Namespace,
		Name:          "vjailbreak-ui",
		ContainerName: "vjailbreak-ui-container",
		ImagePrefix:   "quay.io/platform9/vjailbreak-ui",
	},
}

// UpgradeExecutor handles the upgrade process as a standalone job
type UpgradeExecutor struct {
	kubeClient    client.Client
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
	config        *rest.Config
	progress      *UpgradeProgress
	mu            sync.Mutex
}

// getProgressConfigMapName returns unique ConfigMap name based on Job UID to prevent collisions
func getProgressConfigMapName() string {
	jobUID := os.Getenv("JOB_UID")
	if jobUID != "" {
		return ProgressConfigMapNamePrefix + "-" + jobUID
	}
	return ProgressConfigMapNamePrefix
}

// NewUpgradeExecutor creates a new upgrade executor with in-cluster config
func NewUpgradeExecutor() (*UpgradeExecutor, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	config.QPS = 20
	config.Burst = 40

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))

	kubeClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &UpgradeExecutor{
		kubeClient:    kubeClient,
		clientset:     clientset,
		dynamicClient: dynamicClient,
		config:        config,
	}, nil
}

// Execute runs the upgrade process
func (e *UpgradeExecutor) Execute(ctx context.Context, targetVersion string, autoCleanup bool) error {
	log.Printf("Upgrade job started for target version: %s", targetVersion)

	existingProgress, err := e.loadProgress(ctx)
	if err == nil && existingProgress != nil {
		switch existingProgress.Status {
		case StatusInProgress, StatusDeploying, StatusVerifyingStability, StatusRollingBack:
			log.Printf("WARNING: Found existing upgrade in progress (status=%s, step=%s, target=%s)",
				existingProgress.Status, existingProgress.CurrentStep, existingProgress.TargetVersion)
			log.Printf("Job restart detected - aborting to prevent duplicate upgrade. Manual cleanup may be required.")
			return fmt.Errorf("existing upgrade in progress detected - cannot start new upgrade (use rollback or manual cleanup)")
		case StatusPending:
			log.Printf("Found pending upgrade progress from server - proceeding with upgrade")
		default:
			log.Printf("Found previous upgrade progress (status=%s) - proceeding with new upgrade", existingProgress.Status)
		}
	}

	currentVersion, err := GetCurrentVersion(ctx, e.clientset)
	if err != nil {
		log.Printf("Warning: Could not get current version: %v", err)
		currentVersion = "unknown"
	}

	e.progress = &UpgradeProgress{
		CurrentStep:     "Starting upgrade",
		TotalSteps:      TotalUpgradeSteps,
		CompletedSteps:  0,
		Status:          StatusInProgress,
		StartTime:       time.Now(),
		PreviousVersion: currentVersion,
		TargetVersion:   targetVersion,
		PodName:         os.Getenv("POD_NAME"),
		JobID:           os.Getenv("JOB_UID"),
	}
	e.saveProgress(ctx)

	log.Printf("Starting upgrade from %s to %s", currentVersion, targetVersion)

	if err := e.runPreUpgradePhase(ctx, targetVersion, autoCleanup); err != nil {
		return e.handleFailure(ctx, err, "Pre-upgrade phase failed")
	}

	backupID, err := e.runBackupAndCRDPhase(ctx, targetVersion)
	if err != nil {
		return e.handleFailure(ctx, err, "Backup/CRD phase failed")
	}

	if err := e.runConfigMapPhase(ctx, targetVersion); err != nil {
		log.Printf("Warning: ConfigMap update had issues: %v", err)
	}

	if err := e.runDeploymentPhase(ctx, targetVersion, backupID); err != nil {
		return e.handleFailure(ctx, err, "Deployment phase failed")
	}

	log.Printf("Upgrade completed successfully")
	return nil
}

func (e *UpgradeExecutor) runPreUpgradePhase(ctx context.Context, targetVersion string, autoCleanup bool) error {
	if targetVersion == "" {
		return fmt.Errorf("targetVersion cannot be empty")
	}

	e.updateProgress("Running pre-upgrade checks", StatusInProgress, "")
	result, err := RunPreUpgradeChecks(ctx, e.kubeClient, e.dynamicClient, targetVersion)
	if err != nil {
		return fmt.Errorf("pre-upgrade checks failed: %w", err)
	}

	allPassed := result.NoMigrationPlans && result.NoRollingMigrationPlans &&
		result.VMwareCredsDeleted && result.OpenStackCredsDeleted &&
		result.AgentsScaledDown && result.NoCustomResources

	if !allPassed {
		if autoCleanup {
			log.Println("Pre-upgrade checks failed, attempting automatic cleanup...")
			e.updateProgress("Cleaning up resources", StatusInProgress, "")
			if err := CleanupResources(ctx, e.kubeClient, e.config); err != nil {
				return fmt.Errorf("auto-cleanup failed: %w", err)
			}
			result, err = RunPreUpgradeChecks(ctx, e.kubeClient, e.dynamicClient, targetVersion)
			if err != nil {
				return fmt.Errorf("pre-upgrade checks failed after cleanup: %w", err)
			}
			allPassed = result.NoMigrationPlans && result.NoRollingMigrationPlans &&
				result.VMwareCredsDeleted && result.OpenStackCredsDeleted &&
				result.AgentsScaledDown && result.NoCustomResources
			if !allPassed {
				return fmt.Errorf("pre-upgrade checks still failing after cleanup: migrations=%t, rollingMigrations=%t, vmwareCreds=%t, openstackCreds=%t, agents=%t, customResources=%t",
					result.NoMigrationPlans, result.NoRollingMigrationPlans,
					result.VMwareCredsDeleted, result.OpenStackCredsDeleted,
					result.AgentsScaledDown, result.NoCustomResources)
			}
		} else {
			return fmt.Errorf("pre-upgrade checks failed: migrations=%t, rollingMigrations=%t, vmwareCreds=%t, openstackCreds=%t, agents=%t, customResources=%t",
				result.NoMigrationPlans, result.NoRollingMigrationPlans,
				result.VMwareCredsDeleted, result.OpenStackCredsDeleted,
				result.AgentsScaledDown, result.NoCustomResources)
		}
	}

	log.Println("Pre-upgrade checks passed")
	e.updateProgress("Pre-upgrade checks passed", StatusInProgress, "")
	e.incrementCompletedSteps()
	e.saveProgress(ctx)
	return nil
}

func (e *UpgradeExecutor) runBackupAndCRDPhase(ctx context.Context, targetVersion string) (string, error) {
	backupID := time.Now().UTC().Format("20060102T150405Z")

	e.setBackupID(backupID)
	e.updateProgress("Backing up resources", StatusInProgress, "")

	if err := BackupResourcesWithID(ctx, e.kubeClient, e.config, backupID); err != nil {
		return "", fmt.Errorf("backup failed: %w", err)
	}

	e.incrementCompletedSteps()

	e.updateProgress("Updating Custom Resource Definitions", StatusInProgress, "")

	if err := ApplyAllCRDs(ctx, e.kubeClient, targetVersion); err != nil {
		return "", fmt.Errorf("CRD update failed: %w", err)
	}

	e.incrementCompletedSteps()
	e.saveProgress(ctx)

	return backupID, nil
}

func (e *UpgradeExecutor) runConfigMapPhase(ctx context.Context, targetVersion string) error {
	e.updateProgress("Updating version configuration", StatusInProgress, "")

	if err := UpdateVersionConfigMapFromGitHub(ctx, e.kubeClient, targetVersion); err != nil {
		log.Printf("Warning: Failed to update version-config ConfigMap from GitHub: %v", err)
	}

	e.updateProgress("Updating vjailbreak settings", StatusInProgress, "")

	if err := UpdateVjailbreakSettingsFromGitHub(ctx, e.kubeClient, targetVersion); err != nil {
		log.Printf("Warning: Failed to update vjailbreak-settings ConfigMap from GitHub: %v", err)
	}

	e.incrementCompletedSteps()
	e.saveProgress(ctx)
	return nil
}

func (e *UpgradeExecutor) runDeploymentPhase(ctx context.Context, targetVersion, backupID string) error {
	e.updateProgress("Upgrading deployments", StatusDeploying, "")
	e.saveProgress(ctx)

	var controllerConfig DeploymentConfig
	for _, cfg := range DeploymentConfigs {
		if cfg.Name == "migration-controller-manager" {
			controllerConfig = cfg
			break
		}
	}
	if controllerConfig.Name == "" {
		return fmt.Errorf("controller deployment config not found")
	}

	phaseStart := time.Now()
	originalReplicas, err := e.scaleDeployment(ctx, controllerConfig, 0, "Scaling down controller")
	if err != nil {
		return err
	}
	e.setOriginalReplicas(controllerConfig.Name, originalReplicas)

	e.updateProgress("Waiting for controller to scale down", StatusDeploying, "")
	if err := e.waitForDeploymentScaledDown(ctx, controllerConfig); err != nil {
		return err
	}
	e.incrementCompletedSteps()
	e.saveProgress(ctx)

	e.updateProgress("Applying controller deployment from GitHub", StatusDeploying, "")
	if err := ApplyManifestFromGitHub(ctx, e.kubeClient, targetVersion, "deploy/05controller-deployment.yaml"); err != nil {
		return fmt.Errorf("failed to apply controller deployment from GitHub: %w", err)
	}

	e.incrementCompletedSteps()
	e.saveProgress(ctx)

	if _, err := e.scaleDeployment(ctx, controllerConfig, originalReplicas, "Scaling up controller"); err != nil {
		return err
	}

	e.updateProgress("Waiting for controller to be ready", StatusDeploying, "")
	if err := e.waitForDeploymentReady(ctx, controllerConfig); err != nil {
		return fmt.Errorf("controller deployment failed readiness check: %w", err)
	}
	e.incrementCompletedSteps()
	e.saveProgress(ctx)

	e.updateProgress("Applying vpwned deployment from GitHub", StatusDeploying, "")
	if err := ApplyManifestFromGitHub(ctx, e.kubeClient, targetVersion, "deploy/06vpwned-deployment.yaml"); err != nil {
		return fmt.Errorf("failed to apply vpwned deployment: %w", err)
	}
	e.incrementCompletedSteps()
	e.saveProgress(ctx)

	e.updateProgress("Applying UI deployment from GitHub", StatusDeploying, "")
	if err := ApplyManifestFromGitHub(ctx, e.kubeClient, targetVersion, "deploy/07ui-deployment.yaml"); err != nil {
		return fmt.Errorf("failed to apply UI deployment: %w", err)
	}
	e.incrementCompletedSteps()
	e.saveProgress(ctx)

	e.updateProgress("Waiting for all deployments to be ready", StatusDeploying, "")
	for _, cfg := range []DeploymentConfig{
		controllerConfig,
		{Name: "migration-vpwned-sdk", Namespace: Namespace},
		{Name: "vjailbreak-ui", Namespace: Namespace},
	} {
		if err := e.waitForDeploymentReady(ctx, cfg); err != nil {
			return fmt.Errorf("deployment %s not ready: %w", cfg.Name, err)
		}
	}

	e.updateProgress("Verifying upgrade stability", StatusVerifyingStability, "")

	ok := true
	for _, cfg := range DeploymentConfigs {
		if err := e.waitForDeploymentReady(ctx, cfg); err != nil {
			log.Printf("Stability check failed for deployment %s: %v", cfg.Name, err)
			ok = false
			break
		}
	}

	e.incrementCompletedSteps()
	e.saveProgress(ctx)

	if ok {
		log.Println("Upgrade looks stable. Cleaning up backup ConfigMaps...")
		if err := CleanupBackupConfigMaps(ctx, e.kubeClient, backupID); err != nil {
			log.Printf("Warning: Failed to cleanup backup ConfigMaps: %v", err)
		} else {
			log.Println("Backup ConfigMaps cleaned up.")
		}

		if err := CleanupAllOldBackups(ctx, e.kubeClient, backupID); err != nil {
			log.Printf("Warning: Failed to cleanup old backup ConfigMaps: %v", err)
		}

		e.incrementCompletedSteps()
		e.recordPhaseTiming("deployment_phase", time.Since(phaseStart))
		e.setEndTime(time.Now())
		e.setResult("success")
		e.updateProgress("Upgrade completed successfully", StatusCompleted, "")
		e.saveProgress(ctx)
		return nil
	}

	log.Println("Post-upgrade stability checks failed; keeping backups for investigation.")
	e.recordPhaseTiming("deployment_phase", time.Since(phaseStart))
	e.setEndTime(time.Now())
	e.setResult("failure")
	e.updateProgress("Deployments not stable after upgrade", StatusFailed, "Deployments are unstable; backups retained for investigation")
	e.saveProgress(ctx)
	return fmt.Errorf("upgrade completed but deployments are unstable")
}

func (e *UpgradeExecutor) scaleDeployment(ctx context.Context, cfg DeploymentConfig, replicas int32, step string) (int32, error) {
	e.updateProgress(step, StatusDeploying, "")

	var originalReplicas int32 = 1
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dep := &appsv1.Deployment{}
		if getErr := e.kubeClient.Get(ctx, client.ObjectKey{Name: cfg.Name, Namespace: cfg.Namespace}, dep); getErr != nil {
			return getErr
		}
		if dep.Spec.Replicas != nil {
			originalReplicas = *dep.Spec.Replicas
		}
		dep.Spec.Replicas = &replicas
		return e.kubeClient.Update(ctx, dep)
	}); err != nil {
		return 0, fmt.Errorf("failed to scale deployment %s: %w", cfg.Name, err)
	}

	return originalReplicas, nil
}

func (e *UpgradeExecutor) waitForDeploymentReady(ctx context.Context, cfg DeploymentConfig) error {
	timeout := 5 * time.Minute
	interval := 10 * time.Second

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	deadline := time.Now().Add(timeout)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("deployment %s not ready within timeout", cfg.Name)
			}
		}

		dep := &appsv1.Deployment{}
		if err := e.kubeClient.Get(ctx, client.ObjectKey{Name: cfg.Name, Namespace: cfg.Namespace}, dep); err != nil {
			if kerrors.IsNotFound(err) {
				return fmt.Errorf("deployment %s not found", cfg.Name)
			}
			return fmt.Errorf("failed to get deployment %s: %w", cfg.Name, err)
		}

		desiredReplicas := int32(1)
		if dep.Spec.Replicas != nil {
			desiredReplicas = *dep.Spec.Replicas
		}

		replicasReady := dep.Status.ReadyReplicas == desiredReplicas && dep.Status.UpdatedReplicas == desiredReplicas

		generationMatches := dep.Status.ObservedGeneration >= dep.Generation

		availableCondition := false
		for _, cond := range dep.Status.Conditions {
			if cond.Type == appsv1.DeploymentAvailable && cond.Status == corev1.ConditionTrue {
				availableCondition = true
				break
			}
		}

		if replicasReady && availableCondition && generationMatches {
			return nil
		}

		log.Printf("Waiting for deployment %s: ready=%d/%d updated=%d available=%v generation=%d/%d",
			cfg.Name, dep.Status.ReadyReplicas, desiredReplicas, dep.Status.UpdatedReplicas, availableCondition,
			dep.Status.ObservedGeneration, dep.Generation)

		podList := &corev1.PodList{}
		podLabels := client.MatchingLabels{}
		if dep.Spec.Selector != nil && dep.Spec.Selector.MatchLabels != nil {
			podLabels = client.MatchingLabels(dep.Spec.Selector.MatchLabels)
		}
		if err := e.kubeClient.List(ctx, podList, client.InNamespace(cfg.Namespace), podLabels); err == nil {
			for _, pod := range podList.Items {
				log.Printf("  Pod %s: phase=%s", pod.Name, pod.Status.Phase)

				for _, cs := range pod.Status.ContainerStatuses {
					if cs.State.Waiting != nil {
						log.Printf("    Container %s waiting: reason=%s message=%s",
							cs.Name, cs.State.Waiting.Reason, cs.State.Waiting.Message)
					}
					if cs.State.Terminated != nil {
						log.Printf("    Container %s terminated: reason=%s message=%s exitCode=%d",
							cs.Name, cs.State.Terminated.Reason, cs.State.Terminated.Message, cs.State.Terminated.ExitCode)
					}
					if cs.LastTerminationState.Terminated != nil {
						log.Printf("    Container %s last termination: reason=%s message=%s exitCode=%d",
							cs.Name, cs.LastTerminationState.Terminated.Reason,
							cs.LastTerminationState.Terminated.Message, cs.LastTerminationState.Terminated.ExitCode)
					}
					if !cs.Ready {
						log.Printf("    Container %s not ready: restartCount=%d", cs.Name, cs.RestartCount)
					}
				}

				for _, cond := range pod.Status.Conditions {
					if cond.Status != corev1.ConditionTrue {
						log.Printf("    Condition %s=%s: reason=%s message=%s",
							cond.Type, cond.Status, cond.Reason, cond.Message)
					}
				}
			}
		}
	}
}

func (e *UpgradeExecutor) waitForDeploymentScaledDown(ctx context.Context, cfg DeploymentConfig) error {
	timeout := 5 * time.Minute
	interval := 10 * time.Second

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	deadline := time.Now().Add(timeout)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("deployment %s not scaled down within timeout", cfg.Name)
			}
		}

		dep := &appsv1.Deployment{}
		if err := e.kubeClient.Get(ctx, client.ObjectKey{Name: cfg.Name, Namespace: cfg.Namespace}, dep); err != nil {
			if kerrors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("failed to get deployment %s: %w", cfg.Name, err)
		}

		log.Printf("Waiting for deployment %s to scale down: replicas=%d, readyReplicas=%d", cfg.Name, dep.Status.Replicas, dep.Status.ReadyReplicas)

		if dep.Status.Replicas == 0 && dep.Status.ReadyReplicas == 0 {
			log.Printf("Deployment %s successfully scaled down.", cfg.Name)
			return nil
		}
	}
}

func (e *UpgradeExecutor) handleFailure(ctx context.Context, err error, message string) error {
	log.Printf("%s: %v. Rolling back.", message, err)

	e.setEndTime(time.Now())
	e.setResult("failure")
	e.updateProgress("Upgrade failed, rolling back...", StatusFailed, err.Error())
	e.saveProgress(ctx)

	e.mu.Lock()
	previousVersion := e.progress.PreviousVersion
	e.mu.Unlock()
	if previousVersion != "" && previousVersion != "unknown" {
		if restoreErr := UpdateVersionConfigMapFromGitHub(ctx, e.kubeClient, previousVersion); restoreErr != nil {
			log.Printf("Warning: Failed to restore version-config: %v", restoreErr)
		} else {
			log.Printf("Restored version-config to previous version: %s", previousVersion)
		}
	}

	if rollbackErr := RestoreResources(ctx, e.kubeClient, e.progress.BackupID); rollbackErr != nil {
		log.Printf("CRITICAL: Rollback failed: %v", rollbackErr)
		e.updateProgress("Deployment failed and rollback also failed", StatusRollbackFailed, err.Error())
	} else {
		log.Printf("Rollback completed successfully.")
		e.updateProgress("Rolled back due to failure", StatusRolledBack, err.Error())
	}
	e.saveProgress(ctx)

	return err
}

func (e *UpgradeExecutor) updateProgress(step, status, errMsg string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.progress.CurrentStep = step
	e.progress.Status = status
	if errMsg != "" {
		e.progress.Error = errMsg
	}
}

func (e *UpgradeExecutor) incrementCompletedSteps() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.progress.CompletedSteps++
}

func (e *UpgradeExecutor) setEndTime(t time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.progress.EndTime = &t
}

func (e *UpgradeExecutor) setBackupID(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.progress.BackupID = id
}

func (e *UpgradeExecutor) setResult(result string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.progress.Result = result
}

func (e *UpgradeExecutor) setOriginalReplicas(deploymentName string, replicas int32) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.progress.OriginalReplicas == nil {
		e.progress.OriginalReplicas = make(map[string]int32)
	}
	e.progress.OriginalReplicas[deploymentName] = replicas
}

func (e *UpgradeExecutor) getOriginalReplicas(deploymentName string) (int32, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.progress.OriginalReplicas == nil {
		return 0, false
	}
	replicas, ok := e.progress.OriginalReplicas[deploymentName]
	return replicas, ok
}

func (e *UpgradeExecutor) recordPhaseTiming(phaseName string, duration time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.progress.PhaseTimings == nil {
		e.progress.PhaseTimings = make(map[string]string)
	}
	e.progress.PhaseTimings[phaseName] = duration.String()
}

func (e *UpgradeExecutor) saveProgress(ctx context.Context) {
	e.mu.Lock()
	if e.progress == nil {
		e.mu.Unlock()
		return
	}
	progressCopy := *e.progress
	e.mu.Unlock()

	progressJSON, err := json.Marshal(progressCopy)
	if err != nil {
		log.Printf("Error marshaling progress: %v", err)
		return
	}

	key := client.ObjectKey{Name: getProgressConfigMapName(), Namespace: Namespace}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing := &corev1.ConfigMap{}
		if err := e.kubeClient.Get(ctx, key, existing); err == nil {
			if existing.Data == nil {
				existing.Data = map[string]string{}
			}
			existing.Data["progress"] = string(progressJSON)
			return e.kubeClient.Update(ctx, existing)
		} else if kerrors.IsNotFound(err) {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      getProgressConfigMapName(),
					Namespace: Namespace,
					Labels: map[string]string{
						"app": "vjailbreak-upgrade",
					},
				},
				Data: map[string]string{"progress": string(progressJSON)},
			}
			return e.kubeClient.Create(ctx, cm)
		} else {
			return err
		}
	}); err != nil {
		log.Printf("Error saving upgrade progress ConfigMap: %v", err)
	}
}

func (e *UpgradeExecutor) loadProgress(ctx context.Context) (*UpgradeProgress, error) {
	cm := &corev1.ConfigMap{}
	key := client.ObjectKey{Name: getProgressConfigMapName(), Namespace: Namespace}
	if err := e.kubeClient.Get(ctx, key, cm); err != nil {
		return nil, err
	}

	progressData, ok := cm.Data["progress"]
	if !ok {
		return nil, fmt.Errorf("progress key not found in ConfigMap")
	}

	var progress UpgradeProgress
	if err := json.Unmarshal([]byte(progressData), &progress); err != nil {
		return nil, fmt.Errorf("failed to unmarshal progress: %w", err)
	}

	return &progress, nil
}

func (e *UpgradeExecutor) ExecuteRollback(ctx context.Context, previousVersion, targetVersion, backupID string) error {
	log.Printf("Rollback job started: %s -> %s (backupID=%s)", targetVersion, previousVersion, backupID)

	existingBackupID := backupID
	if existingBackupID == "" {
		existingProgress, err := e.loadProgress(ctx)
		if err == nil && existingProgress != nil && existingProgress.BackupID != "" {
			existingBackupID = existingProgress.BackupID
			log.Printf("Loaded BackupID from existing progress: %s", existingBackupID)
		}
	}

	e.progress = &UpgradeProgress{
		CurrentStep:     "Starting rollback",
		TotalSteps:      TotalRollbackSteps,
		CompletedSteps:  0,
		Status:          StatusRollingBack,
		StartTime:       time.Now(),
		PreviousVersion: previousVersion,
		TargetVersion:   targetVersion,
		BackupID:        existingBackupID,
	}
	e.saveProgress(ctx)

	if previousVersion == "" || previousVersion == "unknown" {
		if e.progress.BackupID == "" {
			err := fmt.Errorf("rollback requires either valid previousVersion for manifest-driven rollback OR backupID for snapshot-based rollback; previousVersion=%q, backupID=%q", previousVersion, e.progress.BackupID)
			e.setEndTime(time.Now())
			e.updateProgress("Rollback failed", StatusRollbackFailed, err.Error())
			e.saveProgress(ctx)
			return err
		}

		log.Printf("Performing explicit snapshot-based rollback with backupID=%s", e.progress.BackupID)
		e.updateProgress("Restoring resources from backup snapshot", StatusRollingBack, "")

		if err := RestoreResources(ctx, e.kubeClient, e.progress.BackupID); err != nil {
			e.setEndTime(time.Now())
			e.updateProgress("Rollback failed", StatusRollbackFailed, err.Error())
			e.saveProgress(ctx)
			return err
		}

		e.progress.CompletedSteps = TotalRollbackSteps
		e.setEndTime(time.Now())
		e.setResult("success")
		e.updateProgress("Rollback completed successfully (snapshot-based)", StatusRolledBack, "")
		e.saveProgress(ctx)
		return nil
	}

	var controllerConfig DeploymentConfig
	for _, cfg := range DeploymentConfigs {
		if cfg.Name == "migration-controller-manager" {
			controllerConfig = cfg
			break
		}
	}

	var originalReplicas int32 = 1
	if stored, ok := e.getOriginalReplicas(controllerConfig.Name); ok {
		originalReplicas = stored
		log.Printf("Using stored original replicas for %s: %d", controllerConfig.Name, originalReplicas)
	} else {
		readReplicas, err := e.scaleDeployment(ctx, controllerConfig, 0, "Scaling down controller for rollback")
		if err != nil {
			return e.handleRollbackFailure(ctx, err)
		}
		originalReplicas = readReplicas
	}

	if _, err := e.scaleDeployment(ctx, controllerConfig, 0, "Scaling down controller for rollback"); err != nil {
		return e.handleRollbackFailure(ctx, err)
	}

	if err := e.waitForDeploymentScaledDown(ctx, controllerConfig); err != nil {
		return e.handleRollbackFailure(ctx, err)
	}
	e.incrementCompletedSteps()
	e.saveProgress(ctx)

	e.updateProgress("Restoring CRDs from previous version", StatusRollingBack, "")
	if err := ApplyAllCRDs(ctx, e.kubeClient, previousVersion); err != nil {
		log.Printf("Warning: Failed to restore CRDs: %v", err)
	}

	log.Println("Waiting for CRDs to be established...")
	if err := waitForCRDEstablished(ctx, e.kubeClient, 2*time.Minute); err != nil {
		log.Printf("Warning: CRDs may not be fully established: %v", err)
	}
	e.incrementCompletedSteps()
	e.saveProgress(ctx)

	e.updateProgress("Restoring controller deployment from GitHub", StatusRollingBack, "")
	if err := ApplyManifestFromGitHub(ctx, e.kubeClient, previousVersion, "deploy/05controller-deployment.yaml"); err != nil {
		return e.handleRollbackFailure(ctx, err)
	}

	e.updateProgress("Restoring vpwned deployment from GitHub", StatusRollingBack, "")
	if err := ApplyManifestFromGitHub(ctx, e.kubeClient, previousVersion, "deploy/06vpwned-deployment.yaml"); err != nil {
		log.Printf("Warning: Failed to restore vpwned deployment: %v", err)
	}

	e.updateProgress("Restoring UI deployment from GitHub", StatusRollingBack, "")
	if err := ApplyManifestFromGitHub(ctx, e.kubeClient, previousVersion, "deploy/07ui-deployment.yaml"); err != nil {
		log.Printf("Warning: Failed to restore UI deployment: %v", err)
	}
	e.incrementCompletedSteps()
	e.saveProgress(ctx)

	if _, err := e.scaleDeployment(ctx, controllerConfig, originalReplicas, "Scaling up controller"); err != nil {
		return e.handleRollbackFailure(ctx, err)
	}

	if err := e.waitForDeploymentReady(ctx, controllerConfig); err != nil {
		return e.handleRollbackFailure(ctx, err)
	}

	e.updateProgress("Waiting for all deployments to be ready", StatusRollingBack, "")
	for _, cfg := range []DeploymentConfig{
		controllerConfig,
		{Name: "migration-vpwned-sdk", Namespace: Namespace},
		{Name: "vjailbreak-ui", Namespace: Namespace},
	} {
		if err := e.waitForDeploymentReady(ctx, cfg); err != nil {
			log.Printf("Warning: Deployment %s not ready: %v", cfg.Name, err)
		}
	}
	e.incrementCompletedSteps()
	e.saveProgress(ctx)

	e.updateProgress("Restoring version configuration", StatusRollingBack, "")
	if err := UpdateVersionConfigMapFromGitHub(ctx, e.kubeClient, previousVersion); err != nil {
		log.Printf("Warning: Failed to restore version-config: %v", err)
	}

	e.updateProgress("Restoring vjailbreak settings", StatusRollingBack, "")
	if err := UpdateVjailbreakSettingsFromGitHub(ctx, e.kubeClient, previousVersion); err != nil {
		log.Printf("Warning: Failed to restore vjailbreak-settings: %v", err)
	}
	e.incrementCompletedSteps()
	e.saveProgress(ctx)

	e.setEndTime(time.Now())
	e.updateProgress("Rollback completed successfully", StatusRolledBack, "")
	e.saveProgress(ctx)

	log.Printf("Manifest-driven rollback completed: %s -> %s", targetVersion, previousVersion)
	return nil
}

func (e *UpgradeExecutor) handleRollbackFailure(ctx context.Context, err error) error {
	e.setEndTime(time.Now())
	e.updateProgress("Rollback failed", StatusRollbackFailed, err.Error())
	e.saveProgress(ctx)
	return err
}
