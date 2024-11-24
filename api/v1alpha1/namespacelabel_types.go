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

// NamespacelabelSpec defines the desired state of Namespacelabel
type NamespacelabelSpec struct {
	// Labels is a map of key-value pairs that should be applied to the target namespace.
	// The keys are the label names, and the values are the corresponding label values.
	Labels map[string]string `json:"labels,omitempty"`
}

// NamespacelabelStatus defines the observed state of Namespacelabel
type NamespacelabelStatus struct {
	// AppliedLabels represents the labels that were successfully applied to the namespace.
	// This map includes key-value pairs of all successfully applied labels.
	AppliedLabels map[string]string `json:"appliedLabels,omitempty"`

	// Conditions is a list of conditions that provide additional insight into the status of the Namespacelabel.
	// Conditions can include statuses like LabelsApplied, LabelsSkipped, and others.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// SkippedLabels represents the labels that could not be applied due to conflicts or other restrictions.
	// This map includes key-value pairs of all labels that were skipped.
	SkippedLabels map[string]string `json:"skippedLabels,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Namespacelabel is an object that grants permissions to users to label their namespace.
// This object allows users to specify and manage labels for namespaces they own.
type Namespacelabel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NamespacelabelSpec   `json:"spec,omitempty"`
	Status NamespacelabelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NamespacelabelList contains a list of Namespacelabel objects.
type NamespacelabelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Namespacelabel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Namespacelabel{}, &NamespacelabelList{})
}
