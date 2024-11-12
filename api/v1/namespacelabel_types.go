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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespacelabelSpec defines the desired state of Namespacelabel
type NamespacelabelSpec struct {
	Labels map[string]string `json:"labels,omitempty"`
}

// NamespacelabelStatus defines the observed state of Namespacelabel object - the Namespacelabel object grant permissions for users to label their namespaces
type NamespacelabelStatus struct {
	AppliedLabels map[string]string `json:"appliedLabels,omitempty"`

	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// List of conditions for better tracking of status
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// List of conditions for better tracking of status
	SkippedLabels map[string]string `json:"skippedLabels,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Namespacelabel is an object which grant the option to give permissions for user to label his namespace.
type Namespacelabel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NamespacelabelSpec   `json:"spec,omitempty"`
	Status NamespacelabelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NamespacelabelList contains a list of Namespacelabel
type NamespacelabelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Namespacelabel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Namespacelabel{}, &NamespacelabelList{})
}
