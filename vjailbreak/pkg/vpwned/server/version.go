package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	api "github.com/platform9/vjailbreak/pkg/vpwned/api/proto/v1/service"
	"github.com/platform9/vjailbreak/pkg/vpwned/upgrade"
	version "github.com/platform9/vjailbreak/pkg/vpwned/version"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpgradeProgress is an alias to the shared type in upgrade package
type UpgradeProgress = upgrade.UpgradeProgress

const (
	progressConfigMapName    = "vjailbreak-upgrade-progress"
	upgradeJobName           = "vjailbreak-upgrade-job"
	rollbackJobName          = "vjailbreak-rollback-job"
	namespace                = "migration-system"
	upgradeJobServiceAccount = "migration-controller-manager"
)

// DeploymentConfig holds deployment information for upgrade helpers
type DeploymentConfig struct {
	Namespace string
	Name      string
}

// saveProgressToConfigMap saves progress directly to ConfigMap (stateless) with retry on conflict
func saveProgressToConfigMap(ctx context.Context, kubeClient client.Client, progress *UpgradeProgress) error {
	progressJSON, err := json.Marshal(progress)
	if err != nil {
		return fmt.Errorf("failed to marshal progress: %w", err)
	}

	key := client.ObjectKey{Name: progressConfigMapName, Namespace: namespace}

	// Use retry to handle conflicts
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing := &corev1.ConfigMap{}
		if err := kubeClient.Get(ctx, key, existing); err == nil {
			// Update existing ConfigMap - preserve other keys
			if existing.Data == nil {
				existing.Data = map[string]string{}
			}
			existing.Data["progress"] = string(progressJSON)
			return kubeClient.Update(ctx, existing)
		} else if kerrors.IsNotFound(err) {
			// Create new ConfigMap with labels
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      progressConfigMapName,
					Namespace: namespace,
					Labels: map[string]string{
						"app": "vjailbreak-upgrade",
					},
				},
				Data: map[string]string{"progress": string(progressJSON)},
			}
			return kubeClient.Create(ctx, cm)
		} else {
			return err
		}
	})
}

// updateProgressStatusOnly updates only status, error, and endTime fields without overwriting job-written fields
func updateProgressStatusOnly(ctx context.Context, kubeClient client.Client, status, errMsg string, endTime *time.Time) error {
	key := client.ObjectKey{Name: progressConfigMapName, Namespace: namespace}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cm := &corev1.ConfigMap{}
		if err := kubeClient.Get(ctx, key, cm); err != nil {
			return err
		}

		progressJSON, ok := cm.Data["progress"]
		if !ok {
			return fmt.Errorf("progress key not found in ConfigMap")
		}

		var progress UpgradeProgress
		if err := json.Unmarshal([]byte(progressJSON), &progress); err != nil {
			return err
		}

		progress.Status = status
		if errMsg != "" {
			progress.Error = errMsg
		}
		if endTime != nil {
			progress.EndTime = endTime
		}

		updatedJSON, err := json.Marshal(progress)
		if err != nil {
			return err
		}
		cm.Data["progress"] = string(updatedJSON)
		return kubeClient.Update(ctx, cm)
	})
}

// loadProgressFromConfigMap loads progress directly from ConfigMap (stateless)
func loadProgressFromConfigMap(ctx context.Context, kubeClient client.Client) (*UpgradeProgress, error) {
	cm := &corev1.ConfigMap{}
	err := kubeClient.Get(ctx, client.ObjectKey{Name: progressConfigMapName, Namespace: namespace}, cm)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if progressJSON, ok := cm.Data["progress"]; ok {
		var progress UpgradeProgress
		if err := json.Unmarshal([]byte(progressJSON), &progress); err != nil {
			return nil, err
		}
		return &progress, nil
	}
	return nil, nil
}

type VpwnedVersion struct {
	api.UnimplementedVersionServer
}

