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

// ClusterMigrationPhase represents the current phase of the cluster migration
// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed;Paused
type ClusterMigrationPhase string

const (
	// ClusterMigrationPhasePending indicates the migration is waiting to start
	ClusterMigrationPhasePending ClusterMigrationPhase = "Pending"
	// ClusterMigrationPhaseRunning indicates the migration is in progress
	ClusterMigrationPhaseRunning ClusterMigrationPhase = "Running"
	// ClusterMigrationPhaseSucceeded indicates the migration has completed successfully
	ClusterMigrationPhaseSucceeded ClusterMigrationPhase = "Succeeded"
	// ClusterMigrationPhaseFailed indicates the migration has failed
	ClusterMigrationPhaseFailed ClusterMigrationPhase = "Failed"
	// ClusterMigrationPhasePaused indicates the migration is paused
	ClusterMigrationPhasePaused ClusterMigrationPhase = "Paused"
)

// ClusterMigrationSpec defines the desired state of ClusterMigration
type ClusterMigrationSpec struct {
	// ClusterName is the name of the vCenter cluster to be migrated
	ClusterName string `json:"clusterName"`

	// ESXIMigrationSequence is the sequence of ESXi hosts to be migrated
	ESXIMigrationSequence []string `json:"esxiMigrationSequence"`

	// OpenstackCredsRef is the reference to the OpenStack credentials
	OpenstackCredsRef corev1.LocalObjectReference `json:"openstackCredsRef"`

	// VMwareCredsRef is the reference to the VMware credentials
	VMwareCredsRef corev1.LocalObjectReference `json:"vmwareCredsRef"`

	// RollingMigrationPlanRef is the reference to the RollingMigrationPlan
	RollingMigrationPlanRef corev1.LocalObjectReference `json:"rollingMigrationPlanRef"`
}

// ClusterMigrationStatus defines the observed state of ClusterMigration
type ClusterMigrationStatus struct {
	// CurrentESXi is the name of the current ESXi host being migrated
	CurrentESXi string `json:"currentESXi"`
	// Phase is the current phase of the migration
	Phase ClusterMigrationPhase `json:"phase"`
	// Message is the message associated with the current state of the migration
	Message string `json:"message"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ClusterMigration is the Schema for the clustermigrations API that orchestrates the migration
// of an entire VMware vCenter cluster to Platform9 Distributed Cloud (PCD).
// It manages the ordered migration of ESXi hosts within a cluster, tracking progress
// and maintaining references to required credentials and migration plans.
// ClusterMigration resources coordinate with ESXIMigration resources to ensure
// hosts are migrated in the correct sequence with proper dependency management.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Current ESXI",type="string",JSONPath=".status.currentESXI"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ClusterMigration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterMigrationSpec   `json:"spec,omitempty"`
	Status ClusterMigrationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterMigrationList contains a list of ClusterMigration
type ClusterMigrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterMigration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterMigration{}, &ClusterMigrationList{})
}
