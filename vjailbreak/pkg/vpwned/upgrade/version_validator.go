package upgrade

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	sigsyaml "sigs.k8s.io/yaml"

	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ValidationResult struct {
	NoMigrationPlans        bool
	NoRollingMigrationPlans bool
	VMwareCredsDeleted      bool
	OpenStackCredsDeleted   bool
	AgentsScaledDown        bool
	NoCustomResources       bool
	PassedAll               bool
}

type CRDInfo struct {
	Name    string
	Version string
	Group   string
}

type CRInfo struct {
	Group    string
	Version  string
	Kind     string
	Plural   string
	Singular string
}

func DiscoverCurrentCRs(ctx context.Context, kubeClient client.Client) ([]CRInfo, error) {
	var currentCRs []CRInfo

	crdList := &apiextensionsv1.CustomResourceDefinitionList{}
	if err := kubeClient.List(ctx, crdList); err != nil {
		return nil, fmt.Errorf("failed to list CRDs: %w", err)
	}

	for _, crd := range crdList.Items {
		if strings.Contains(crd.Spec.Group, "vjailbreak") {
			for _, version := range crd.Spec.Versions {
				crInfo := CRInfo{
					Group:    crd.Spec.Group,
					Version:  version.Name,
					Kind:     crd.Spec.Names.Kind,
					Plural:   crd.Spec.Names.Plural,
					Singular: crd.Spec.Names.Singular,
				}
				currentCRs = append(currentCRs, crInfo)
			}
		}
	}

	return currentCRs, nil
}

func RunPreUpgradeChecks(ctx context.Context, kubeClient client.Client, dynamicClient dynamic.Interface, targetVersion string) (*ValidationResult, error) {
	result := &ValidationResult{}

	gvr := schema.GroupVersionResource{Group: "vjailbreak.k8s.pf9.io", Version: "v1alpha1", Resource: "migrationplans"}
	unstructuredList, err := dynamicClient.Resource(gvr).Namespace(Namespace).List(ctx, metav1.ListOptions{})
	if err == nil && len(unstructuredList.Items) == 0 {
		result.NoMigrationPlans = true
	}

	gvr = schema.GroupVersionResource{Group: "vjailbreak.k8s.pf9.io", Version: "v1alpha1", Resource: "rollingmigrationplans"}
	unstructuredList, err = dynamicClient.Resource(gvr).Namespace(Namespace).List(ctx, metav1.ListOptions{})
	if err == nil && len(unstructuredList.Items) == 0 {
		result.NoRollingMigrationPlans = true
	}

	vmwareSecret := &corev1.Secret{}
	err = kubeClient.Get(ctx, client.ObjectKey{Name: "vmware-credentials", Namespace: Namespace}, vmwareSecret)
	if err != nil {
		if kerrors.IsNotFound(err) {
			result.VMwareCredsDeleted = true
		} else {
			return nil, err
		}
	} else {
		result.VMwareCredsDeleted = false
	}

	openstackSecret := &corev1.Secret{}
	err = kubeClient.Get(ctx, client.ObjectKey{Name: "openstack-credentials", Namespace: Namespace}, openstackSecret)
	if err != nil {
		if kerrors.IsNotFound(err) {
			result.OpenStackCredsDeleted = true
		} else {
			return nil, err
		}
	} else {
		result.OpenStackCredsDeleted = false
	}
	gvr = schema.GroupVersionResource{
		Group:    "vjailbreak.k8s.pf9.io",
		Version:  "v1alpha1",
		Resource: "vjailbreaknodes",
	}
	list, err := dynamicClient.Resource(gvr).Namespace(Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		result.AgentsScaledDown = true
	} else if len(list.Items) == 1 && list.Items[0].GetName() == "vjailbreak-master" {
		result.AgentsScaledDown = true
	} else {
		result.AgentsScaledDown = false
	}

	result.NoCustomResources, err = checkForAnyCustomResources(ctx, kubeClient, dynamicClient)
	if err != nil {
		return nil, err
	}

	result.PassedAll = result.NoMigrationPlans &&
		result.NoRollingMigrationPlans &&
		result.VMwareCredsDeleted &&
		result.OpenStackCredsDeleted &&
		result.AgentsScaledDown &&
		result.NoCustomResources
	return result, nil
}

