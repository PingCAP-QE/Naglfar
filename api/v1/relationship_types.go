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
	"github.com/PingCAP-QE/Naglfar/pkg/ref"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RelationshipSpec defines the desired state of Relationship
type RelationshipSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// RelationshipStatus defines the observed state of Relationship
type RelationshipStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	OneToMany map[string]RefList `json:"oneToMany,omitempty"`
	ManyToOne map[string]ref.Ref `json:"manyToOne,omitempty"`
}

type RefList []ref.Ref

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Relationship is the Schema for the relationships API
type Relationship struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RelationshipSpec   `json:"spec,omitempty"`
	Status RelationshipStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RelationshipList contains a list of Relationship
type RelationshipList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Relationship `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Relationship{}, &RelationshipList{})
}
