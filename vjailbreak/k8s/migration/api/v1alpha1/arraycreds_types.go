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

// DatastoreInfo holds datastore information including backing device NAA
type DatastoreInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Capacity    int64  `json:"capacity"`
	FreeSpace   int64  `json:"freeSpace"`
	BackingNAA  string `json:"backingNAA"`  // NAA identifier of the backing LUN (for VMFS datastores)
	BackingUUID string `json:"backingUUID"` // UUID of the backing device
	MoID        string `json:"moID"`        // Managed object ID of the datastore
}

// ArrayCredsInfo holds the actual storage array credentials after decoding from secret
type ArrayCredsInfo struct {
	// Hostname is the storage array IP address or hostname
	Hostname string
	// Username is the storage array username
	Username string
	// Password is the storage array password
	Password string
	// SkipSSLVerification is whether to skip SSL certificate verification
	SkipSSLVerification bool
}

// ArrayCredsSpec defines the desired state of ArrayCreds
type ArrayCredsSpec struct {
	// VendorType is the storage array vendor type (e.g., pure, ontap, hpalletra)
	VendorType string `json:"vendorType"`

	// SecretRef is the reference to the Kubernetes secret holding storage array credentials
	SecretRef corev1.ObjectReference `json:"secretRef,omitempty"`

	// OpenStackMapping is the openstack mapping for this array
	OpenStackMapping OpenstackMapping `json:"openstackMapping,omitempty"`

	// AutoDiscovered indicates if this ArrayCreds was auto-discovered from OpenStack
	// +optional
	AutoDiscovered bool `json:"autoDiscovered,omitempty"`
}

// OpenstackMapping holds the OpenStack Cinder configuration mapping
type OpenstackMapping struct {
	// VolumeType is the Cinder volume type associated with this mapping
	VolumeType string `json:"volumeType"`
	// CinderBackendName is the Cinder backend name for this mapping
	// This is the backend configured in cinder.conf (e.g., "pure-01")
	CinderBackendName string `json:"cinderBackendName"`
	// CinderBackendPool is the pool name within the backend (optional)
	CinderBackendPool string `json:"cinderBackendPool,omitempty"`
	// CinderHost is the full Cinder host string for manage API
	// Format: hostname@backend or hostname@backend#pool (e.g., "pcd-ce@pure-iscsi-1#vt-pure-iscsi")
	CinderHost string `json:"cinderHost,omitempty"`
}

// ArrayCredsStatus defines the observed state of ArrayCreds
type ArrayCredsStatus struct {
	// ArrayValidationStatus is the status of the storage array validation
	// Possible values: Pending, Succeeded, Failed, AwaitingCredentials
	ArrayValidationStatus string `json:"arrayValidationStatus,omitempty"`
	// ArrayValidationMessage is the message associated with the storage array validation
	ArrayValidationMessage string `json:"arrayValidationMessage,omitempty"`
	// DataStore is the list of datastores associated with this array
	DataStore []DatastoreInfo `json:"dataStore,omitempty"`
	// Phase indicates the current phase of the ArrayCreds
	// Possible values: Discovered, Configured, Validated, Failed
	Phase string `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=`.spec.vendorType`,name=Vendor,type=string
// +kubebuilder:printcolumn:JSONPath=`.spec.openstackMapping.volumeType`,name=VolumeType,type=string
// +kubebuilder:printcolumn:JSONPath=`.spec.openstackMapping.cinderBackendName`,name=Backend,type=string
// +kubebuilder:printcolumn:JSONPath=`.status.phase`,name=Phase,type=string
// +kubebuilder:printcolumn:JSONPath=`.status.arrayValidationStatus`,name=Status,type=string

// ArrayCreds is the Schema for the storage array credentials API that defines authentication
// and connection details for storage arrays. It provides a secure way to store and validate
// storage array credentials for use in migration operations, supporting multiple vendors, but as of now
// We have qualified pure storage flash array.
type ArrayCreds struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArrayCredsSpec   `json:"spec,omitempty"`
	Status ArrayCredsStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ArrayCredsList contains a list of ArrayCreds
type ArrayCredsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArrayCreds `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArrayCreds{}, &ArrayCredsList{})
}