func checkForAnyCustomResources(ctx context.Context, kubeClient client.Client, dynamicClient dynamic.Interface) (bool, error) {
	currentCRs, err := DiscoverCurrentCRs(ctx, kubeClient)
	if err != nil {
		return false, fmt.Errorf("failed to discover current CRs: %w", err)
	}

	for _, crInfo := range currentCRs {
		gvr := schema.GroupVersionResource{
			Group:    crInfo.Group,
			Version:  crInfo.Version,
			Resource: crInfo.Plural,
		}

		unstructuredList, err := dynamicClient.Resource(gvr).Namespace(Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Printf("Warning: Could not list %s CRs: %v", crInfo.Kind, err)
			continue
		}

		for _, item := range unstructuredList.Items {
			if crInfo.Kind == "VjailbreakNode" && item.GetName() == "vjailbreak-master" {
				continue
			}
			log.Printf("Found custom resource %s: %s", crInfo.Kind, item.GetName())
			return false, nil
		}
	}

	return true, nil
}

func BackupResourcesWithID(ctx context.Context, kubeClient client.Client, restConfig *rest.Config, backupID string) error {
	log.Println("Starting backup of resources...")
	backupLabel := map[string]string{"vjailbreak-backup": "true", "vjailbreak-backup-id": backupID}

	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	crdList := &apiextensionsv1.CustomResourceDefinitionList{}
	if err := kubeClient.List(ctx, crdList); err == nil {
		for _, crd := range crdList.Items {
			if strings.Contains(crd.Spec.Group, "vjailbreak") {
				var buffer strings.Builder
				crd.GetObjectKind().SetGroupVersionKind(apiextensionsv1.SchemeGroupVersion.WithKind("CustomResourceDefinition"))
				if err := s.Encode(&crd, &buffer); err == nil {
					backupCM := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{Name: "backup-crd-" + strings.ReplaceAll(crd.Name, ".", "-"), Namespace: Namespace, Labels: backupLabel},
					}
					_, _ = controllerutil.CreateOrUpdate(ctx, kubeClient, backupCM, func() error {
						backupCM.Data = map[string]string{"resource": buffer.String()}
						return nil
					})
				}
			}
		}
	}

	vjailbreakConfigMaps := []string{"version-config", "vjailbreak-settings"}
	for _, cmName := range vjailbreakConfigMaps {
		cm := &corev1.ConfigMap{}
		if err := kubeClient.Get(ctx, client.ObjectKey{Name: cmName, Namespace: Namespace}, cm); err == nil {
			var buffer strings.Builder
			cm.GetObjectKind().SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
			if err := s.Encode(cm, &buffer); err == nil {
				backupCM := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "backup-cm-" + cmName, Namespace: Namespace, Labels: backupLabel},
				}
				_, _ = controllerutil.CreateOrUpdate(ctx, kubeClient, backupCM, func() error {
					backupCM.Data = map[string]string{"resource": buffer.String()}
					return nil
				})
			}
		} else {
			log.Printf("Warning: ConfigMap %s not found for backup: %v", cmName, err)
		}
	}

	vjailbreakDeployments := []string{"migration-controller-manager", "migration-vpwned-sdk", "vjailbreak-ui"}
	for _, depName := range vjailbreakDeployments {
		dep := &appsv1.Deployment{}
		if err := kubeClient.Get(ctx, client.ObjectKey{Name: depName, Namespace: Namespace}, dep); err == nil {
			var buffer strings.Builder
			dep.GetObjectKind().SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
			if err := s.Encode(dep, &buffer); err == nil {
				backupCM := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "backup-deploy-" + depName, Namespace: Namespace, Labels: backupLabel},
				}
				_, _ = controllerutil.CreateOrUpdate(ctx, kubeClient, backupCM, func() error {
					backupCM.Data = map[string]string{"resource": buffer.String()}
					return nil
				})
			}
		} else {
			log.Printf("Warning: Deployment %s not found for backup: %v", depName, err)
		}
	}
	log.Println("Backup completed.")
	return nil
}

