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

// NetworkMappingSpec defines the desired state of NetworkMapping including
// mappings between VMware and OpenStack networks
type NetworkMappingSpec struct {
	// Networks is the list of network mappings between source (VMware) and target (OpenStack) environments
	Networks []Network `json:"networks"`
}

// Network represents a mapping between source and target networks
type Network struct {
	// Source is the name of the source network in VMware
	Source string `json:"source"`
	// Target is the name of the target network in OpenStack
	Target string `json:"target"`
}

// NetworkMappingStatus defines the observed state of NetworkMapping
type NetworkMappingStatus struct {
	// NetworkmappingValidationStatus indicates the validation status of the network mapping
	// Valid states include: "Valid", "Invalid", "Pending", "ValidationFailed"
	NetworkmappingValidationStatus string `json:"networkMappingValidationStatus,omitempty"`
	// NetworkmappingValidationMessage provides detailed validation information including
	// information about available networks and any validation errors
	NetworkmappingValidationMessage string `json:"networkMappingValidationMessage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.networkMappingValidationStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// NetworkMapping is the Schema for the networkmappings API that defines
// mappings between VMware and OpenStack networks to be used during migration
type NetworkMapping struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkMappingSpec   `json:"spec,omitempty"`
	Status NetworkMappingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkMappingList contains a list of NetworkMapping
type NetworkMappingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkMapping `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkMapping{}, &NetworkMappingList{})
}