func (s *VpwnedVersion) Version(ctx context.Context, in *api.VersionRequest) (*api.VersionResponse, error) {
	return &api.VersionResponse{Version: version.Version}, nil
}

func (s *VpwnedVersion) GetAvailableTags(ctx context.Context, in *api.VersionRequest) (*api.AvailableUpdatesResponse, error) {
	tags, err := upgrade.GetAllTags(ctx)
	if err != nil {
		log.Printf("Error fetching tags: %v", err)
		return nil, err
	}

	log.Printf("Found %d available tags", len(tags))

	var protoUpdates []*api.ReleaseInfo
	for _, tag := range tags {
		protoUpdates = append(protoUpdates, &api.ReleaseInfo{
			Version:      tag,
			ReleaseNotes: "",
		})
	}
	return &api.AvailableUpdatesResponse{Updates: protoUpdates}, nil
}

func (s *VpwnedVersion) InitiateUpgrade(ctx context.Context, in *api.UpgradeRequest) (*api.UpgradeResponse, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(batchv1.AddToScheme(scheme))
	kubeClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	// Check if an upgrade job is already running
	existingJob := &batchv1.Job{}
	if err := kubeClient.Get(ctx, client.ObjectKey{Name: upgradeJobName, Namespace: namespace}, existingJob); err == nil {
		if existingJob.Status.Active > 0 {
			return nil, fmt.Errorf("upgrade job already running")
		}
		// Delete completed/failed job - poll until deleted
		if err := kubeClient.Delete(ctx, existingJob); err != nil && !kerrors.IsNotFound(err) {
			log.Printf("Warning: Failed to delete old upgrade job: %v", err)
		}
		// Poll until job is deleted (max 30 seconds) with proper error handling
		jobDeleted := false
		for i := 0; i < 30; i++ {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			err := kubeClient.Get(ctx, client.ObjectKey{Name: upgradeJobName, Namespace: namespace}, existingJob)
			if kerrors.IsNotFound(err) {
				jobDeleted = true
				break
			}
			if err != nil {
				log.Printf("Warning: Error checking job deletion: %v", err)
			}
			time.Sleep(1 * time.Second)
		}
		if !jobDeleted {
			return nil, fmt.Errorf("old upgrade job still exists after 30s - cannot start new upgrade")
		}
	}

	currentVersion, err := upgrade.GetCurrentVersion(ctx, clientset)
	if err != nil {
		log.Printf("Warning: Could not get current version: %v", err)
		currentVersion = "unknown"
	}

	// Create dynamic client for pre-upgrade checks
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Run pre-upgrade checks synchronously
	var preUpgradeChecks *upgrade.ValidationResult
	checks, err := upgrade.RunPreUpgradeChecks(ctx, kubeClient, dynamicClient, in.TargetVersion)
	if err != nil {
		return nil, fmt.Errorf("pre-upgrade checks failed: %w", err)
	}

	if !checks.PassedAll && in.AutoCleanup {
		if err := upgrade.CleanupResources(ctx, kubeClient, config); err != nil {
			return nil, fmt.Errorf("automatic cleanup failed: %w", err)
		}
		checks, err = upgrade.RunPreUpgradeChecks(ctx, kubeClient, dynamicClient, in.TargetVersion)
		if err != nil || !checks.PassedAll {
			return nil, fmt.Errorf("checks failed even after cleanup")
		}
	}

	if !checks.PassedAll {
		return &api.UpgradeResponse{
			Checks: &api.ValidationResult{
				NoMigrationPlans:        checks.NoMigrationPlans,
				NoRollingMigrationPlans: checks.NoRollingMigrationPlans,
				VmwareCredsDeleted:      checks.VMwareCredsDeleted,
				OpenstackCredsDeleted:   checks.OpenStackCredsDeleted,
				AgentsScaledDown:        checks.AgentsScaledDown,
				NoCustomResources:       checks.NoCustomResources,
				PassedAll:               false,
			},
			UpgradeStarted:  false,
			CleanupRequired: true,
		}, nil
	}
	preUpgradeChecks = checks

	// Verify images exist
	ok, imageErr := upgrade.CheckImagesExist(ctx, in.TargetVersion)
	if imageErr != nil {
		return nil, fmt.Errorf("image validation failed: %w", imageErr)
	}
	if !ok {
		return nil, fmt.Errorf("image validation failed: images not found for version %s", in.TargetVersion)
	}

	progress := &UpgradeProgress{
		CurrentStep:     "Creating upgrade job",
		TotalSteps:      upgrade.TotalUpgradeSteps,
		CompletedSteps:  0,
		Status:          "pending",
		StartTime:       time.Now(),
		PreviousVersion: currentVersion,
		TargetVersion:   in.TargetVersion,
	}
	if err := saveProgressToConfigMap(ctx, kubeClient, progress); err != nil {
		log.Printf("Warning: failed to save initial progress: %v", err)
	}

	log.Printf("Starting upgrade from %s to %s via Job", currentVersion, in.TargetVersion)

	// Create the upgrade job
	if err := createUpgradeJob(ctx, kubeClient, in.TargetVersion, currentVersion, in.AutoCleanup); err != nil {
		now := time.Now()
		progress.Status = "failed"
		progress.Error = fmt.Sprintf("Failed to create upgrade job: %v", err)
		progress.EndTime = &now
		_ = saveProgressToConfigMap(ctx, kubeClient, progress)
		return nil, fmt.Errorf("failed to create upgrade job: %w", err)
	}

	protoChecks := &api.ValidationResult{}
	if preUpgradeChecks != nil {
		protoChecks = &api.ValidationResult{
			NoMigrationPlans:        preUpgradeChecks.NoMigrationPlans,
			NoRollingMigrationPlans: preUpgradeChecks.NoRollingMigrationPlans,
			VmwareCredsDeleted:      preUpgradeChecks.VMwareCredsDeleted,
			OpenstackCredsDeleted:   preUpgradeChecks.OpenStackCredsDeleted,
			AgentsScaledDown:        preUpgradeChecks.AgentsScaledDown,
			NoCustomResources:       preUpgradeChecks.NoCustomResources,
			PassedAll:               preUpgradeChecks.PassedAll,
		}
	}

	return &api.UpgradeResponse{
		Checks:         protoChecks,
		UpgradeStarted: true,
	}, nil
}

