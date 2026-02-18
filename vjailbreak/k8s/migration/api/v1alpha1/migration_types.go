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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VMMigrationPhase represents the current phase of the VM migration process from VMware to OpenStack,
// tracking the detailed progression through various stages including validation, data copying,
// disk conversion, and cutover. Each phase provides visibility into the migration's progress,
// enabling precise monitoring and troubleshooting of the migration workflow.
// +kubebuilder:validation:Enum=Pending;Validating;ValidationFailed;AwaitingDataCopyStart;CopyingBlocks;CopyingChangedBlocks;ConvertingDisk;AwaitingCutOverStartTime;AwaitingAdminCutOver;Succeeded;Failed;Unknown;ConnectingToESXi;CreatingInitiatorGroup;CreatingVolume;ImportingToCinder;MappingVolume;RescanningStorage;XCOPYInProgress
type VMMigrationPhase string

// MigrationConditionType represents the type of condition for a migration, used to track
// specific state conditions that may affect the migration process, such as resource availability,
// network connectivity, or authentication status.
type MigrationConditionType string

const (
	// VMMigrationPhasePending indicates the migration is waiting to start
	VMMigrationPhasePending VMMigrationPhase = "Pending"
	// VMMigrationPhaseValidating indicates the migration prerequisites are being validated
	VMMigrationPhaseValidating VMMigrationPhase = "Validating"
	// VMMigrationPhaseValidationFailed indicates the migration prerequisites validation failed
	VMMigrationPhaseValidationFailed VMMigrationPhase = "ValidationFailed"
	// VMMigrationPhaseAwaitingDataCopyStart indicates the migration is waiting to begin data copy
	VMMigrationPhaseAwaitingDataCopyStart VMMigrationPhase = "AwaitingDataCopyStart"
	// VMMigrationPhaseCopying indicates initial block copying is in progress
	VMMigrationPhaseCopying VMMigrationPhase = "CopyingBlocks"
	// VMMigrationPhaseCopyingChangedBlocks indicates copying of changed blocks is in progress
	VMMigrationPhaseCopyingChangedBlocks VMMigrationPhase = "CopyingChangedBlocks"
	// VMMigrationPhaseConvertingDisk indicates disk format conversion is in progress
	VMMigrationPhaseConvertingDisk VMMigrationPhase = "ConvertingDisk"
	// VMMigrationPhaseAwaitingCutOverStartTime indicates waiting for scheduled cutover time
	VMMigrationPhaseAwaitingCutOverStartTime VMMigrationPhase = "AwaitingCutOverStartTime"
	// VMMigrationPhaseAwaitingAdminCutOver indicates waiting for admin to initiate cutover
	VMMigrationPhaseAwaitingAdminCutOver VMMigrationPhase = "AwaitingAdminCutOver"
	// VMMigrationPhaseSucceeded indicates the migration completed successfully
	VMMigrationPhaseSucceeded VMMigrationPhase = "Succeeded"
	// VMMigrationPhaseFailed indicates the migration has failed
	VMMigrationPhaseFailed VMMigrationPhase = "Failed"
	// VMMigrationPhaseUnknown indicates the migration state is unknown
	VMMigrationPhaseUnknown VMMigrationPhase = "Unknown"

	// VMMigrationPhaseConnectingToESXi indicates SSH connection to ESXi host is being established (StorageAcceleratedCopy specific phase)
	VMMigrationPhaseConnectingToESXi VMMigrationPhase = "ConnectingToESXi"
	// VMMigrationPhaseCreatingInitiatorGroup indicates initiator group is being created/updated
	VMMigrationPhaseCreatingInitiatorGroup VMMigrationPhase = "CreatingInitiatorGroup"
	// VMMigrationPhaseCreatingVolume indicates volume is being created on storage array
	VMMigrationPhaseCreatingVolume VMMigrationPhase = "CreatingVolume"
	// VMMigrationPhaseImportingToCinder indicates volume is being imported to OpenStack Cinder
	VMMigrationPhaseImportingToCinder VMMigrationPhase = "ImportingToCinder"
	// VMMigrationPhaseMappingVolume indicates volume is being mapped to ESXi host
	VMMigrationPhaseMappingVolume VMMigrationPhase = "MappingVolume"
	// VMMigrationPhaseRescanningStorage indicates ESXi storage adapters are being rescanned
	VMMigrationPhaseRescanningStorage VMMigrationPhase = "RescanningStorage"
	// VMMigrationPhaseStorageAcceleratedCopyInProgress indicates StorageAcceleratedCopy operation is in progress
	VMMigrationPhaseStorageAcceleratedCopyInProgress VMMigrationPhase = "StorageAcceleratedCopyInProgress"
)

// MigrationSpec defines the desired state of Migration
type MigrationSpec struct {
	// MigrationPlan is the name of the migration plan
	MigrationPlan string `json:"migrationPlan"`

	// PodRef is the name of the pod
	PodRef string `json:"podRef"`

	// VMName is the name of the VM getting migrated from VMWare to Openstack
	VMName string `json:"vmName"`

	// InitiateCutover is the flag to initiate cutover
	InitiateCutover bool `json:"initiateCutover"`

	// DisconnectSourceNetwork specifies whether to disconnect the source VM's network interfaces
	// after a successful migration to prevent network conflicts. Defaults to false.
	// +optional
	DisconnectSourceNetwork bool `json:"disconnectSourceNetwork,omitempty"`

	// AssignedIP is the comma-separated list of user-assigned IPs for cold migration
	// Format: "IP1,IP2,IP3" where each IP corresponds to a network interface by index
	AssignedIP string `json:"assignedIP,omitempty"`

	// MigrationType indicates whether this is a hot (live) or cold migration
	// +optional
	// +kubebuilder:validation:Enum=hot;cold;mock
	MigrationType string `json:"migrationType,omitempty"`
}

// MigrationStatus defines the observed state of Migration
type MigrationStatus struct {
	// Phase is the current phase of the migration
	Phase VMMigrationPhase `json:"phase"`

	// Conditions is the list of conditions of the migration object pod
	Conditions []corev1.PodCondition `json:"conditions,omitempty"`

	// AgentName is the name of the agent where migration is running
	AgentName string `json:"agentName,omitempty"`

	// CurrentDisk tracks which disk is currently being copied (e.g., "0", "1")
	// Extracted from migration pod events
	// +optional
	CurrentDisk string `json:"currentDisk,omitempty"`

	// TotalDisks is the total number of disks to be migrated
	// +optional
	TotalDisks int `json:"totalDisks,omitempty"`
	// Retryable indicates whether this migration can be retried when it fails.
	// Set to false for VMs with RDM (Raw Device Mapping) disks that share storage,
	// as RDM disk migration state prevents automatic retry.
	// +optional
	Retryable *bool `json:"retryable,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Agent Name",type="string",JSONPath=".status.agentName"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Migration is the Schema for the migrations API that represents a single virtual machine
// migration job from VMware to OpenStack. It tracks the complete migration lifecycle
// including validation, data transfer, disk conversion, and cutover phases. Migration resources
// provide detailed status monitoring, error handling, and manual intervention points like
// cutover initiation. Each Migration is associated with a MigrationPlan and executes on a specific
// VjailbreakNode agent, tracking progress via status updates and condition changes.
type Migration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of Migration
	Spec MigrationSpec `json:"spec,omitempty"`

	// Status defines the observed state of Migration
	Status MigrationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MigrationList contains a list of Migration
type MigrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of Migration objects
	Items []Migration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Migration{}, &MigrationList{})
}
