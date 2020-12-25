// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ChaosBindingSpec defines the desired state of ChaosBinding
type ChaosBindingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// reference of resource
	Resource string `json:"resources"`
}

// ChaosBindingStatus defines the observed state of ChaosBinding
type ChaosBindingStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ChaosApply ~>
	Applies map[string]*ChaosResources `json:"applies"`
}

type ChaosResources struct {
}

// +kubebuilder:object:root=true

// ChaosBinding is the Schema for the chaosbindings API
// +kubebuilder:subresource:status
type ChaosBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChaosBindingSpec   `json:"spec,omitempty"`
	Status ChaosBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ChaosBindingList contains a list of ChaosBinding
type ChaosBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChaosBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChaosBinding{}, &ChaosBindingList{})
}