// createUpgradeJob creates a Kubernetes Job to run the upgrade
func createUpgradeJob(ctx context.Context, kubeClient client.Client, targetVersion, currentVersion string, autoCleanup bool) error {
	vpwnedImage, err := getCurrentVpwnedImage(ctx, kubeClient)
	if err != nil {
		return fmt.Errorf("failed to get current vpwned image: %w", err)
	}

	backoffLimit := int32(0)
	ttlSeconds := int32(86400)
	activeDeadlineSeconds := int64(3600)

	autoCleanupStr := "false"
	if autoCleanup {
		autoCleanupStr = "true"
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      upgradeJobName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":                          "vjailbreak-upgrade",
				"vjailbreak.k8s.pf9.io/job":    "upgrade",
				"vjailbreak.k8s.pf9.io/target": targetVersion,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSeconds,
			ActiveDeadlineSeconds:   &activeDeadlineSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                       "vjailbreak-upgrade",
						"vjailbreak.k8s.pf9.io/job": "upgrade",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: upgradeJobServiceAccount,
					Containers: []corev1.Container{
						{
							Name:            "upgrade",
							Image:           vpwnedImage,
							ImagePullPolicy: corev1.PullAlways,
							Command:         []string{"./vpwctl", "upgrade-job"},
							Env: []corev1.EnvVar{
								{
									Name:  "UPGRADE_TARGET_VERSION",
									Value: targetVersion,
								},
								{
									Name:  "UPGRADE_PREVIOUS_VERSION",
									Value: currentVersion,
								},
								{
									Name:  "UPGRADE_AUTO_CLEANUP",
									Value: autoCleanupStr,
								},
								{
									Name:  "UPGRADE_MODE",
									Value: "upgrade",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	if err := kubeClient.Create(ctx, job); err != nil {
		return fmt.Errorf("failed to create upgrade job: %w", err)
	}

	log.Printf("Created upgrade job %s for version %s", upgradeJobName, targetVersion)
	return nil
}

// getCurrentVpwnedImage returns the CURRENT vpwned image
func getCurrentVpwnedImage(ctx context.Context, kubeClient client.Client) (string, error) {
	dep := &appsv1.Deployment{}
	if err := kubeClient.Get(ctx, client.ObjectKey{Name: "migration-vpwned-sdk", Namespace: namespace}, dep); err != nil {
		return "", fmt.Errorf("cannot get vpwned deployment: %w", err)
	}

	for _, container := range dep.Spec.Template.Spec.Containers {
		if container.Name == "vpwned" {
			return container.Image, nil
		}
	}

	if len(dep.Spec.Template.Spec.Containers) > 0 {
		log.Printf("Warning: container 'vpwned' not found, using first container image")
		return dep.Spec.Template.Spec.Containers[0].Image, nil
	}

	return "", fmt.Errorf("vpwned deployment has no containers")
}

// createRollbackJob creates a Kubernetes Job to run the rollback
func createRollbackJob(ctx context.Context, kubeClient client.Client, previousVersion, targetVersion string) error {
	vpwnedImage, err := getCurrentVpwnedImage(ctx, kubeClient)
	if err != nil {
		return fmt.Errorf("failed to get current vpwned image: %w", err)
	}

	backoffLimit := int32(0)
	ttlSeconds := int32(86400)
	activeDeadlineSeconds := int64(3600)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rollbackJobName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":                          "vjailbreak-rollback",
				"vjailbreak.k8s.pf9.io/job":    "rollback",
				"vjailbreak.k8s.pf9.io/target": previousVersion,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSeconds,
			ActiveDeadlineSeconds:   &activeDeadlineSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                       "vjailbreak-rollback",
						"vjailbreak.k8s.pf9.io/job": "rollback",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: upgradeJobServiceAccount,
					Containers: []corev1.Container{
						{
							Name:            "rollback",
							Image:           vpwnedImage,
							ImagePullPolicy: corev1.PullAlways,
							Command:         []string{"./vpwctl", "upgrade-job"},
							Env: []corev1.EnvVar{
								{
									Name:  "UPGRADE_TARGET_VERSION",
									Value: targetVersion,
								},
								{
									Name:  "UPGRADE_PREVIOUS_VERSION",
									Value: previousVersion,
								},
								{
									Name:  "UPGRADE_MODE",
									Value: "rollback",
								},
							},
						},
					},
				},
			},
		},
	}

	if err := kubeClient.Create(ctx, job); err != nil {
		return fmt.Errorf("failed to create rollback job: %w", err)
	}

	log.Printf("Created rollback job %s for version %s", rollbackJobName, previousVersion)
	return nil
}

func (s *VpwnedVersion) RollbackUpgrade(ctx context.Context, in *api.VersionRequest) (*api.UpgradeProgressResponse, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(batchv1.AddToScheme(scheme))
	kubeClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	progress, err := loadProgressFromConfigMap(ctx, kubeClient)
	if err != nil {
		log.Printf("Warning: failed to load progress: %v", err)
	}

	var previousVersion, targetVersion string
	if progress != nil {
		previousVersion = progress.PreviousVersion
		targetVersion = progress.TargetVersion
	}

	if previousVersion == "" || previousVersion == "unknown" {
		return nil, fmt.Errorf("cannot rollback: previous version unknown - manifest-driven rollback requires known previous version")
	}

	upgradeJob := &batchv1.Job{}
	if err := kubeClient.Get(ctx, client.ObjectKey{Name: upgradeJobName, Namespace: namespace}, upgradeJob); err == nil {
		if upgradeJob.Status.Active > 0 {
			return nil, fmt.Errorf("cannot rollback: upgrade job is still running - wait for it to complete or delete it first")
		}
	}

	existingJob := &batchv1.Job{}
	if err := kubeClient.Get(ctx, client.ObjectKey{Name: rollbackJobName, Namespace: namespace}, existingJob); err == nil {
		if existingJob.Status.Active > 0 {
			return nil, fmt.Errorf("rollback job already running")
		}
		if err := kubeClient.Delete(ctx, existingJob); err != nil && !kerrors.IsNotFound(err) {
			log.Printf("Warning: Failed to delete old rollback job: %v", err)
		}
		jobDeleted := false
		for i := 0; i < 30; i++ {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			err := kubeClient.Get(ctx, client.ObjectKey{Name: rollbackJobName, Namespace: namespace}, existingJob)
			if kerrors.IsNotFound(err) {
				jobDeleted = true
				break
			}
			if err != nil {
				log.Printf("Warning: Error checking rollback job deletion: %v", err)
			}
			time.Sleep(1 * time.Second)
		}
		if !jobDeleted {
			return nil, fmt.Errorf("old rollback job still exists after 30s - cannot start new rollback")
		}
	}

	log.Printf("Starting manifest-driven rollback from %s to %s via Job", targetVersion, previousVersion)

	if progress == nil {
		progress = &UpgradeProgress{StartTime: time.Now()}
	}
	progress.CurrentStep = "Creating rollback job"
	progress.Status = "rolling_back"
	_ = saveProgressToConfigMap(ctx, kubeClient, progress)

	if err := createRollbackJob(ctx, kubeClient, previousVersion, targetVersion); err != nil {
		now := time.Now()
		progress.Status = "rollback_failed"
		progress.Error = fmt.Sprintf("Failed to create rollback job: %v", err)
		progress.EndTime = &now
		_ = saveProgressToConfigMap(ctx, kubeClient, progress)
		return &api.UpgradeProgressResponse{
			Status:      "rollback_failed",
			Error:       fmt.Sprintf("Failed to create rollback job: %v", err),
			CurrentStep: "Creating rollback job",
		}, err
	}

	return &api.UpgradeProgressResponse{
		Status:      "rolling_back",
		Error:       "",
		CurrentStep: "Rollback job created",
	}, nil
}

func (s *VpwnedVersion) GetUpgradeProgress(ctx context.Context, in *api.VersionRequest) (*api.UpgradeProgressResponse, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(batchv1.AddToScheme(scheme))
	kubeClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	// Load progress from ConfigMap (stateless)
	progress, err := loadProgressFromConfigMap(ctx, kubeClient)
	if err != nil {
		log.Printf("Warning: failed to load progress: %v", err)
	}

	if progress == nil {
		return &api.UpgradeProgressResponse{
			Status: "no_upgrade_in_progress",
		}, nil
	}

	jobName := upgradeJobName
	if progress.Status == "rolling_back" {
		jobName = rollbackJobName
	}

	job := &batchv1.Job{}
	jobErr := kubeClient.Get(ctx, client.ObjectKey{Name: jobName, Namespace: namespace}, job)
	if jobErr == nil {
		jobFailed := false
		jobComplete := false
		var failureReason string

		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobFailed && cond.Status == corev1.ConditionTrue {
				jobFailed = true
				failureReason = cond.Reason
				if cond.Message != "" {
					failureReason = cond.Message
				}
			}
			if cond.Type == batchv1.JobComplete && cond.Status == corev1.ConditionTrue {
				jobComplete = true
			}
		}

		if !jobFailed && job.Status.Failed > 0 {
			jobFailed = true
			failureReason = "Job pod failed"
		}
		if !jobComplete && job.Status.Succeeded > 0 {
			jobComplete = true
		}

		if jobFailed && progress.Status != "failed" && progress.Status != "rollback_failed" {
			errMsg := failureReason
			if errMsg == "" {
				errMsg = "Upgrade job failed"
			}
			now := time.Now()
			_ = updateProgressStatusOnly(ctx, kubeClient, "failed", errMsg, &now)
			progress.Status = "failed"
			progress.Error = errMsg
			progress.EndTime = &now
		} else if jobComplete && progress.Status != "completed" && progress.Status != "rolled_back" {
			newStatus := "completed"
			if progress.Status == "rolling_back" {
				newStatus = "rolled_back"
			}
			now := time.Now()
			_ = updateProgressStatusOnly(ctx, kubeClient, newStatus, "", &now)
			progress.Status = newStatus
			progress.EndTime = &now
		}
	} else if kerrors.IsNotFound(jobErr) {
		if progress.Status == "in_progress" || progress.Status == "deploying" || progress.Status == "rolling_back" {
			if time.Since(progress.StartTime) > 30*time.Minute {
				log.Printf("Warning: Job not found and progress stale for >30min, marking as unknown")
				now := time.Now()
				errMsg := "Upgrade job not found - may have been deleted or TTL expired"
				_ = updateProgressStatusOnly(ctx, kubeClient, "unknown", errMsg, &now)
				progress.Status = "unknown"
				progress.Error = errMsg
				progress.EndTime = &now
			}
		}
	}

	var progressPercent float32
	if progress.TotalSteps > 0 {
		progressPercent = float32(progress.CompletedSteps) / float32(progress.TotalSteps) * 100
	}

	if progress.Status == "completed" || progress.Status == "deployments_ready" || progress.Status == "rolled_back" {
		progressPercent = 100
	}

	return &api.UpgradeProgressResponse{
		CurrentStep: progress.CurrentStep,
		Progress:    progressPercent,
		Status:      progress.Status,
		Error:       progress.Error,
		StartTime:   progress.StartTime.Format(time.RFC3339),
		EndTime: func() string {
			if progress.EndTime != nil {
				return progress.EndTime.Format(time.RFC3339)
			}
			return ""
		}(),
	}, nil
}