func RestoreResources(ctx context.Context, kubeClient client.Client, backupID string) error {
	log.Printf("Restoring resources from backups (backupID=%s)...", backupID)

	backupLabelSelector := client.MatchingLabels{"vjailbreak-backup": "true"}
	if backupID != "" {
		backupLabelSelector["vjailbreak-backup-id"] = backupID
	}
	backupCMList := &corev1.ConfigMapList{}
	if err := kubeClient.List(ctx, backupCMList, client.InNamespace(Namespace), backupLabelSelector); err != nil {
		return fmt.Errorf("failed to list backup ConfigMaps: %w", err)
	}

	if len(backupCMList.Items) == 0 && backupID != "" {
		return fmt.Errorf("no backups found for backupID=%s", backupID)
	}

	crdBackups := map[string]corev1.ConfigMap{}
	cmBackups := map[string]corev1.ConfigMap{}
	deployBackups := map[string]corev1.ConfigMap{}
	otherBackups := []corev1.ConfigMap{}

	for _, cm := range backupCMList.Items {
		switch {
		case strings.HasPrefix(cm.Name, "backup-crd-"):
			crdBackups[cm.Name] = cm
		case strings.HasPrefix(cm.Name, "backup-cm-"):
			cmBackups[cm.Name] = cm
		case strings.HasPrefix(cm.Name, "backup-deploy-"):
			deployBackups[cm.Name] = cm
		default:
			otherBackups = append(otherBackups, cm)
		}
	}

	for _, cm := range crdBackups {
		yamlData, ok := cm.Data["resource"]
		if !ok {
			log.Printf("backup %s missing resource key; skipping", cm.Name)
			continue
		}
		if err := applyRestoredObject(ctx, kubeClient, []byte(yamlData)); err != nil {
			log.Printf("Failed to restore CRD from backup %s: %v", cm.Name, err)
		} else {
			log.Printf("Restored CRD from backup %s", cm.Name)
		}
	}

	if err := waitForCRDEstablished(ctx, kubeClient, 2*time.Minute); err != nil {
		log.Printf("Warning: CRDs not all established: %v", err)
	}
	for _, cm := range cmBackups {
		yamlData, ok := cm.Data["resource"]
		if !ok {
			continue
		}
		if err := applyRestoredObject(ctx, kubeClient, []byte(yamlData)); err != nil {
			log.Printf("Failed to restore ConfigMap from backup %s: %v", cm.Name, err)
		} else {
			log.Printf("Restored ConfigMap from backup %s", cm.Name)
		}
	}

	controllerName := "migration-controller-manager"
	uiName := "vjailbreak-ui"
	sdkName := "migration-vpwned-sdk"
	ns := Namespace
	findDeployBackup := func(name string) (corev1.ConfigMap, bool) {
		key := "backup-deploy-" + name
		cm, ok := deployBackups[key]
		return cm, ok
	}

	if err := scaleDeploymentTo(ctx, kubeClient, controllerName, ns, 0); err != nil {
		log.Printf("Warning: failed to scale down controller %s: %v", controllerName, err)
	} else {
		log.Printf("Scaled down controller %s", controllerName)
		if err := waitForDeploymentScaledDownLocal(ctx, kubeClient, controllerName, ns, 60*time.Second); err != nil {
			log.Printf("Controller %s did not fully scale down: %v", controllerName, err)
		} else {
			log.Printf("Controller %s fully scaled down", controllerName)
		}
	}

	if cm, ok := findDeployBackup(controllerName); ok {
		yamlData := cm.Data["resource"]
		desired := parseReplicasFromDeploymentYAML([]byte(yamlData))
		if err := applyRestoredObject(ctx, kubeClient, []byte(yamlData)); err != nil {
			log.Printf("Failed to apply controller deployment backup %s: %v", cm.Name, err)
		} else {
			log.Printf("Applied controller deployment backup %s", cm.Name)
			if desired < 1 {
				desired = 1
			}
			if err := scaleDeploymentTo(ctx, kubeClient, controllerName, ns, desired); err != nil {
				log.Printf("Failed to scale up controller %s to %d: %v", controllerName, desired, err)
			} else {
				if err := waitForDeploymentReadyLocal(ctx, kubeClient, controllerName, ns, 5*time.Minute); err != nil {
					log.Printf("Controller %s did not become ready: %v", controllerName, err)
				} else {
					log.Printf("Controller %s is ready", controllerName)
				}
			}
		}
	} else {
		log.Printf("No controller backup found for %s", controllerName)
	}

	if cm, ok := findDeployBackup(uiName); ok {
		yamlData := cm.Data["resource"]
		desired := parseReplicasFromDeploymentYAML([]byte(yamlData))
		if err := applyRestoredObject(ctx, kubeClient, []byte(yamlData)); err != nil {
			log.Printf("Failed to apply UI deployment backup %s: %v", cm.Name, err)
		} else {
			if desired < 1 {
				desired = 1
			}
			if err := scaleDeploymentTo(ctx, kubeClient, uiName, ns, desired); err != nil {
				log.Printf("Failed to scale UI %s to %d: %v", uiName, desired, err)
			} else {
				if err := waitForDeploymentReadyLocal(ctx, kubeClient, uiName, ns, 5*time.Minute); err != nil {
					log.Printf("UI %s did not become ready: %v", uiName, err)
				} else {
					log.Printf("UI %s is ready", uiName)
				}
			}
		}
	} else {
		log.Printf("No UI backup found for %s", uiName)
	}

	if cm, ok := findDeployBackup(sdkName); ok {
		yamlData := cm.Data["resource"]
		desired := parseReplicasFromDeploymentYAML([]byte(yamlData))
		if err := applyRestoredObject(ctx, kubeClient, []byte(yamlData)); err != nil {
			log.Printf("Failed to apply SDK deployment backup %s: %v", cm.Name, err)
		} else {
			if desired < 1 {
				desired = 1
			}
			if err := scaleDeploymentTo(ctx, kubeClient, sdkName, ns, desired); err != nil {
				log.Printf("Failed to scale SDK %s to %d: %v", sdkName, desired, err)
			} else {
				if err := waitForDeploymentReadyLocal(ctx, kubeClient, sdkName, ns, 5*time.Minute); err != nil {
					log.Printf("SDK %s did not become ready: %v", sdkName, err)
				} else {
					log.Printf("SDK %s is ready", sdkName)
				}
			}
		}
	} else {
		log.Printf("No SDK backup found for %s", sdkName)
	}

	for _, cm := range otherBackups {
		yamlData, ok := cm.Data["resource"]
		if !ok {
			continue
		}
		if err := applyRestoredObject(ctx, kubeClient, []byte(yamlData)); err != nil {
			log.Printf("Failed to restore resource from backup %s: %v", cm.Name, err)
		} else {
			log.Printf("Restored resource from backup %s", cm.Name)
		}
	}

	log.Println("Restore completed.")
	return nil
}

