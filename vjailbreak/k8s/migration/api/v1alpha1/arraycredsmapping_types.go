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

// ArrayCredsMappingSpec defines the desired state of ArrayCredsMapping including
// mappings between VMware datastores and ArrayCreds for StorageAcceleratedCopy data copy
type ArrayCredsMappingSpec struct {
	// Mappings is a list of datastore to ArrayCreds mappings
	Mappings []DatastoreArrayCredsMapping `json:"mappings"`
}

// DatastoreArrayCredsMapping represents a mapping between a VMware datastore and ArrayCreds
type DatastoreArrayCredsMapping struct {
	// Source is the name of the source datastore in VMware
	Source string `json:"source"`
	// Target is the name of the ArrayCreds resource to use for this datastore
	Target string `json:"target"`
}

// ArrayCredsMappingStatus defines the observed state of ArrayCredsMapping
type ArrayCredsMappingStatus struct {
	// ValidationStatus indicates the validation status of the ArrayCreds mapping
	// Valid states include: "Valid", "Invalid", "Pending", "ValidationFailed"
	ValidationStatus string `json:"validationStatus,omitempty"`
	// ValidationMessage provides detailed validation information including
	// information about available ArrayCreds and any validation errors
	ValidationMessage string `json:"validationMessage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.validationStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ArrayCredsMapping is the Schema for the arraycredsmappings API that defines
// mappings between VMware datastores and ArrayCreds for StorageAcceleratedCopy storage migration
type ArrayCredsMapping struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArrayCredsMappingSpec   `json:"spec,omitempty"`
	Status ArrayCredsMappingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ArrayCredsMappingList contains a list of ArrayCredsMapping
type ArrayCredsMappingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArrayCredsMapping `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArrayCredsMapping{}, &ArrayCredsMappingList{})
}