func (s *VpwnedVersion) Cleanup(ctx context.Context, in *api.CleanupRequest) (*api.CleanupResponse, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	kubeClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	if err := upgrade.CleanupResources(ctx, kubeClient, config); err != nil {
		return &api.CleanupResponse{Success: false, Message: fmt.Sprintf("Cleanup failed: %v", err)}, nil
	}
	return &api.CleanupResponse{Success: true, Message: "All resources cleaned up successfully"}, nil
}

func checkAndDeleteMigrationPlans(ctx context.Context, restConfig *rest.Config) (bool, string) {
	gvr := schema.GroupVersionResource{Group: "vjailbreak.k8s.pf9.io", Version: "v1alpha1", Resource: "migrationplans"}
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return false, "Failed to create dynamic client"
	}

	list, err := dynamicClient.Resource(gvr).Namespace("migration-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, "Failed to list MigrationPlans"
	}

	for _, item := range list.Items {
		_ = dynamicClient.Resource(gvr).Namespace("migration-system").Delete(ctx, item.GetName(), metav1.DeleteOptions{})
	}

	timeout := 5 * time.Minute
	interval := 5 * time.Second
	for start := time.Now(); time.Since(start) < timeout; {
		list, err := dynamicClient.Resource(gvr).Namespace("migration-system").List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, "Failed to re-list MigrationPlans during polling"
		}
		if len(list.Items) == 0 {
			return true, "No MigrationPlans remaining"
		}
		time.Sleep(interval)
	}

	return false, "MigrationPlans still exist after 5 minute timeout"
}

