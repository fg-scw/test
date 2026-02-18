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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VMwareClusterPhase represents the lifecycle phase of a VMware cluster during the migration process,
// tracking its progression from initial discovery through running state to completion or failure.
// This status tracking enables monitoring of the overall migration progress at the cluster level.
type VMwareClusterPhase string

const (
	// VMwareClusterPending is the initial phase of a VMwareCluster
	VMwareClusterPending VMwareClusterPhase = "Pending"
	// VMwareClusterRunning is the phase of a VMwareCluster when it is running
	VMwareClusterRunning VMwareClusterPhase = "Running"
	// VMwareClusterFailed is the phase of a VMwareCluster when it fails
	VMwareClusterFailed VMwareClusterPhase = "Failed"
	// VMwareClusterCompleted is the final phase of a VMwareCluster
	VMwareClusterCompleted VMwareClusterPhase = "Completed"
)

// VMwareClusterSpec defines the desired state of VMwareCluster
type VMwareClusterSpec struct {
	// Name is the name of the VMware cluster
	Name string `json:"name,omitempty"`
	// Hosts is the list of hosts in the VMware cluster
	Hosts []string `json:"hosts,omitempty"`
}

// VMwareClusterStatus defines the observed state of VMwareCluster
type VMwareClusterStatus struct {
	// Phase is the current phase of the VMwareCluster
	Phase VMwareClusterPhase `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VMwareCluster is the Schema for the vmwareclusters API that represents a VMware vSphere cluster
// in the source environment. It tracks cluster configuration, member hosts, and migration status
// as part of the VMware to Platform9 Distributed Cloud migration process. VMwareCluster resources
// serve as source components that are migrated to corresponding PCDCluster resources in the target environment.
type VMwareCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VMwareClusterSpec   `json:"spec,omitempty"`
	Status VMwareClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VMwareClusterList contains a list of VMwareCluster
type VMwareClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VMwareCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VMwareCluster{}, &VMwareClusterList{})
}
