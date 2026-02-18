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

// BMCProviderName represents the supported Bare Metal Controller provider types
// that can be used for provisioning physical hosts during migration.
type BMCProviderName string

// BootSource defines the operating system boot source configuration for provisioning
// physical hosts through the BMC provider.
type BootSource struct {
	//+kubebuilder:default="jammy"
	// Release is the OS release version to be used (e.g., "jammy" for Ubuntu 22.04)
	Release string `json:"release"`
}

const (
	// MAASProvider represents the Metal As A Service provider for bare metal provisioning
	MAASProvider BMCProviderName = "MAAS"
)

// BMConfigSpec defines the desired state of BMConfig
type BMConfigSpec struct {
	// UserName is the username for the BM server
	UserName string `json:"userName,omitempty"`
	// Password is the password for the BM server
	Password string `json:"password,omitempty"`
	// APIKey is the API key for the BM server
	APIKey string `json:"apiKey"`
	// APIUrl is the API URL for the BM server
	APIUrl string `json:"apiUrl"`
	// Insecure is a boolean indicating whether to use insecure connection
	//+kubebuilder:default=false
	Insecure bool `json:"insecure,omitempty"`
	// ProviderType is the BMC provider type
	//+kubebuilder:default="MAAS"
	ProviderType BMCProviderName `json:"providerType"`
	// UserDataSecretRef is the reference to the secret containing user data for the BMC
	UserDataSecretRef corev1.SecretReference `json:"userDataSecretRef,omitempty"`
	// BootSource is the boot source for the BMC
	BootSource BootSource `json:"bootSource,omitempty"`
}

// BMConfigStatus defines the observed state of BMConfig
type BMConfigStatus struct {
	// ValidationStatus is the status of the validation
	ValidationStatus string `json:"validationStatus,omitempty"`
	// ValidationMessage is the message associated with the validation
	ValidationMessage string `json:"validationMessage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BMConfig is the Schema for the bmconfigs API that defines authentication and configuration
// details for Bare Metal Controller (BMC) providers such as MAAS. It contains credentials,
// connection information, and boot source configurations needed to provision physical hosts
// for use during the ESXi to PCD migration process. BMConfig enables the automatic
// provisioning of PCD hosts as replacement infrastructure for migrated ESXi hosts.
type BMConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BMConfigSpec   `json:"spec,omitempty"`
	Status BMConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BMConfigList contains a list of BMConfig
type BMConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BMConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BMConfig{}, &BMConfigList{})
}
