// Package utils provides utility functions for VMware to OpenStack migration operations.
// It includes functions for managing migration status conditions, tracking migration progress,
// handling events related to migrations, and other migration lifecycle management functions.
// These utilities support the core migration process between VMware and OpenStack environments,
// including validation, data copying, migration execution, and failure handling.
package utils

import (
	"slices"
	"sort"
	"strings"

	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	"github.com/platform9/vjailbreak/k8s/migration/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MigrationUtils defines the interface for migration utility functions.
type MigrationUtils interface {
	// CreateValidatedCondition creates a validated condition for the migration.
	CreateValidatedCondition(migration *vjailbreakv1alpha1.Migration, eventList *corev1.EventList) []corev1.PodCondition

	// CreateDataCopyCondition creates a data copy condition for the migration.
	CreateDataCopyCondition(migration *vjailbreakv1alpha1.Migration, eventList *corev1.EventList) []corev1.PodCondition

	// CreateMigratingCondition creates a migrated condition for the migration.
	CreateMigratingCondition(migration *vjailbreakv1alpha1.Migration, eventList *corev1.EventList) []corev1.PodCondition

	// SetCutoverLabel sets the cutover label based on the initiateCutover flag.
	SetCutoverLabel(initiateCutover bool, currentLabel string) string

	// SplitEventStringOnComma splits a string by comma and returns a slice of substrings.
	SplitEventStringOnComma(input string) (string, string)

	// GetSatusConditions returns the status conditions of the migration.
	GetSatusConditions(migration *vjailbreakv1alpha1.Migration) []corev1.PodCondition

	// GetConditonIndex returns the index of the condition in the conditions slice.
	GetConditonIndex(conditions []corev1.PodCondition, conditionType corev1.PodConditionType, reasons ...string) int

	// GeneratePodCondition generates a pod condition.
	GeneratePodCondition(conditionType corev1.PodConditionType,
		status corev1.ConditionStatus,
		reason, message string,
		timestamp metav1.Time) *corev1.PodCondition

	// SortConditionsByLastTransitionTime sorts conditions by LastTransitionTime.
	SortConditionsByLastTransitionTime(conditions []corev1.PodCondition)
}

// CreateValidatedCondition creates a validated condition for a migration
func CreateValidatedCondition(migration *vjailbreakv1alpha1.Migration, eventList *corev1.EventList) []corev1.PodCondition {
	existingConditions := migration.Status.Conditions
	for i := 0; i < len(eventList.Items); i++ {
		if eventList.Items[i].Reason != constants.MigrationReason || eventList.Items[i].Message != "Creating volumes in OpenStack" {
			continue
		}

		idx := GetConditonIndex(existingConditions, constants.MigrationConditionTypeValidated, constants.MigrationReason)
		statuscondition := GeneratePodCondition(constants.MigrationConditionTypeValidated,
			corev1.ConditionTrue,
			constants.MigrationReason,
			"Migration validated successfully",
			eventList.Items[i].LastTimestamp)

		if idx == -1 {
			existingConditions = append(existingConditions, *statuscondition)
		} else {
			existingConditions[idx] = *statuscondition
		}
		break
	}
	return existingConditions
}

// CreateDataCopyCondition creates a data copy condition for a migration
func CreateDataCopyCondition(migration *vjailbreakv1alpha1.Migration, eventList *corev1.EventList) []corev1.PodCondition {
	existingConditions := migration.Status.Conditions
	for i := 0; i < len(eventList.Items); i++ {
		if eventList.Items[i].Reason != constants.MigrationReason || !strings.Contains(eventList.Items[i].Message, "Copying disk") {
			continue
		}
		reason, message := SplitEventStringOnComma(eventList.Items[i].Message)
		idx := GetConditonIndex(existingConditions, constants.MigrationConditionTypeDataCopy, reason)
		statuscondition := GeneratePodCondition(constants.MigrationConditionTypeDataCopy,
			corev1.ConditionTrue,
			reason,
			message,
			eventList.Items[i].LastTimestamp)

		if idx == -1 {
			existingConditions = append(existingConditions, *statuscondition)
		} else {
			existingConditions[idx] = *statuscondition
		}
		break
	}
	return existingConditions
}

// CreateMigratingCondition creates a migrating condition for a migration
func CreateMigratingCondition(migration *vjailbreakv1alpha1.Migration, eventList *corev1.EventList) []corev1.PodCondition {
	existingConditions := migration.Status.Conditions
	for i := 0; i < len(eventList.Items); i++ {
		if eventList.Items[i].Reason != constants.MigrationReason || eventList.Items[i].Message != "Converting disk" {
			continue
		}

		idx := GetConditonIndex(existingConditions, constants.MigrationConditionTypeMigrating, constants.MigrationReason)
		statuscondition := GeneratePodCondition(constants.MigrationConditionTypeMigrating,
			corev1.ConditionTrue,
			constants.MigrationReason,
			"Migrating VM from VMware to OpenStack",
			eventList.Items[i].LastTimestamp)

		if idx == -1 {
			existingConditions = append(existingConditions, *statuscondition)
		} else {
			existingConditions[idx] = *statuscondition
		}
	}
	return existingConditions
}

// CreateFailedCondition creates or updates a failed condition for a migration based on events.
// It analyzes event logs to identify failure reasons and updates the migration's status conditions accordingly.
func CreateFailedCondition(migration *vjailbreakv1alpha1.Migration, eventList *corev1.EventList) []corev1.PodCondition {
	existingConditions := migration.Status.Conditions
	for i := 0; i < len(eventList.Items); i++ {
		if eventList.Items[i].Reason != constants.MigrationReason || !strings.Contains(eventList.Items[i].Message, "failed to") {
			continue
		}

		idx := GetConditonIndex(existingConditions, constants.MigrationConditionTypeFailed, constants.MigrationReason)
		statuscondition := GeneratePodCondition(constants.MigrationConditionTypeFailed,
			corev1.ConditionTrue,
			constants.MigrationReason,
			eventList.Items[i].Message,
			eventList.Items[i].LastTimestamp)

		if idx == -1 {
			existingConditions = append(existingConditions, *statuscondition)
		} else {
			existingConditions[idx] = *statuscondition
		}
	}
	return existingConditions
}

// CreateSucceededCondition creates or updates a succeeded condition for a migration based on events.
func CreateSucceededCondition(migration *vjailbreakv1alpha1.Migration, eventList *corev1.EventList) []corev1.PodCondition {
	existingConditions := migration.Status.Conditions
	for i := 0; i < len(eventList.Items); i++ {
		if eventList.Items[i].Reason != constants.MigrationReason || !strings.Contains(eventList.Items[i].Message, "VM created successfully") {
			continue
		}

		idx := GetConditonIndex(existingConditions, constants.MigrationConditionTypeMigrated, constants.MigrationReason)
		statuscondition := GeneratePodCondition(constants.MigrationConditionTypeMigrated,
			corev1.ConditionTrue,
			constants.MigrationReason,
			"VM successfully migrated from VMware to OpenStack",
			eventList.Items[i].LastTimestamp)

		if idx == -1 {
			existingConditions = append(existingConditions, *statuscondition)
		} else {
			existingConditions[idx] = *statuscondition
		}
		break
	}
	return existingConditions
}

// SetCutoverLabel sets the cutover label for a migration
func SetCutoverLabel(initiateCutover bool, currentLabel string) string {
	// If initiateCutover is true, return the current label
	if initiateCutover {
		return currentLabel
	}
	// If initiateCutover is false, set the label to "yes" (User should not be able to change it)
	return constants.StartCutOverYes
}

// SplitEventStringOnComma splits a string by comma and returns a slice of substrings.
func SplitEventStringOnComma(input string) (reason, message string) {
	// SplitEventStringOnComma splits a string by comma and returns a slice of substrings.
	parts := strings.Split(input, ",")
	if len(parts) > 1 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(parts[0]), ""
}

// GetSatusConditions returns the status conditions of a migration
func GetSatusConditions(migration *vjailbreakv1alpha1.Migration) []corev1.PodCondition {
	// GetSatusConditions returns the status conditions of a migration
	return migration.Status.Conditions
}

// GetConditonIndex returns the index of a condition in the conditions slice based on type and reasons
func GetConditonIndex(conditions []corev1.PodCondition, conditionType corev1.PodConditionType, reasons ...string) int {
	// GetConditonIndex returns the index of a condition in the conditions slice based on type and reasons
	for i, c := range conditions {
		if c.Type == conditionType && slices.Contains(reasons, c.Reason) {
			return i
		}
	}
	return -1
}

// GeneratePodCondition creates a new pod condition with the given parameters
func GeneratePodCondition(conditionType corev1.PodConditionType,
	status corev1.ConditionStatus,
	reason, message string,
	timestamp metav1.Time) *corev1.PodCondition {
	// GeneratePodCondition creates a new pod condition with the given parameters
	return &corev1.PodCondition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: timestamp,
	}
}