func parseReplicasFromDeploymentYAML(data []byte) int32 {
	var dep appsv1.Deployment
	if err := yaml.Unmarshal(data, &dep); err != nil {
		return 1
	}
	if dep.Spec.Replicas == nil {
		return 1
	}
	return *dep.Spec.Replicas
}

func scaleDeploymentTo(ctx context.Context, kubeClient client.Client, name, namespace string, target int32) error {
	// retry.DefaultRetry will retry the update function up to 5 times between each attempt.
	// This automatically handle locking conflicts which can happen if the resource is modified between the Get and Update calls.
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dep := &appsv1.Deployment{}
		if err := kubeClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, dep); err != nil {
			return err
		}
		dep.Spec.Replicas = &target
		return kubeClient.Update(ctx, dep)
	})
}

func waitForDeploymentReadyLocal(ctx context.Context, kubeClient client.Client, name, namespace string, timeout time.Duration) error {
	interval := 10 * time.Second
	for start := time.Now(); time.Since(start) < timeout; {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		dep := &appsv1.Deployment{}
		if err := kubeClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, dep); err != nil {
			if kerrors.IsNotFound(err) {
				return fmt.Errorf("deployment %s not found", name)
			}
			return err
		}
		desired := int32(1)
		if dep.Spec.Replicas != nil {
			desired = *dep.Spec.Replicas
		}
		if dep.Status.ReadyReplicas == desired && dep.Status.UpdatedReplicas == desired {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("deployment %s not ready within timeout", name)
}

func waitForDeploymentScaledDownLocal(ctx context.Context, kubeClient client.Client, name, namespace string, timeout time.Duration) error {
	interval := 5 * time.Second
	for start := time.Now(); time.Since(start) < timeout; {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		dep := &appsv1.Deployment{}
		if err := kubeClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, dep); err != nil {
			if kerrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		if dep.Status.Replicas == 0 {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("deployment %s not scaled down within timeout", name)
}

func waitForCRDEstablished(ctx context.Context, kubeClient client.Client, timeout time.Duration) error {
	start := time.Now()
	for {
		if time.Since(start) > timeout {
			return fmt.Errorf("timed out waiting for CRDs to become Established")
		}
		crdList := &apiextensionsv1.CustomResourceDefinitionList{}
		if err := kubeClient.List(ctx, crdList); err != nil {
			log.Printf("Error listing CRDs while waiting for Established: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		allEstablished := true
		for _, crd := range crdList.Items {
			if !strings.Contains(crd.Spec.Group, "vjailbreak") {
				continue
			}
			established := false
			for _, c := range crd.Status.Conditions {
				if c.Type == apiextensionsv1.Established && c.Status == apiextensionsv1.ConditionTrue {
					established = true
					break
				}
			}
			if !established {
				allEstablished = false
				break
			}
		}
		if allEstablished {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
}

func CleanupBackupConfigMaps(ctx context.Context, kubeClient client.Client, backupID string) error {
	selector := client.MatchingLabels{"vjailbreak-backup": "true"}
	if backupID != "" {
		selector = client.MatchingLabels{"vjailbreak-backup": "true", "vjailbreak-backup-id": backupID}
	}
	backupCMList := &corev1.ConfigMapList{}
	if err := kubeClient.List(ctx, backupCMList, client.InNamespace(Namespace), selector); err != nil {
		return fmt.Errorf("failed to list backup ConfigMaps for cleanup: %w", err)
	}
	for _, cm := range backupCMList.Items {
		if err := kubeClient.Delete(ctx, &cm); err != nil && !kerrors.IsNotFound(err) {
			log.Printf("Failed to delete backup ConfigMap %s: %v", cm.Name, err)
		}
	}
	return nil
}

func applyRestoredObject(ctx context.Context, kubeClient client.Client, data []byte) error {
	jsonData, err := sigsyaml.YAMLToJSON(data)
	if err != nil {
		return fmt.Errorf("failed to convert yaml to json: %w", err)
	}
	unstructuredObj := &unstructured.Unstructured{}
	if err := unstructuredObj.UnmarshalJSON(jsonData); err != nil {
		return fmt.Errorf("failed to unmarshal into unstructured: %w", err)
	}

	objKey := client.ObjectKey{Name: unstructuredObj.GetName(), Namespace: unstructuredObj.GetNamespace()}
	existingObj := &unstructured.Unstructured{}
	existingObj.SetGroupVersionKind(unstructuredObj.GroupVersionKind())

	err = kubeClient.Get(ctx, objKey, existingObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			unstructuredObj.SetResourceVersion("")
			return kubeClient.Create(ctx, unstructuredObj)
		}
		return err
	}

	unstructuredObj.SetResourceVersion(existingObj.GetResourceVersion())
	return kubeClient.Update(ctx, unstructuredObj)
}

func CleanupResources(ctx context.Context, kubeClient client.Client, restConfig *rest.Config) error {
	log.Println("Starting automatic resource cleanup...")
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	gvrMigrationPlans := schema.GroupVersionResource{Group: "vjailbreak.k8s.pf9.io", Version: "v1alpha1", Resource: "migrationplans"}
	mpList, err := dynamicClient.Resource(gvrMigrationPlans).Namespace(Namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, item := range mpList.Items {
			_ = dynamicClient.Resource(gvrMigrationPlans).Namespace(Namespace).Delete(ctx, item.GetName(), metav1.DeleteOptions{})
		}
		log.Println("Deleted MigrationPlans.")
	}

	gvrRollingPlans := schema.GroupVersionResource{Group: "vjailbreak.k8s.pf9.io", Version: "v1alpha1", Resource: "rollingmigrationplans"}
	rmpList, err := dynamicClient.Resource(gvrRollingPlans).Namespace(Namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, item := range rmpList.Items {
			_ = dynamicClient.Resource(gvrRollingPlans).Namespace(Namespace).Delete(ctx, item.GetName(), metav1.DeleteOptions{})
		}
		log.Println("Deleted RollingMigrationPlans.")
	}

	gvrNodes := schema.GroupVersionResource{Group: "vjailbreak.k8s.pf9.io", Version: "v1alpha1", Resource: "vjailbreaknodes"}
	nodeList, err := dynamicClient.Resource(gvrNodes).Namespace(Namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, item := range nodeList.Items {
			if item.GetName() != "vjailbreak-master" {
				_ = dynamicClient.Resource(gvrNodes).Namespace(Namespace).Delete(ctx, item.GetName(), metav1.DeleteOptions{})
			}
		}
		log.Println("Scaled down agents by deleting non-master nodes.")
	}

	vmwareSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "vmware-credentials", Namespace: Namespace}}
	if err := kubeClient.Delete(ctx, vmwareSecret); err == nil || kerrors.IsNotFound(err) {
		log.Println("Secret vmware-credentials deleted.")
	}

	openstackSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "openstack-credentials", Namespace: Namespace}}
	if err := kubeClient.Delete(ctx, openstackSecret); err == nil || kerrors.IsNotFound(err) {
		log.Println("Secret openstack-credentials deleted.")
	}

	if err := deleteAllCustomResources(ctx, kubeClient, restConfig); err != nil {
		log.Printf("Failed to delete all custom resources: %v", err)
	}

	log.Println("Resource cleanup completed.")
	return nil
}

func deleteAllCustomResources(ctx context.Context, kubeClient client.Client, restConfig *rest.Config) error {
	currentCRs, err := DiscoverCurrentCRs(ctx, kubeClient)
	if err != nil {
		return fmt.Errorf("failed to discover current CRs: %w", err)
	}

	for _, crInfo := range currentCRs {
		if err := deleteCRInstances(ctx, restConfig, crInfo); err != nil {
			log.Printf("Warning: Failed to delete %s CRs: %v", crInfo.Kind, err)
		}
	}

	return nil
}

func deleteCRInstances(ctx context.Context, restConfig *rest.Config, crInfo CRInfo) error {
	gvr := schema.GroupVersionResource{
		Group:    crInfo.Group,
		Version:  crInfo.Version,
		Resource: crInfo.Plural,
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client for %s: %w", crInfo.Kind, err)
	}

	unstructuredList, err := dynamicClient.Resource(gvr).Namespace(Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list %s CRs: %w", crInfo.Kind, err)
	}

	for _, item := range unstructuredList.Items {
		err := dynamicClient.Resource(gvr).Namespace(Namespace).Delete(ctx, item.GetName(), metav1.DeleteOptions{})
		if err != nil {
			log.Printf("Failed to delete %s %s: %v", crInfo.Kind, item.GetName(), err)
		} else {
			log.Printf("Deleted %s: %s", crInfo.Kind, item.GetName())
		}
	}

	if len(unstructuredList.Items) > 0 {
		log.Printf("Deleted %d %s CRs", len(unstructuredList.Items), crInfo.Kind)
	}

	return nil
}

func ApplyAllCRDs(ctx context.Context, kubeClient client.Client, tag string) error {
	url := fmt.Sprintf("https://raw.githubusercontent.com/platform9/vjailbreak/%s/deploy/00crds.yaml", tag)
	log.Printf("Fetching upgrade manifest from: %s", url)

	resp, err := httpGetWithRetry(ctx, url, 3)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest from %s: %w", url, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read manifest body: %w", err)
	}

	decoder := utilyaml.NewYAMLOrJSONDecoder(strings.NewReader(string(bodyBytes)), 4096)

	for {
		u := &unstructured.Unstructured{}
		if err := decoder.Decode(u); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read manifest body: %w", err)
		}

		if u.GetKind() == "" {
			continue
		}

		if u.GetKind() == "Namespace" {
			log.Printf("Skipping Namespace resource: %s", u.GetName())
			continue
		}

		if u.GetKind() == "CustomResourceDefinition" {
			spec, found, _ := unstructured.NestedMap(u.Object, "spec")
			if !found {
				log.Printf("Skipping CRD without spec: %s", u.GetName())
				continue
			}

			group, ok := spec["group"].(string)
			if !ok {
				log.Printf("Skipping CRD without group in spec: %s", u.GetName())
				continue
			}

			if !strings.Contains(group, "vjailbreak") {
				log.Printf("Skipping non-vjailbreak CRD: %s (group: %s)", u.GetName(), group)
				continue
			}
		}

		key := types.NamespacedName{Name: u.GetName(), Namespace: u.GetNamespace()}

		existing := &unstructured.Unstructured{}
		existing.SetGroupVersionKind(u.GroupVersionKind())
		err := kubeClient.Get(ctx, key, existing)

		if kerrors.IsNotFound(err) {
			if err := retry.OnError(
				retry.DefaultRetry,
				func(err error) bool {
					return kerrors.IsTooManyRequests(err)
				},
				func() error {
					return kubeClient.Create(ctx, u)
				},
			); err != nil {
				log.Printf("Failed to create %s %s/%s: %v", u.GetKind(), u.GetNamespace(), u.GetName(), err)
				return err
			}
			log.Printf("Created %s %s/%s", u.GetKind(), u.GetNamespace(), u.GetName())
		} else if err == nil {
			u.SetResourceVersion(existing.GetResourceVersion())
			if err := retry.OnError(
				retry.DefaultRetry,
				func(err error) bool {
					return kerrors.IsTooManyRequests(err)
				},
				func() error {
					return kubeClient.Update(ctx, u)
				},
			); err != nil {
				log.Printf("Failed to update %s %s/%s: %v", u.GetKind(), u.GetNamespace(), u.GetName(), err)
				return err
			}
			log.Printf("Updated %s %s/%s", u.GetKind(), u.GetNamespace(), u.GetName())
		} else {
			return fmt.Errorf("failed to get existing resource %s/%s: %w", u.GetNamespace(), u.GetName(), err)
		}
	}

	log.Printf("Successfully applied all resources from manifest %s", tag)
	return nil
}

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func httpGetWithRetry(ctx context.Context, url string, maxRetries int) (*http.Response, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := httpClient.Do(req)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				return resp, nil
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
				lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
				backoff := time.Duration(1<<uint(i)) * time.Second
				log.Printf("Retrying HTTP request to %s after %v (attempt %d/%d)", url, backoff, i+1, maxRetries)
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				continue
			}
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
		}
		lastErr = err
		backoff := time.Duration(1<<uint(i)) * time.Second
		log.Printf("Retrying HTTP request to %s after %v (attempt %d/%d): %v", url, backoff, i+1, maxRetries, err)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func fetchVersionConfigFromGitHub(ctx context.Context, tag string) ([]byte, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/platform9/vjailbreak/%s/image_builder/configs/version-config.yaml", tag)
	resp, err := httpGetWithRetry(ctx, url, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version-config: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func fetchVjailbreakSettingsFromGitHub(ctx context.Context, tag string) ([]byte, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/platform9/vjailbreak/%s/image_builder/configs/vjailbreak-settings.yaml", tag)
	resp, err := httpGetWithRetry(ctx, url, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vjailbreak-settings: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func ApplyManifestFromGitHub(ctx context.Context, kubeClient client.Client, tag, manifestPath string) error {
	url := fmt.Sprintf("https://raw.githubusercontent.com/platform9/vjailbreak/%s/%s", tag, manifestPath)
	log.Printf("Fetching deployment manifest from: %s", url)

	resp, err := httpGetWithRetry(ctx, url, 3)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest from %s: %w", url, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read manifest body: %w", err)
	}

	decoder := utilyaml.NewYAMLOrJSONDecoder(strings.NewReader(string(bodyBytes)), 4096)

	for {
		u := &unstructured.Unstructured{}
		if err := decoder.Decode(u); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode manifest: %w", err)
		}

		if u.GetKind() == "" {
			continue
		}

		if u.GetName() == "" {
			return fmt.Errorf("manifest %s contains %s with empty metadata.name",
				manifestPath, u.GetKind())
		}

		if u.GetNamespace() == "" && u.GetKind() != "CustomResourceDefinition" {
			u.SetNamespace(Namespace)
		}

		key := types.NamespacedName{Name: u.GetName(), Namespace: u.GetNamespace()}

		existing := &unstructured.Unstructured{}
		existing.SetGroupVersionKind(u.GroupVersionKind())
		err := kubeClient.Get(ctx, key, existing)

		retryFn := func(err error) bool {
			return kerrors.IsTooManyRequests(err) || kerrors.IsConflict(err)
		}

		if kerrors.IsNotFound(err) {
			if err := retry.OnError(
				retry.DefaultRetry,
				retryFn,
				func() error {
					return kubeClient.Create(ctx, u)
				},
			); err != nil {
				return fmt.Errorf("failed to create %s %s/%s: %w",
					u.GetKind(), u.GetNamespace(), u.GetName(), err)
			}
			log.Printf("Created %s %s/%s from GitHub manifest",
				u.GetKind(), u.GetNamespace(), u.GetName())
		} else if err == nil {
			patch := client.MergeFrom(existing.DeepCopy())
			if err := retry.OnError(
				retry.DefaultRetry,
				retryFn,
				func() error {
					return kubeClient.Patch(ctx, u, patch)
				},
			); err != nil {
				return fmt.Errorf("failed to patch %s %s/%s: %w",
					u.GetKind(), u.GetNamespace(), u.GetName(), err)
			}
			log.Printf("Patched %s %s/%s from GitHub manifest",
				u.GetKind(), u.GetNamespace(), u.GetName())
		} else {
			return fmt.Errorf("failed to get existing resource %s/%s: %w",
				u.GetNamespace(), u.GetName(), err)
		}
	}

	log.Printf("Successfully applied manifest %s from tag %s", manifestPath, tag)
	return nil
}

func UpdateVersionConfigMapFromGitHub(ctx context.Context, kubeClient client.Client, tag string) error {
	data, err := fetchVersionConfigFromGitHub(ctx, tag)
	if err != nil {
		return err
	}
	rendered := strings.ReplaceAll(string(data), "${TAG}", tag)
	cm := &corev1.ConfigMap{}
	if err := yaml.Unmarshal([]byte(rendered), cm); err != nil {
		return err
	}
	cm.Namespace = Namespace
	cm.Name = "version-config"

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing := &corev1.ConfigMap{}
		err := kubeClient.Get(ctx, client.ObjectKey{Name: cm.Name, Namespace: cm.Namespace}, existing)
		if kerrors.IsNotFound(err) {
			if createErr := kubeClient.Create(ctx, cm); createErr != nil {
				return createErr
			}
			log.Printf("Successfully created version-config ConfigMap for version %s.", tag)
			return nil
		} else if err != nil {
			return err
		}
		cm.ResourceVersion = existing.ResourceVersion
		if updateErr := kubeClient.Update(ctx, cm); updateErr != nil {
			return updateErr
		}
		log.Printf("Successfully updated version-config ConfigMap to version %s.", tag)
		return nil
	})
}

