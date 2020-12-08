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

	"github.com/PingCAP-QE/Naglfar/pkg/ref"
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
	MachineToResources map[string]ResourceRefList `json:"machineToResources"`
	ResourceToMachine  map[string]MachineRef      `json:"resourceToMachine"`

	// accepted requests(satisfied and not deployed)
	AcceptedRequests AcceptResources `json:"acceptedRequests"`

	// exclusive lock: machine name -> request ref
	MachineLocks map[string]ref.Ref `json:"machineLock"`
}

type AcceptResources []ref.Ref

// IfExist predicates whether r is exist
func (a *AcceptResources) IfExist(r ref.Ref) bool {
	if a == nil {
		return false
	}
	for _, item := range *a {
		if item == r {
			return true
		}
	}
	return false
}

func (a *AcceptResources) Remove(r ref.Ref) {
	newA := make(AcceptResources, 0)
	for _, item := range *a {
		if item != r {
			newA = append(newA, item)
		}
	}
	*a = newA
}

type ResourceRefList []ResourceRef

func (r ResourceRefList) Clone() ResourceRefList {
	resources := make(ResourceRefList, 0)
	for _, ref := range r {
		resources = append(resources, ref)
	}
	return resources
}

type MachineRef struct {
	ref.Ref `json:",inline"`
	Binding ResourceBinding `json:"binding"`
}

type ResourceRef struct {
	ref.Ref `json:",inline"`
	Binding ResourceBinding `json:"binding"`
}

type ResourceBinding struct {
	Memory BytesSize              `json:"memory"`
	CPUSet []int                  `json:"cpuSet,omitempty"`
	Disks  map[string]DiskBinding `json:"disks,omitempty"`
}

type DiskBinding struct {
	Kind       DiskKind  `json:"kind"`
	Size       BytesSize `json:"size"`
	Device     string    `json:"device"`
	OriginPath string    `json:"originPath"`
	MountPath  string    `json:"mountPath"`
}

// +kubebuilder:object:root=true

// Relationship is the Schema for the relationships API
// +kubebuilder:subresource:status
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