// SortConditionsByLastTransitionTime sorts conditions by LastTransitionTime
func SortConditionsByLastTransitionTime(conditions []corev1.PodCondition) {
	// SortConditionsByLastTransitionTime sorts conditions by LastTransitionTime
	sort.Slice(conditions, func(i, j int) bool {
		return conditions[i].LastTransitionTime.Before(&conditions[j].LastTransitionTime)
	})
}

// CreateStorageAcceleratedCopyCondition creates a StorageAcceleratedCopy condition for a migration based on StorageAcceleratedCopy-specific events
func CreateStorageAcceleratedCopyCondition(migration *vjailbreakv1alpha1.Migration, eventList *corev1.EventList) []corev1.PodCondition {
	existingConditions := migration.Status.Conditions
	for i := 0; i < len(eventList.Items); i++ {
		if eventList.Items[i].Reason != constants.MigrationReason {
			continue
		}

		message := eventList.Items[i].Message
		var conditionMessage string
		var conditionReason string

		// Check for StorageAcceleratedCopy specific events
		switch {
		case strings.Contains(message, "Connecting to ESXi"):
			conditionReason = "ConnectingToESXi"
			conditionMessage = message
		case strings.Contains(message, "Creating/updating initiator group"):
			conditionReason = "MappingInitiatorGroup"
			conditionMessage = message
		case strings.Contains(message, "Creating target volume"):
			conditionReason = "CreatingVolume"
			conditionMessage = message
		case strings.Contains(message, "Cinder managing the volume"):
			conditionReason = "ImportingToCinder"
			conditionMessage = message
		case strings.Contains(message, "Mapping target volume"):
			conditionReason = "MappingVolume"
			conditionMessage = message
		case strings.Contains(message, "Waiting for target volume"):
			conditionReason = "RescanningStorage"
			conditionMessage = message
		default:
			continue
		}

		idx := GetConditonIndex(existingConditions, constants.MigrationConditionTypeStorageAcceleratedCopy, conditionReason)
		statuscondition := GeneratePodCondition(constants.MigrationConditionTypeStorageAcceleratedCopy,
			corev1.ConditionTrue,
			conditionReason,
			conditionMessage,
			eventList.Items[i].LastTimestamp)

		if idx == -1 {
			existingConditions = append(existingConditions, *statuscondition)
		} else {
			existingConditions[idx] = *statuscondition
		}
		break
	}
	return existingConditions
}