func checkAndDeleteRollingMigrationPlans(ctx context.Context, restConfig *rest.Config) (bool, string) {
	gvr := schema.GroupVersionResource{Group: "vjailbreak.k8s.pf9.io", Version: "v1alpha1", Resource: "rollingmigrationplans"}
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return false, "Failed to create dynamic client"
	}

	list, err := dynamicClient.Resource(gvr).Namespace("migration-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, "Failed to list RollingMigrationPlans"
	}

	for _, item := range list.Items {
		_ = dynamicClient.Resource(gvr).Namespace("migration-system").Delete(ctx, item.GetName(), metav1.DeleteOptions{})
	}

	timeout := 5 * time.Minute
	interval := 5 * time.Second
	for start := time.Now(); time.Since(start) < timeout; {
		list, err := dynamicClient.Resource(gvr).Namespace("migration-system").List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, "Failed to re-list RollingMigrationPlans during polling"
		}
		if len(list.Items) == 0 {
			return true, "No RollingMigrationPlans remaining"
		}
		time.Sleep(interval)
	}

	return false, "RollingMigrationPlans still exist after 5 minute timeout"
}

func checkAndScaleDownAgent(ctx context.Context, restConfig *rest.Config) (bool, string) {
	gvr := schema.GroupVersionResource{Group: "vjailbreak.k8s.pf9.io", Version: "v1alpha1", Resource: "vjailbreaknodes"}
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return false, "Failed to create dynamic client"
	}

	list, err := dynamicClient.Resource(gvr).Namespace("migration-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, "Failed to list VjailbreakNodes"
	}

	for _, item := range list.Items {
		if item.GetName() == "vjailbreak-master" {
			continue
		}
		_ = dynamicClient.Resource(gvr).Namespace("migration-system").Delete(ctx, item.GetName(), metav1.DeleteOptions{})
	}

	timeout := 5 * time.Minute
	interval := 5 * time.Second
	for start := time.Now(); time.Since(start) < timeout; {
		list, err := dynamicClient.Resource(gvr).Namespace("migration-system").List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, "Failed to re-list VjailbreakNodes"
		}

		if len(list.Items) == 0 || (len(list.Items) == 1 && list.Items[0].GetName() == "vjailbreak-master") {
			return true, "Agents scaled down"
		}
		time.Sleep(interval)
	}

	return false, "Non-master agents still exist after 5 minute timeout"
}

