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

// ChaosApplySpec defines the desired state of ChaosApply
type ChaosApplySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ChaosApply. Edit ChaosApply_types.go to remove/update
	Request string `json:"request"`
	Rules   string `json:"rules"`
}

// ChaosApplyStatus defines the observed state of ChaosApply
type ChaosApplyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ApplyTable map[string]*ChaosResourcesInstance
}

type ChaosResourcesInstance struct {
	Tcs        bool `json:"tcs,omitempty"`
	TimeOffset bool `json:"timeOffset,omitempty"`
	Stress     bool `json:"stress,omitempty"`
	Io         bool `json:"io,omitempty"`
}

type ChaosApplyInstance struct {
	Instance  string `json:"instance"`
	StartTime int64  `json:"startTime"`
}

// +kubebuilder:object:root=true

// ChaosApply is the Schema for the chaosapplies API
// +kubebuilder:subresource:status
type ChaosApply struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChaosApplySpec   `json:"spec,omitempty"`
	Status ChaosApplyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ChaosApplyList contains a list of ChaosApply
type ChaosApplyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChaosApply `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChaosApply{}, &ChaosApplyList{})
}
