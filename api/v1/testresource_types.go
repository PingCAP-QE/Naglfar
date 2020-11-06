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
	"strings"

	"github.com/PingCAP-QE/Naglfar/pkg/ref"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourcePending       ResourceState = "pending"
	ResourceFail                        = "fail"
	ResourceUninitialized               = "uninitialized"
	ResourceReady                       = "ready"
	ResourceFinish                      = "finish"
)

const cleanerImage = "alpine:latest"

// +kubebuilder:validation:Enum=pending;fail;uninitialized;ready;finish
type ResourceState string

type DiskSpec struct {
	// default /mnt/<name>
	// +optional
	MountPath string `json:"mountPath"`

	// +optional
	Kind DiskKind `json:"kind"`

	// +optional
	Size BytesSize `json:"size"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TestResourceSpec defines the desired state of TestResource
type TestResourceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Memory BytesSize `json:"memory"`

	CPUPercent int32 `json:"cpuPercent"`

	// +optional
	MachineSelector string `json:"machineSelector,omitempty"`

	// +optional
	Disks map[string]DiskSpec `json:"disks,omitempty"`

	// If sets, it means the static machine is required
	// +optional
	TestMachineResource string `json:"testMachineResource,omitempty"`
}

// TestResourceStatus defines the observed state of TestResource
type TestResourceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// default pending
	// +optional
	State ResourceState `json:"state"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	Commands []string `json:"commands,omitempty"`

	// ClusterIP is the ip address of the container in the overlay(or calico) network
	// +optional
	ClusterIP string `json:"clusterIP"`

	// +optional
	Username string `json:"username"`

	// +optional
	Password string `json:"password"`

	// +optional
	SSHPort int `json:"sshPort"`

	// TODO: delete me
	// +optional
	HostMachine ref.Ref `json:"hostMachine"`
}

// +kubebuilder:object:root=true

// TestResource is the Schema for the testresources API
// +kubebuilder:subresource:status
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

func (r *TestResource) ContainerName() string {
	return fmt.Sprintf("%s.%s", r.Namespace, r.Name)
}

func (r *TestResource) ContainerCleanerName() string {
	return fmt.Sprintf("%s.%s-cleaner", r.Namespace, r.Name)
}

func (r *TestResource) ContainerConfig(binding *ResourceBinding) (*container.Config, *container.HostConfig) {
	mounts := make([]mount.Mount, 0)
	for _, disk := range binding.Disks {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: disk.OriginPath,
			Target: disk.MountPath,
		})
	}

	config := &container.Config{
		Image: r.Status.Image,
	}

	hostConfig := &container.HostConfig{
		Mounts: mounts,
		Resources: container.Resources{
			Memory:   binding.Memory.Unwrap(),
			CPUQuota: int64(binding.CPUPercent) * 1000,
		},
	}

	if len(r.Status.Commands) != 0 {
		script := strings.Join(r.Status.Commands, ";")
		config.Cmd = []string{"bash", "-c", script}
	}

	return config, hostConfig
}

func (r *TestResource) ContainerCleanerConfig(binding *ResourceBinding) (*container.Config, *container.HostConfig) {
	mounts := make([]mount.Mount, 0)
	for _, disk := range binding.Disks {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: disk.OriginPath,
			Target: disk.MountPath,
		})
	}

	config := &container.Config{
		Image: cleanerImage,
	}

	hostConfig := &container.HostConfig{
		Mounts: mounts,
	}

	if len(mounts) > 0 {
		config.Cmd = []string{"rm", "-rf"}
		for _, mnt := range mounts {
			config.Cmd = append(config.Cmd, path.Join(mnt.Target, "*"))
		}
	}

	return config, hostConfig
}

func init() {
	SchemeBuilder.Register(&TestResource{}, &TestResourceList{})
}