func checkAndDeleteSecret(ctx context.Context, kubeClient client.Client, restConfig *rest.Config, credType string) (bool, string) {
	gvr := schema.GroupVersionResource{
		Group:    "vjailbreak.k8s.pf9.io",
		Version:  "v1alpha1",
		Resource: credType,
	}
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return false, "Failed to create dynamic client"
	}
	list, err := dynamicClient.Resource(gvr).Namespace("migration-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, "Failed to list " + credType + " CRs"
	}
	allDeleted := true
	var failed []string
	for _, item := range list.Items {
		spec, ok := item.UnstructuredContent()["spec"].(map[string]interface{})
		if !ok {
			failed = append(failed, item.GetName()+": no spec")
			allDeleted = false
			continue
		}
		secretRef, ok := spec["secretRef"].(map[string]interface{})
		if !ok {
			failed = append(failed, item.GetName()+": no secretRef")
			allDeleted = false
			continue
		}
		secretName, ok := secretRef["name"].(string)
		if !ok || secretName == "" {
			failed = append(failed, item.GetName()+": no secretRef.name")
			allDeleted = false
			continue
		}
		secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: "migration-system"}}
		_ = kubeClient.Delete(ctx, secret)
		err := dynamicClient.Resource(gvr).Namespace("migration-system").Delete(ctx, item.GetName(), metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			failed = append(failed, item.GetName()+": CR delete failed")
			allDeleted = false
		}
	}
	if allDeleted {
		return true, "All " + credType + " and their secrets deleted"
	}
	return false, "Some " + credType + " or their secrets could not be deleted: " + strings.Join(failed, ", ")
}