func UpdateVjailbreakSettingsFromGitHub(ctx context.Context, kubeClient client.Client, tag string) error {
	data, err := fetchVjailbreakSettingsFromGitHub(ctx, tag)
	if err != nil {
		return err
	}
	rendered := strings.ReplaceAll(string(data), "${TAG}", tag)
	cm := &corev1.ConfigMap{}
	if err := yaml.Unmarshal([]byte(rendered), cm); err != nil {
		return err
	}
	cm.Namespace = Namespace
	cm.Name = "vjailbreak-settings"

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing := &corev1.ConfigMap{}
		err := kubeClient.Get(ctx, client.ObjectKey{Name: cm.Name, Namespace: cm.Namespace}, existing)
		if kerrors.IsNotFound(err) {
			if createErr := kubeClient.Create(ctx, cm); createErr != nil {
				return createErr
			}
			log.Printf("Successfully created vjailbreak-settings ConfigMap for version %s.", tag)
			return nil
		} else if err != nil {
			return err
		}
		cm.ResourceVersion = existing.ResourceVersion
		if updateErr := kubeClient.Update(ctx, cm); updateErr != nil {
			return updateErr
		}
		log.Printf("Successfully updated vjailbreak-settings ConfigMap to version %s.", tag)
		return nil
	})
}

