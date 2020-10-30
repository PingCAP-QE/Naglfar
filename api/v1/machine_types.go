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
	"path"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	NVMEKind  DiskKind = "nvme"
	OtherKind          = "other"
)

// +kubebuilder:validation:Enum=nvme;other
type DiskKind string

type ReserveResources struct {
	// default 100
	// +optional
	CPUPercent int32 `json:"cpuPercent"`

	// default 1 GiB
	// +optional
	Memory BytesSize `json:"memory"`
}

type StorageDevice struct {
	Filesystem string    `json:"filesystem"`
	Total      BytesSize `json:"total"`
	Used       BytesSize `json:"used"`
	MountPoint string    `json:"mountPoint"`
}

type MachineInfo struct {
	Hostname       string                   `json:"hostname"`
	Architecture   string                   `json:"architecture"`
	Threads        int32                    `json:"threads"`
	Memory         BytesSize                `json:"memory"`
	StorageDevices map[string]StorageDevice `json:"devices,omitempty"`
}

type DiskResource struct {
	Size BytesSize `json:"size"`
	Kind DiskKind  `json:"kind"`
}

type AvailableResource struct {
	Memory     BytesSize               `json:"memory"`
	CPUPercent int32                   `json:"cpuPercent"`
	Disks      map[string]DiskResource `json:"disks,omitempty"`
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
	Timeout Duration `json:"timeout"`

	// +optional
	Reserve *ReserveResources `json:"reserve"`
}

// MachineStatus defines the observed state of Machine
type MachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	Info *MachineInfo `json:"info,omitempty"`

	// +optional
	TestResources []corev1.ObjectReference `json:"testResources,omitempty"`
}

// +kubebuilder:object:root=true

// Machine is the Schema for the machines API
// +kubebuilder:subresource:status
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

func (r *Machine) Available() *AvailableResource {
	if r.Status.Info == nil {
		return nil
	}

	available := new(AvailableResource)
	available.Memory = r.Status.Info.Memory.Sub(r.Spec.Reserve.Memory)
	available.CPUPercent = r.Status.Info.Threads*100 - r.Spec.Reserve.CPUPercent
	available.Disks = make(map[string]DiskResource)

	for device, disk := range r.Status.Info.StorageDevices {
		if disk.Used.Unwrap() != 0 {
			continue
		}

		diskResource := DiskResource{}
		diskResource.Size = disk.Total

		if strings.HasPrefix(path.Base(device), "nvme") {
			diskResource.Kind = NVMEKind
		} else {
			diskResource.Kind = OtherKind
		}

		available.Disks[device] = diskResource
	}

	return available
}

func init() {
	SchemeBuilder.Register(&Machine{}, &MachineList{})
}
