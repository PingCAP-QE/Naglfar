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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NVMEKind  DiskKind = "nvme"
	OtherKind          = "other"
)

// +kubebuilder:validation:Enum=nvme;other
type DiskKind string

type DiskSpec struct {
	Name      string   `json:"name"`
	Kind      DiskKind `json:"kind"`
	Size      int64    `json:"size"`
	MountPath string   `json:"mountPath"`
}

type DiskStatus struct {
	Kind      DiskKind `json:"kind"`
	Size      int64    `json:"size"`
	Path      string   `json:"path"`
	MountPath string   `json:"mountPath"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TestResourceSpec defines the desired state of TestResource
type TestResourceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Memory int64 `json:"memory"`

	CPUPercent int32 `json:"cpuPercent"`

	// +optional
	Disks []*DiskSpec `json:"disks"`

	// If sets, it means the static machine is required
	// +optional
	TestMachineResource string `json:"testMachineResource,omitempty"`
}

// TestResourceStatus defines the observed state of TestResource
type TestResourceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	Initialized bool `json:"initialized"`

	// +optional
	HostMachine string `json:"hostMachine"`

	// ClusterIP is the ip address of the container in the overlay(or calico) network
	// +optional
	ClusterIP string `json:"clusterIP"`

	// +optional
	DiskStat []*DiskStatus `json:"diskStat,omitempty"`

	// +optional
	Username string `json:"username"`

	// +optional
	Password string `json:"password"`

	// +optional
	SSHPort int `json:"sshPort"`
}

// +kubebuilder:object:root=true

// TestResource is the Schema for the testresources API
type TestResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestResourceSpec   `json:"spec,omitempty"`
	Status TestResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TestResourceList contains a list of TestResource
type TestResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TestResource{}, &TestResourceList{})
}
