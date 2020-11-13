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
	"fmt"
	"path"
	"sort"
	"strings"

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
	// default 1
	// +optional
	Cores int32 `json:"cores"`

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
	Size      BytesSize `json:"size"`
	Kind      DiskKind  `json:"kind"`
	MountPath string    `json:"mountPath"`
}

type AvailableResource struct {
	Memory     BytesSize               `json:"memory"`
	IdleCPUSet []int                   `json:"idleCPUSet,omitempty"`
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
	SSHPort int `json:"sshPort"`

	// +kubebuilder:validation:Minimum=0
	// default 2375 (unencrypted) or 2376(encrypted)
	// +optional
	DockerPort int `json:"dockerPort"`

	// +optional
	DockerVersion string `json:"dockerVersion,omitempty"`

	// default false
	// +optional
	DockerTLS bool `json:"dockerTLS"`

	// default 10s
	// +optional
	Timeout Duration `json:"timeout"`

	// +optional
	Reserve *ReserveResources `json:"reserve"`

	// +optional
	ExclusiveDisks []string `json:"exclusiveDisks,omitempty"`
}

// MachineStatus defines the observed state of Machine
type MachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	Info *MachineInfo `json:"info,omitempty"`
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
	available.IdleCPUSet = deleteCPUSet(makeCPUSet(r.Status.Info.Threads), makeCPUSet(r.Spec.Reserve.Cores))
	available.Disks = make(map[string]DiskResource)

	exclusiveSet := make(map[string]bool)

	for _, device := range r.Spec.ExclusiveDisks {
		exclusiveSet[device] = true
	}

	for device, disk := range r.Status.Info.StorageDevices {
		if _, ok := exclusiveSet[device]; ok {
			diskResource := DiskResource{
				Size:      disk.Total.Sub(disk.Used),
				MountPath: disk.MountPoint,
			}

			if strings.HasPrefix(path.Base(device), "nvme") {
				diskResource.Kind = NVMEKind
			} else {
				diskResource.Kind = OtherKind
			}

			available.Disks[device] = diskResource
		}
	}

	return available
}

func makeCPUSet(threads int32) (cpuSet []int) {
	for i := 0; i < int(threads); i++ {
		cpuSet = append(cpuSet, i)
	}
	return
}

func deleteCPUSet(cpuSet []int, allocSet []int) (idleCPUSet []int) {
	set := make(map[int]struct{})
	for _, core := range cpuSet {
		set[core] = struct{}{}
	}
	for _, core := range allocSet {
		delete(set, core)
	}
	for core := range set {
		idleCPUSet = append(idleCPUSet, core)
	}
	sort.Slice(idleCPUSet, func(i, j int) bool {
		return idleCPUSet[i] < idleCPUSet[j]
	})
	return
}

func cpuSetStr(cpuSet []int) (str string) {
	for i, core := range cpuSet {
		if i == 0 {
			str = fmt.Sprintf("%d", core)
		} else {
			str = fmt.Sprintf("%s,%d", str, core)
		}
	}
	return
}

func (r *Machine) Rest(resources ResourceRefList) (rest *AvailableResource) {
	rest = r.Available()

	if rest == nil {
		return
	}

	for _, refer := range resources {
		rest.IdleCPUSet = deleteCPUSet(rest.IdleCPUSet, refer.Binding.CPUSet)
		rest.Memory = rest.Memory.Sub(refer.Binding.Memory)

		if len(refer.Binding.Disks) != 0 {
			for _, diskBinding := range refer.Binding.Disks {
				if _, ok := rest.Disks[diskBinding.Device]; !ok {
					// something wrong,
					rest = nil
					return
				}

				delete(rest.Disks, diskBinding.Device)
			}
			continue
		}
	}

	return
}

func (r *Machine) DockerURL() string {
	scheme := "http"
	if r.Spec.DockerTLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, r.Spec.Host, r.Spec.DockerPort)
}

func init() {
	SchemeBuilder.Register(&Machine{}, &MachineList{})
}
