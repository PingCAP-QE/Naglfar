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

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourcePending       ResourceState = "pending"
	ResourceFail                        = "fail"
	ResourceUninitialized               = "uninitialized"
	ResourceReady                       = "ready"
	ResourceFinish                      = "finish"
	ResourceDestroy                     = "destroy"
)

const cleanerImage = "alpine:latest"

// +kubebuilder:validation:Enum=pending;fail;uninitialized;ready;finish;destroy
type ResourceState string

func (r ResourceState) IsRequired() bool {
	return r != ResourcePending && r != ResourceFail
}
func (r ResourceState) IsInstalled() bool {
	return r == ResourceReady || r == ResourceFinish
}

type DiskSpec struct {
	// default /mnt/<name>
	// +optional
	MountPath string `json:"mountPath"`

	// +optional
	Kind DiskKind `json:"kind"`

	// +optional
	Size BytesSize `json:"size"`
}

type DiskStatus struct {
	Kind       DiskKind  `json:"kind"`
	Size       BytesSize `json:"size"`
	Device     string    `json:"device"`
	OriginPath string    `json:"originPath"`
	MountPath  string    `json:"mountPath"`
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

// https://github.com/moby/moby/blob/master/api/types/mount/mount.go#L23

type TestResourceMount struct {
	Type     mount.Type `json:"type,omitempty"`
	Source   string     `json:"source,omitempty"`
	Target   string     `json:"target,omitempty"`
	ReadOnly bool       `json:"readOnly,omitempty"`
}

// TestResourceStatus defines the observed state of TestResource
type TestResourceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// default pending
	// +optional
	State ResourceState `json:"state"`

	// +optional
	HostMachine *corev1.ObjectReference `json:"hostMachine,omitempty"`

	// Container configuration section

	// default false
	Privilege bool `json:"privilege,omitempty"`

	// +optional
	DiskStat map[string]DiskStatus `json:"diskStat,omitempty"`

	// +optional
	Mounts []TestResourceMount `json:"mount,omitempty"`

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

func (r *TestResource) ContainerConfig() (*container.Config, *container.HostConfig) {
	mounts := make([]mount.Mount, 0)
	for _, disk := range r.Status.DiskStat {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: disk.OriginPath,
			Target: disk.MountPath,
		})
	}

	// bind mounts
	for _, m := range r.Status.Mounts {
		mounts = append(mounts, mount.Mount{
			Type:     m.Type,
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}

	config := &container.Config{
		Image: r.Status.Image,
	}

	hostConfig := &container.HostConfig{
		Mounts: mounts,
		Resources: container.Resources{
			Memory:   r.Spec.Memory.Unwrap(),
			CPUQuota: int64(r.Spec.CPUPercent) * 1000,
		},
		// set privilege
		Privileged: r.Status.Privilege,
	}

	if len(r.Status.Commands) != 0 {
		script := strings.Join(r.Status.Commands, ";")
		config.Cmd = []string{"bash", "-c", script}
	}

	return config, hostConfig
}

func (r *TestResource) ContainerCleanerConfig() (*container.Config, *container.HostConfig) {
	mounts := make([]mount.Mount, 0)
	for _, disk := range r.Status.DiskStat {
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
		PortBindings: nat.PortMap{
			"22": {},
		},
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
