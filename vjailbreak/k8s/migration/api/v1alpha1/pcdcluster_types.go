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

// PCDClusterSpec defines the desired state of PCDCluster
type PCDClusterSpec struct {
	// ClusterName is the name of the PCD cluster
	ClusterName string `json:"clusterName,omitempty"`
	// Description is the description of the PCD cluster
	Description string `json:"description,omitempty"`
	// Hosts is the list of hosts in the PCD cluster
	Hosts []string `json:"hosts,omitempty"`
	// VMHighAvailability indicates if VM high availability is enabled
	VMHighAvailability bool `json:"vmHighAvailability,omitempty"`
	// EnableAutoResourceRebalancing indicates if auto resource rebalancing is enabled
	EnableAutoResourceRebalancing bool `json:"enableAutoResourceRebalancing,omitempty"`
	// RebalancingFrequencyMins defines how often rebalancing occurs in minutes
	RebalancingFrequencyMins int `json:"rebalancingFrequencyMins,omitempty"`
}

// PCDClusterStatus defines the observed state of PCDCluster
type PCDClusterStatus struct {
	// ClusterID is the ID of the PCD cluster
	ClusterID string `json:"clusterID,omitempty"`
	// AggregateID is the aggregate ID in the PCD cluster
	AggregateID int `json:"aggregateID,omitempty"`
	// CreatedAt indicates when the cluster was created
	CreatedAt string `json:"createdAt,omitempty"`
	// UpdatedAt indicates when the cluster was last updated
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PCDCluster is the Schema for the pcdclusters API that represents a Platform9 Distributed Cloud
// cluster in the migration system. It defines cluster configuration including host membership,
// high availability settings, and resource rebalancing policies. PCDCluster resources track both
// the desired state of clusters as part of migration planning and the observed state of actual
// PCD clusters in the target environment. They serve as destination targets for VM migrations
// from VMware ESXi environments.
type PCDCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PCDClusterSpec   `json:"spec,omitempty"`
	Status PCDClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PCDClusterList contains a list of PCDCluster
type PCDClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PCDCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PCDCluster{}, &PCDClusterList{})
}
