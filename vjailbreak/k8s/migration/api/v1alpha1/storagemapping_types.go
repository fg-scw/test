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

// StorageMappingSpec defines the desired state of StorageMapping including
// mappings between VMware and OpenStack storage types
type StorageMappingSpec struct {
	// Storages is a list of storage mappings between source (VMware) and target (OpenStack) environments
	Storages []Storage `json:"storages"`
}

// Storage represents a mapping between source and target storage types
type Storage struct {
	// Source is the name of the source storage type in VMware
	Source string `json:"source"`
	// Target is the name of the target storage type in OpenStack
	Target string `json:"target"`
}

// StorageMappingStatus defines the observed state of StorageMapping
type StorageMappingStatus struct {
	// StoragemappingValidationStatus indicates the validation status of the storage mapping
	// Valid states include: "Valid", "Invalid", "Pending", "ValidationFailed"
	StoragemappingValidationStatus string `json:"storageMappingValidationStatus,omitempty"`
	// StoragemappingValidationMessage provides detailed validation information including
	// information about available storage types and any validation errors
	StoragemappingValidationMessage string `json:"storageMappingValidationMessage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.storageMappingValidationStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// StorageMapping is the Schema for the storagemappings API that defines
// mappings between VMware and OpenStack storage types to be used during migration
type StorageMapping struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageMappingSpec   `json:"spec,omitempty"`
	Status StorageMappingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StorageMappingList contains a list of StorageMapping
type StorageMappingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StorageMapping `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StorageMapping{}, &StorageMappingList{})
}
