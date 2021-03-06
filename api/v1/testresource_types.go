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
	"fmt"
	"path"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/PingCAP-QE/Naglfar/pkg/ref"
	"github.com/PingCAP-QE/Naglfar/pkg/util"
)

const (
	ResourcePending       = ResourceState("pending")
	ResourceUninitialized = ResourceState("uninitialized")
	ResourceReady         = ResourceState("ready")
	ResourceFinish        = ResourceState("finish")
	ResourceDestroy       = ResourceState("destroy")

	ResourcePhasePending   = ResourcePhase("pending")
	ResourcePhaseRunning   = ResourcePhase("running")
	ResourcePhaseFailed    = ResourcePhase("failed")
	ResourcePhaseSucceeded = ResourcePhase("succeeded")

	PullPolicyAlways       = PullImagePolicy("Always")
	PullPolicyIfNotPresent = PullImagePolicy("IfNotPresent")
)

// TODO: make it configurable
const cleanerImage = "hub.pingcap.net/mahjonp/alpine:latest"

const SSHPort = "22/tcp"

// ResourceState is used on controller reconcile
// +kubebuilder:validation:Enum=pending;fail;uninitialized;ready;finish;destroy
type ResourceState string

// ResourcePhase tells us the running state of resource workload, eg. whether the resource is pending, running, exit with error or exit normally
// +kubebuilder:validation:Enum=pending;running;failed;succeeded
type ResourcePhase string

func (r ResourceState) IsRequired() bool {
	return r != "" && r != ResourcePending
}

// CouldUninstall check whether a resource of current state can be re-initialized
func (r ResourceState) CouldUninstall() bool {
	switch r {
	case ResourceUninitialized, ResourceReady, ResourceFinish:
		return true
	default:
		return false
	}
}

type DiskSpec struct {
	// default /mnt/<name>
	// +optional
	MountPath string `json:"mountPath"`

	// +optional
	Kind DiskKind `json:"kind"`

	// +optional
	Size util.BytesSize `json:"size"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TestResourceSpec defines the desired state of TestResource
type TestResourceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Memory util.BytesSize `json:"memory"`

	Cores int32 `json:"cores"`

	// +optional
	MachineSelector string `json:"machineSelector,omitempty"`

	// +optional
	Disks map[string]DiskSpec `json:"disks,omitempty"`

	// If sets, it means the static machine is required
	// +optional
	Machine string `json:"machine,omitempty"`
}

// https://github.com/moby/moby/blob/master/api/types/mount/mount.go#L23

type TestResourceMount struct {
	Type     mount.Type `json:"type,omitempty"`
	Source   string     `json:"source,omitempty"`
	Target   string     `json:"target,omitempty"`
	ReadOnly bool       `json:"readOnly,omitempty"`
}

// +kubebuilder:validation:Enum=Always;IfNotPresent
type PullImagePolicy string

type ResourceContainerSpec struct {
	// +optional
	// default false
	Privilege bool `json:"privilege,omitempty"`

	// List of kernel capabilities to add to the container
	// +optional
	CapAdd []string `json:"capAdd,omitempty"`

	// Mounts specs used by the container
	// +optional
	Mounts []TestResourceMount `json:"mount,omitempty"`

	// List of volume bindings for this container
	// +optional
	Binds []string `json:"binds,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	// default IfNotPresent
	ImagePullPolicy PullImagePolicy `json:"imagePullPolicy,omitempty"`

	// +optional
	Command []string `json:"command,omitempty"`

	// +optional
	Envs []string `json:"envs,omitempty"`

	// +optional
	ExposedPorts []string `json:"exposedPorts,omitempty"`

	// +optional
	PortBindings string `json:"portBindings,omitempty"`

	// ClusterIP is the ip address of the container in the overlay(or calico) network
	// +optional
	ClusterIP string `json:"clusterIP"`
}

// TestResourceStatus defines the observed state of TestResource
type TestResourceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// default pending
	// +optional
	State ResourceState `json:"state,omitempty"`

	// +optional
	Phase ResourcePhase `json:"phase,omitempty"`

	// means the container's exit code
	// +optional
	ExitCode int `json:"exitCode,omitempty"`

	ResourceContainerSpec `json:",inline"`

	// HostIP is the ip address of the host machine
	// +optional
	HostIP string `json:"hostIP"`

	// +optional
	HostName string `json:"hostname"`

	// +optional
	Password string `json:"password"`

	// +optional
	SSHPort int `json:"sshPort"`

	// +optional
	PortBindings string `json:"portBindings,omitempty"`

	// Records which CR uses this test resource
	// For example, if workloada uses this resource, if workloadb wants to uses this resource too,
	// it needs to wait workloada clean the `claimRef` field of this resource
	ClaimRef *ref.Ref `json:"claimRef,omitempty"`
}

// +kubebuilder:object:root=true

// TestResource is the Schema for the testresources API
// +kubebuilder:resource:shortName="tr"
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="the state of resource"
// +kubebuilder:printcolumn:name="HostIP",type="string",JSONPath=".status.hostIP",description="the host ip of resource"
// +kubebuilder:printcolumn:name="SSHPort",type="integer",JSONPath=".status.sshPort",description="the ssh port of resource"
// +kubebuilder:printcolumn:name="ClusterIP",type="string",JSONPath=".status.clusterIP",description="the cluster ip of resource"
// +kubebuilder:printcolumn:name="PortBindings",type="string",JSONPath=".status.portBindings",description="the port bindings of resource"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
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

	// bind mounts
	for _, m := range r.Status.Mounts {
		mounts = append(mounts, mount.Mount{
			Type:     m.Type,
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}

	exposedPorts := make(nat.PortSet)
	for _, item := range r.Status.ExposedPorts {
		exposedPorts[nat.Port(item)] = struct{}{}
	}
	var entrypoint []string
	var cmd []string
	if len(r.Status.Command) != 0 {
		entrypoint = r.Status.Command[:1]
		cmd = r.Status.Command[1:]
	}
	config := &container.Config{
		Hostname:     r.Status.HostName,
		Image:        r.Status.Image,
		Entrypoint:   entrypoint,
		Cmd:          cmd,
		Env:          r.Status.Envs,
		ExposedPorts: exposedPorts,
	}

	hostConfig := &container.HostConfig{
		Binds:           r.Status.Binds,
		Mounts:          mounts,
		PublishAllPorts: true,
		Resources: container.Resources{
			Memory:     binding.Memory.Unwrap(),
			CpusetCpus: cpuSetStr(binding.CPUSet),
		},
		CapAdd: r.Status.CapAdd,
		// set privilege
		Privileged: r.Status.Privilege,
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
		var subCommand []string
		for _, mnt := range mounts {
			subCommand = append(subCommand, fmt.Sprintf("rm -rf %s", path.Join(mnt.Target, "*")))
		}
		config.Cmd = []string{"sh", "-c", strings.Join(subCommand, ";")}
	}

	return config, hostConfig
}

func init() {
	SchemeBuilder.Register(&TestResource{}, &TestResourceList{})
}
