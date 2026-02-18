/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package migrationmetrics provides Prometheus metrics for tracking VM migration progress and status.
package migrationmetrics

import (
	"time"

	vjailbreakv1alpha1 "github.com/platform9/vjailbreak/k8s/migration/api/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// MigrationPhaseGauge tracks the current phase of each migration
	// Labels: migration_name, vm_name, phase, namespace, agent_name
	// phase values from VMMigrationPhase: Pending, Validating, ValidationFailed, CopyingBlocks, Succeeded, Failed, etc.
	MigrationPhaseGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vjailbreak_migration_phase",
			Help: "Current phase of VM migrations (1=in this phase, 0=not in this phase)",
		},
		[]string{"migration_name", "vm_name", "phase", "namespace", "agent_name"},
	)

	// MigrationDurationSeconds tracks how long each migration has been running
	MigrationDurationSeconds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vjailbreak_migration_duration_seconds",
			Help: "Duration of running migrations in seconds since start time",
		},
		[]string{"migration_name", "vm_name", "namespace"},
	)

	// MigrationStartedTotal counts total migrations started
	MigrationStartedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vjailbreak_migration_started_total",
			Help: "Total number of VM migrations started",
		},
		[]string{"vm_name", "namespace"},
	)

	// MigrationCompletedTotal counts total completed migrations by status
	MigrationCompletedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vjailbreak_migration_completed_total",
			Help: "Total number of VM migrations completed",
		},
		[]string{"vm_name", "status", "namespace"}, // status: Succeeded, Failed
	)

	// MigrationExpectedDurationSeconds stores the expected duration threshold for alerting
	MigrationExpectedDurationSeconds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vjailbreak_migration_expected_duration_seconds",
			Help: "Expected duration threshold for migrations in seconds (for alerting on slow migrations)",
		},
		[]string{"namespace"},
	)

	// MigrationInfo provides metadata about migrations
	MigrationInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vjailbreak_migration_info",
			Help: "Information about migrations with labels for metadata (always 1)",
		},
		[]string{"migration_name", "vm_name", "namespace", "migration_plan", "agent_name"},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		MigrationPhaseGauge,
		MigrationDurationSeconds,
		MigrationStartedTotal,
		MigrationCompletedTotal,
		MigrationExpectedDurationSeconds,
		MigrationInfo,
	)
}

// RecordMigrationStarted records when a migration starts
func RecordMigrationStarted(migrationName, vmName, namespace, migrationPlan, agentName string) {
	MigrationStartedTotal.WithLabelValues(vmName, namespace).Inc()
	MigrationPhaseGauge.WithLabelValues(migrationName, vmName, string(vjailbreakv1alpha1.VMMigrationPhasePending), namespace, agentName).Set(1)
	MigrationInfo.WithLabelValues(migrationName, vmName, namespace, migrationPlan, agentName).Set(1)
}

// UpdateMigrationPhase updates the current phase of a migration
func UpdateMigrationPhase(migrationName, vmName, namespace, agentName string, phase vjailbreakv1alpha1.VMMigrationPhase) {
	// Get all possible phases
	allPhases := []vjailbreakv1alpha1.VMMigrationPhase{
		vjailbreakv1alpha1.VMMigrationPhasePending,
		vjailbreakv1alpha1.VMMigrationPhaseValidating,
		vjailbreakv1alpha1.VMMigrationPhaseValidationFailed,
		vjailbreakv1alpha1.VMMigrationPhaseAwaitingDataCopyStart,
		vjailbreakv1alpha1.VMMigrationPhaseCopying,
		vjailbreakv1alpha1.VMMigrationPhaseCopyingChangedBlocks,
		vjailbreakv1alpha1.VMMigrationPhaseConvertingDisk,
		vjailbreakv1alpha1.VMMigrationPhaseAwaitingCutOverStartTime,
		vjailbreakv1alpha1.VMMigrationPhaseAwaitingAdminCutOver,
		vjailbreakv1alpha1.VMMigrationPhaseSucceeded,
		vjailbreakv1alpha1.VMMigrationPhaseFailed,
		vjailbreakv1alpha1.VMMigrationPhaseUnknown,
	}

	// Clear all phases first (set to 0)
	for _, p := range allPhases {
		MigrationPhaseGauge.WithLabelValues(migrationName, vmName, string(p), namespace, agentName).Set(0)
	}

	// Set current phase to 1
	MigrationPhaseGauge.WithLabelValues(migrationName, vmName, string(phase), namespace, agentName).Set(1)
}

// RecordMigrationProgress updates migration duration based on start time
func RecordMigrationProgress(migrationName, vmName, namespace string, startTime time.Time) {
	duration := time.Since(startTime).Seconds()
	MigrationDurationSeconds.WithLabelValues(migrationName, vmName, namespace).Set(duration)
}

// RecordMigrationCompleted records when a migration completes
func RecordMigrationCompleted(migrationName, vmName, namespace, agentName string, phase vjailbreakv1alpha1.VMMigrationPhase) {
	var status string
	if phase == vjailbreakv1alpha1.VMMigrationPhaseSucceeded {
		status = "Succeeded"
	} else {
		status = "Failed"
	}

	MigrationCompletedTotal.WithLabelValues(vmName, status, namespace).Inc()
	UpdateMigrationPhase(migrationName, vmName, namespace, agentName, phase)

	// Clear duration metric for completed migrations
	MigrationDurationSeconds.DeleteLabelValues(migrationName, vmName, namespace)
}

// SetExpectedDuration sets the expected duration threshold for alerting
func SetExpectedDuration(namespace string, durationSeconds float64) {
	MigrationExpectedDurationSeconds.WithLabelValues(namespace).Set(durationSeconds)
}

// CleanupMigrationMetrics removes all metrics for a specific migration (when CR is deleted)
func CleanupMigrationMetrics(migrationName, vmName, namespace, migrationPlan, agentName string) {
	// Clean up all phase metrics
	allPhases := []vjailbreakv1alpha1.VMMigrationPhase{
		vjailbreakv1alpha1.VMMigrationPhasePending,
		vjailbreakv1alpha1.VMMigrationPhaseValidating,
		vjailbreakv1alpha1.VMMigrationPhaseValidationFailed,
		vjailbreakv1alpha1.VMMigrationPhaseAwaitingDataCopyStart,
		vjailbreakv1alpha1.VMMigrationPhaseCopying,
		vjailbreakv1alpha1.VMMigrationPhaseCopyingChangedBlocks,
		vjailbreakv1alpha1.VMMigrationPhaseConvertingDisk,
		vjailbreakv1alpha1.VMMigrationPhaseAwaitingCutOverStartTime,
		vjailbreakv1alpha1.VMMigrationPhaseAwaitingAdminCutOver,
		vjailbreakv1alpha1.VMMigrationPhaseSucceeded,
		vjailbreakv1alpha1.VMMigrationPhaseFailed,
		vjailbreakv1alpha1.VMMigrationPhaseUnknown,
	}

	for _, phase := range allPhases {
		MigrationPhaseGauge.DeleteLabelValues(migrationName, vmName, string(phase), namespace, agentName)
	}

	MigrationDurationSeconds.DeleteLabelValues(migrationName, vmName, namespace)
	MigrationInfo.DeleteLabelValues(migrationName, vmName, namespace, migrationPlan, agentName)
}
