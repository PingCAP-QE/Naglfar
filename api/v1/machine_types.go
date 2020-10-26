/*


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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ReserveResources struct {
	// default 100
	// +optional
	CPUPercent int32 `json:"cpuPercent"`

	// +kubebuilder:validation:Minimum=0

	// default 1 << 30
	// +optional
	Memory int64 `json:"memory"`
}

type StorageDevice struct {
	Kind string `json:"kind"`
	Size int64  `json:"size"`
	Path string `json:"path"`
}

type TotalResources struct {
	Cores          int32            `json:"cores"`
	Memory         int64            `json:"memory"`
	StorageDevices []*StorageDevice `json:"devices"`
}

// MachineSpec defines the desired state of Machine
type MachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Username string `json:"username"`

	Password string `json:"password"`

	Host string `json:"host"`

	// +kubebuilder:validation:Minimum=0
	// default 22
	// +optional
	Port int `json:"port"`

	// default 10s
	// +optional
	Timeout string `json:"timeout"`

	// +optional
	Reserve *ReserveResources `json:"reserve"`
}

// MachineStatus defines the observed state of Machine
type MachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	Total *TotalResources `json:"total,omitempty"`

	// +optional
	TestResources []corev1.ObjectReference `json:"testResources,omitempty"`
}

// +kubebuilder:object:root=true

// Machine is the Schema for the machines API
type Machine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineSpec   `json:"spec,omitempty"`
	Status MachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MachineList contains a list of Machine
type MachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Machine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Machine{}, &MachineList{})
}