func checkAndDeleteAllCustomResources(ctx context.Context, kubeClient client.Client, restConfig *rest.Config) (bool, string) {
	currentCRs, err := upgrade.DiscoverCurrentCRs(ctx, kubeClient)
	if err != nil {
		return false, "Failed to discover current CRs: " + err.Error()
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return false, "Failed to create dynamic client: " + err.Error()
	}

	allDeleted := true
	var failed []string
	for _, crInfo := range currentCRs {
		gvr := schema.GroupVersionResource{
			Group:    crInfo.Group,
			Version:  crInfo.Version,
			Resource: crInfo.Plural,
		}
		list, err := dynamicClient.Resource(gvr).Namespace("migration-system").List(ctx, metav1.ListOptions{})
		if err != nil {
			failed = append(failed, crInfo.Kind+": list error")
			allDeleted = false
			continue
		}
		for _, item := range list.Items {
			err := dynamicClient.Resource(gvr).Namespace("migration-system").Delete(ctx, item.GetName(), metav1.DeleteOptions{})
			if err != nil {
				failed = append(failed, crInfo.Kind+":"+item.GetName())
				allDeleted = false
			}
		}
		time.Sleep(2 * time.Second)
		list, err = dynamicClient.Resource(gvr).Namespace("migration-system").List(ctx, metav1.ListOptions{})
		if err != nil || len(list.Items) > 0 {
			failed = append(failed, crInfo.Kind+": not empty after delete")
			allDeleted = false
		}
	}
	if allDeleted {
		return true, "No Custom Resources remaining"
	}
	return false, "Some Custom Resources could not be deleted: " + strings.Join(failed, ", ")
}