func CleanupAllOldBackups(ctx context.Context, kubeClient client.Client, excludeBackupID string) error {
	log.Printf("Cleaning up old backup ConfigMaps (excluding backupID=%s)...", excludeBackupID)

	backupCMList := &corev1.ConfigMapList{}
	if err := kubeClient.List(ctx, backupCMList, client.InNamespace(Namespace), client.MatchingLabels{"vjailbreak-backup": "true"}); err != nil {
		return fmt.Errorf("failed to list backup ConfigMaps: %w", err)
	}

	maxAge := 1 * time.Hour
	deletedCount := 0

	for _, cm := range backupCMList.Items {
		if excludeBackupID != "" {
			if cmBackupID, ok := cm.Labels["vjailbreak-backup-id"]; ok && cmBackupID == excludeBackupID {
				log.Printf("Skipping current backup: %s (backupID=%s)", cm.Name, cmBackupID)
				continue
			}
		}

		if time.Since(cm.CreationTimestamp.Time) < maxAge {
			log.Printf("Skipping recent backup: %s (age=%v)", cm.Name, time.Since(cm.CreationTimestamp.Time))
			continue
		}

		if err := kubeClient.Delete(ctx, &cm); err != nil {
			log.Printf("Failed to delete old backup ConfigMap %s/%s: %v", cm.Namespace, cm.Name, err)
		} else {
			log.Printf("Deleted old backup ConfigMap %s/%s", cm.Namespace, cm.Name)
			deletedCount++
		}
	}

	log.Printf("Old backup ConfigMaps cleanup complete. Deleted %d backups.", deletedCount)
	return nil
}
