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

// VMwareHostSpec defines the desired state of VMwareHost
type VMwareHostSpec struct {
	// Name of the host
	Name string `json:"name,omitempty"`
	// Hardware UUID of the host
	HardwareUUID string `json:"hardwareUuid,omitempty"`
	// Host config ID of the host
	HostConfigID string `json:"hostConfigId,omitempty"`
	// Cluster name of the host
	ClusterName string `json:"clusterName,omitempty"`
}

// VMwareHostStatus defines the observed state of VMwareHost
type VMwareHostStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VMwareHost is the Schema for the vmwarehosts API that represents a VMware ESXi host
// in the source environment. It tracks host configuration, hardware identification, and cluster membership
// as part of the VMware to Platform9 Distributed Cloud migration process. VMwareHost resources are
// source components that are migrated to PCDHost resources in the target environment and are automatically
// removed from vCenter inventory after all VMs have been migrated off and the host is in maintenance mode.
type VMwareHost struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VMwareHostSpec   `json:"spec,omitempty"`
	Status VMwareHostStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VMwareHostList contains a list of VMwareHost
type VMwareHostList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VMwareHost `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VMwareHost{}, &VMwareHostList{})
}