func waitForDeploymentReady(ctx context.Context, kubeClient client.Client, depConfig DeploymentConfig) error {
	timeout := 5 * time.Minute
	interval := 10 * time.Second

	for start := time.Now(); time.Since(start) < timeout; {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		dep := &appsv1.Deployment{}
		if err := kubeClient.Get(ctx, client.ObjectKey{Name: depConfig.Name, Namespace: depConfig.Namespace}, dep); err != nil {
			if kerrors.IsNotFound(err) {
				return fmt.Errorf("deployment %s not found", depConfig.Name)
			}
			return fmt.Errorf("failed to get deployment %s: %w", depConfig.Name, err)
		}

		desiredReplicas := int32(1)
		if dep.Spec.Replicas != nil {
			desiredReplicas = *dep.Spec.Replicas
		}

		if dep.Status.ReadyReplicas == desiredReplicas && dep.Status.UpdatedReplicas == desiredReplicas {
			return nil
		}

		time.Sleep(interval)
	}

	return fmt.Errorf("deployment %s not ready within timeout", depConfig.Name)
}

func waitForDeploymentScaledDown(ctx context.Context, kubeClient client.Client, depConfig DeploymentConfig) error {
	timeout := 5 * time.Minute
	interval := 10 * time.Second

	for start := time.Now(); time.Since(start) < timeout; {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		dep := &appsv1.Deployment{}
		if err := kubeClient.Get(ctx, client.ObjectKey{Name: depConfig.Name, Namespace: depConfig.Namespace}, dep); err != nil {
			if kerrors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("failed to get deployment %s: %w", depConfig.Name, err)
		}

		log.Printf("Waiting for deployment %s to scale down: replicas=%d ready=%d updated=%d", depConfig.Name, dep.Status.Replicas, dep.Status.ReadyReplicas, dep.Status.UpdatedReplicas)

		if dep.Status.Replicas == 0 {
			log.Printf("Deployment %s successfully scaled down.", depConfig.Name)
			return nil
		}

		time.Sleep(interval)
	}

	dep := &appsv1.Deployment{}
	if err := kubeClient.Get(ctx, client.ObjectKey{Name: depConfig.Name, Namespace: depConfig.Namespace}, dep); err == nil {
		return fmt.Errorf("deployment %s not scaled down within timeout (replicas=%d ready=%d updated=%d)", depConfig.Name, dep.Status.Replicas, dep.Status.ReadyReplicas, dep.Status.UpdatedReplicas)
	}
	return fmt.Errorf("deployment %s not scaled down within timeout", depConfig.Name)
}
