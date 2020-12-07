// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	"github.com/PingCAP-QE/Naglfar/pkg/ref"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in AcceptResources) DeepCopyInto(out *AcceptResources) {
	{
		in := &in
		*out = make(AcceptResources, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AcceptResources.
func (in AcceptResources) DeepCopy() AcceptResources {
	if in == nil {
		return nil
	}
	out := new(AcceptResources)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AvailableResource) DeepCopyInto(out *AvailableResource) {
	*out = *in
	if in.IdleCPUSet != nil {
		in, out := &in.IdleCPUSet, &out.IdleCPUSet
		*out = make([]int, len(*in))
		copy(*out, *in)
	}
	if in.Disks != nil {
		in, out := &in.Disks, &out.Disks
		*out = make(map[string]DiskResource, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AvailableResource.
func (in *AvailableResource) DeepCopy() *AvailableResource {
	if in == nil {
		return nil
	}
	out := new(AvailableResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTopologyRef) DeepCopyInto(out *ClusterTopologyRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTopologyRef.
func (in *ClusterTopologyRef) DeepCopy() *ClusterTopologyRef {
	if in == nil {
		return nil
	}
	out := new(ClusterTopologyRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DiskBinding) DeepCopyInto(out *DiskBinding) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DiskBinding.
func (in *DiskBinding) DeepCopy() *DiskBinding {
	if in == nil {
		return nil
	}
	out := new(DiskBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DiskResource) DeepCopyInto(out *DiskResource) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DiskResource.
func (in *DiskResource) DeepCopy() *DiskResource {
	if in == nil {
		return nil
	}
	out := new(DiskResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DiskSpec) DeepCopyInto(out *DiskSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DiskSpec.
func (in *DiskSpec) DeepCopy() *DiskSpec {
	if in == nil {
		return nil
	}
	out := new(DiskSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DockerContainerSpec) DeepCopyInto(out *DockerContainerSpec) {
	*out = *in
	out.ResourceRequest = in.ResourceRequest
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DockerContainerSpec.
func (in *DockerContainerSpec) DeepCopy() *DockerContainerSpec {
	if in == nil {
		return nil
	}
	out := new(DockerContainerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrafanaSpec) DeepCopyInto(out *GrafanaSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrafanaSpec.
func (in *GrafanaSpec) DeepCopy() *GrafanaSpec {
	if in == nil {
		return nil
	}
	out := new(GrafanaSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Machine) DeepCopyInto(out *Machine) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Machine.
func (in *Machine) DeepCopy() *Machine {
	if in == nil {
		return nil
	}
	out := new(Machine)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Machine) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineInfo) DeepCopyInto(out *MachineInfo) {
	*out = *in
	if in.StorageDevices != nil {
		in, out := &in.StorageDevices, &out.StorageDevices
		*out = make(map[string]StorageDevice, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineInfo.
func (in *MachineInfo) DeepCopy() *MachineInfo {
	if in == nil {
		return nil
	}
	out := new(MachineInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineList) DeepCopyInto(out *MachineList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Machine, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineList.
func (in *MachineList) DeepCopy() *MachineList {
	if in == nil {
		return nil
	}
	out := new(MachineList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MachineList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineRef) DeepCopyInto(out *MachineRef) {
	*out = *in
	out.Ref = in.Ref
	in.Binding.DeepCopyInto(&out.Binding)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineRef.
func (in *MachineRef) DeepCopy() *MachineRef {
	if in == nil {
		return nil
	}
	out := new(MachineRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineRequest) DeepCopyInto(out *MachineRequest) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineRequest.
func (in *MachineRequest) DeepCopy() *MachineRequest {
	if in == nil {
		return nil
	}
	out := new(MachineRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineSpec) DeepCopyInto(out *MachineSpec) {
	*out = *in
	if in.Reserve != nil {
		in, out := &in.Reserve, &out.Reserve
		*out = new(ReserveResources)
		**out = **in
	}
	if in.ExclusiveDisks != nil {
		in, out := &in.ExclusiveDisks, &out.ExclusiveDisks
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineSpec.
func (in *MachineSpec) DeepCopy() *MachineSpec {
	if in == nil {
		return nil
	}
	out := new(MachineSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineStatus) DeepCopyInto(out *MachineStatus) {
	*out = *in
	if in.Info != nil {
		in, out := &in.Info, &out.Info
		*out = new(MachineInfo)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineStatus.
func (in *MachineStatus) DeepCopy() *MachineStatus {
	if in == nil {
		return nil
	}
	out := new(MachineStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PDSpec) DeepCopyInto(out *PDSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PDSpec.
func (in *PDSpec) DeepCopy() *PDSpec {
	if in == nil {
		return nil
	}
	out := new(PDSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PrometheusSpec) DeepCopyInto(out *PrometheusSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PrometheusSpec.
func (in *PrometheusSpec) DeepCopy() *PrometheusSpec {
	if in == nil {
		return nil
	}
	out := new(PrometheusSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Relationship) DeepCopyInto(out *Relationship) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Relationship.
func (in *Relationship) DeepCopy() *Relationship {
	if in == nil {
		return nil
	}
	out := new(Relationship)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Relationship) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RelationshipList) DeepCopyInto(out *RelationshipList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Relationship, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RelationshipList.
func (in *RelationshipList) DeepCopy() *RelationshipList {
	if in == nil {
		return nil
	}
	out := new(RelationshipList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RelationshipList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RelationshipSpec) DeepCopyInto(out *RelationshipSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RelationshipSpec.
func (in *RelationshipSpec) DeepCopy() *RelationshipSpec {
	if in == nil {
		return nil
	}
	out := new(RelationshipSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RelationshipStatus) DeepCopyInto(out *RelationshipStatus) {
	*out = *in
	if in.MachineToResources != nil {
		in, out := &in.MachineToResources, &out.MachineToResources
		*out = make(map[string]ResourceRefList, len(*in))
		for key, val := range *in {
			var outVal []ResourceRef
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make(ResourceRefList, len(*in))
				for i := range *in {
					(*in)[i].DeepCopyInto(&(*out)[i])
				}
			}
			(*out)[key] = outVal
		}
	}
	if in.ResourceToMachine != nil {
		in, out := &in.ResourceToMachine, &out.ResourceToMachine
		*out = make(map[string]MachineRef, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.AcceptedRequests != nil {
		in, out := &in.AcceptedRequests, &out.AcceptedRequests
		*out = make(AcceptResources, len(*in))
		copy(*out, *in)
	}
	if in.MachineLocks != nil {
		in, out := &in.MachineLocks, &out.MachineLocks
		*out = make(map[string]ref.Ref, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RelationshipStatus.
func (in *RelationshipStatus) DeepCopy() *RelationshipStatus {
	if in == nil {
		return nil
	}
	out := new(RelationshipStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReserveResources) DeepCopyInto(out *ReserveResources) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReserveResources.
func (in *ReserveResources) DeepCopy() *ReserveResources {
	if in == nil {
		return nil
	}
	out := new(ReserveResources)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceBinding) DeepCopyInto(out *ResourceBinding) {
	*out = *in
	if in.CPUSet != nil {
		in, out := &in.CPUSet, &out.CPUSet
		*out = make([]int, len(*in))
		copy(*out, *in)
	}
	if in.Disks != nil {
		in, out := &in.Disks, &out.Disks
		*out = make(map[string]DiskBinding, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceBinding.
func (in *ResourceBinding) DeepCopy() *ResourceBinding {
	if in == nil {
		return nil
	}
	out := new(ResourceBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceContainerSpec) DeepCopyInto(out *ResourceContainerSpec) {
	*out = *in
	if in.CapAdd != nil {
		in, out := &in.CapAdd, &out.CapAdd
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Mounts != nil {
		in, out := &in.Mounts, &out.Mounts
		*out = make([]TestResourceMount, len(*in))
		copy(*out, *in)
	}
	if in.Binds != nil {
		in, out := &in.Binds, &out.Binds
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Command != nil {
		in, out := &in.Command, &out.Command
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Envs != nil {
		in, out := &in.Envs, &out.Envs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ExposedPorts != nil {
		in, out := &in.ExposedPorts, &out.ExposedPorts
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceContainerSpec.
func (in *ResourceContainerSpec) DeepCopy() *ResourceContainerSpec {
	if in == nil {
		return nil
	}
	out := new(ResourceContainerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceRef) DeepCopyInto(out *ResourceRef) {
	*out = *in
	out.Ref = in.Ref
	in.Binding.DeepCopyInto(&out.Binding)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceRef.
func (in *ResourceRef) DeepCopy() *ResourceRef {
	if in == nil {
		return nil
	}
	out := new(ResourceRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in ResourceRefList) DeepCopyInto(out *ResourceRefList) {
	{
		in := &in
		*out = make(ResourceRefList, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceRefList.
func (in ResourceRefList) DeepCopy() ResourceRefList {
	if in == nil {
		return nil
	}
	out := new(ResourceRefList)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceRequestItem) DeepCopyInto(out *ResourceRequestItem) {
	*out = *in
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceRequestItem.
func (in *ResourceRequestItem) DeepCopy() *ResourceRequestItem {
	if in == nil {
		return nil
	}
	out := new(ResourceRequestItem)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceRequestRef) DeepCopyInto(out *ResourceRequestRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceRequestRef.
func (in *ResourceRequestRef) DeepCopy() *ResourceRequestRef {
	if in == nil {
		return nil
	}
	out := new(ResourceRequestRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServerConfigs) DeepCopyInto(out *ServerConfigs) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServerConfigs.
func (in *ServerConfigs) DeepCopy() *ServerConfigs {
	if in == nil {
		return nil
	}
	out := new(ServerConfigs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StorageDevice) DeepCopyInto(out *StorageDevice) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StorageDevice.
func (in *StorageDevice) DeepCopy() *StorageDevice {
	if in == nil {
		return nil
	}
	out := new(StorageDevice)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestClusterTopology) DeepCopyInto(out *TestClusterTopology) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestClusterTopology.
func (in *TestClusterTopology) DeepCopy() *TestClusterTopology {
	if in == nil {
		return nil
	}
	out := new(TestClusterTopology)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TestClusterTopology) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestClusterTopologyList) DeepCopyInto(out *TestClusterTopologyList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TestClusterTopology, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestClusterTopologyList.
func (in *TestClusterTopologyList) DeepCopy() *TestClusterTopologyList {
	if in == nil {
		return nil
	}
	out := new(TestClusterTopologyList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TestClusterTopologyList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestClusterTopologySpec) DeepCopyInto(out *TestClusterTopologySpec) {
	*out = *in
	if in.TiDBCluster != nil {
		in, out := &in.TiDBCluster, &out.TiDBCluster
		*out = new(TiDBCluster)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestClusterTopologySpec.
func (in *TestClusterTopologySpec) DeepCopy() *TestClusterTopologySpec {
	if in == nil {
		return nil
	}
	out := new(TestClusterTopologySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestClusterTopologyStatus) DeepCopyInto(out *TestClusterTopologyStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestClusterTopologyStatus.
func (in *TestClusterTopologyStatus) DeepCopy() *TestClusterTopologyStatus {
	if in == nil {
		return nil
	}
	out := new(TestClusterTopologyStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestResource) DeepCopyInto(out *TestResource) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestResource.
func (in *TestResource) DeepCopy() *TestResource {
	if in == nil {
		return nil
	}
	out := new(TestResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TestResource) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestResourceList) DeepCopyInto(out *TestResourceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TestResource, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestResourceList.
func (in *TestResourceList) DeepCopy() *TestResourceList {
	if in == nil {
		return nil
	}
	out := new(TestResourceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TestResourceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestResourceMount) DeepCopyInto(out *TestResourceMount) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestResourceMount.
func (in *TestResourceMount) DeepCopy() *TestResourceMount {
	if in == nil {
		return nil
	}
	out := new(TestResourceMount)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestResourceRequest) DeepCopyInto(out *TestResourceRequest) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestResourceRequest.
func (in *TestResourceRequest) DeepCopy() *TestResourceRequest {
	if in == nil {
		return nil
	}
	out := new(TestResourceRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TestResourceRequest) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestResourceRequestList) DeepCopyInto(out *TestResourceRequestList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TestResourceRequest, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestResourceRequestList.
func (in *TestResourceRequestList) DeepCopy() *TestResourceRequestList {
	if in == nil {
		return nil
	}
	out := new(TestResourceRequestList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TestResourceRequestList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestResourceRequestSpec) DeepCopyInto(out *TestResourceRequestSpec) {
	*out = *in
	if in.Machines != nil {
		in, out := &in.Machines, &out.Machines
		*out = make([]*MachineRequest, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(MachineRequest)
				**out = **in
			}
		}
	}
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]*ResourceRequestItem, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(ResourceRequestItem)
				(*in).DeepCopyInto(*out)
			}
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestResourceRequestSpec.
func (in *TestResourceRequestSpec) DeepCopy() *TestResourceRequestSpec {
	if in == nil {
		return nil
	}
	out := new(TestResourceRequestSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestResourceRequestStatus) DeepCopyInto(out *TestResourceRequestStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestResourceRequestStatus.
func (in *TestResourceRequestStatus) DeepCopy() *TestResourceRequestStatus {
	if in == nil {
		return nil
	}
	out := new(TestResourceRequestStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestResourceSpec) DeepCopyInto(out *TestResourceSpec) {
	*out = *in
	if in.Disks != nil {
		in, out := &in.Disks, &out.Disks
		*out = make(map[string]DiskSpec, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestResourceSpec.
func (in *TestResourceSpec) DeepCopy() *TestResourceSpec {
	if in == nil {
		return nil
	}
	out := new(TestResourceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestResourceStatus) DeepCopyInto(out *TestResourceStatus) {
	*out = *in
	in.ResourceContainerSpec.DeepCopyInto(&out.ResourceContainerSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestResourceStatus.
func (in *TestResourceStatus) DeepCopy() *TestResourceStatus {
	if in == nil {
		return nil
	}
	out := new(TestResourceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestWorkflow) DeepCopyInto(out *TestWorkflow) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestWorkflow.
func (in *TestWorkflow) DeepCopy() *TestWorkflow {
	if in == nil {
		return nil
	}
	out := new(TestWorkflow)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TestWorkflow) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestWorkflowList) DeepCopyInto(out *TestWorkflowList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TestWorkflow, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestWorkflowList.
func (in *TestWorkflowList) DeepCopy() *TestWorkflowList {
	if in == nil {
		return nil
	}
	out := new(TestWorkflowList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TestWorkflowList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestWorkflowSpec) DeepCopyInto(out *TestWorkflowSpec) {
	*out = *in
	if in.Steps != nil {
		in, out := &in.Steps, &out.Steps
		*out = make([]WorkflowStepItem, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestWorkflowSpec.
func (in *TestWorkflowSpec) DeepCopy() *TestWorkflowSpec {
	if in == nil {
		return nil
	}
	out := new(TestWorkflowSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestWorkflowStatus) DeepCopyInto(out *TestWorkflowStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestWorkflowStatus.
func (in *TestWorkflowStatus) DeepCopy() *TestWorkflowStatus {
	if in == nil {
		return nil
	}
	out := new(TestWorkflowStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestWorkload) DeepCopyInto(out *TestWorkload) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestWorkload.
func (in *TestWorkload) DeepCopy() *TestWorkload {
	if in == nil {
		return nil
	}
	out := new(TestWorkload)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TestWorkload) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestWorkloadItemSpec) DeepCopyInto(out *TestWorkloadItemSpec) {
	*out = *in
	if in.DockerContainer != nil {
		in, out := &in.DockerContainer, &out.DockerContainer
		*out = new(DockerContainerSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestWorkloadItemSpec.
func (in *TestWorkloadItemSpec) DeepCopy() *TestWorkloadItemSpec {
	if in == nil {
		return nil
	}
	out := new(TestWorkloadItemSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestWorkloadList) DeepCopyInto(out *TestWorkloadList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TestWorkload, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestWorkloadList.
func (in *TestWorkloadList) DeepCopy() *TestWorkloadList {
	if in == nil {
		return nil
	}
	out := new(TestWorkloadList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TestWorkloadList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestWorkloadResult) DeepCopyInto(out *TestWorkloadResult) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestWorkloadResult.
func (in *TestWorkloadResult) DeepCopy() *TestWorkloadResult {
	if in == nil {
		return nil
	}
	out := new(TestWorkloadResult)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestWorkloadSpec) DeepCopyInto(out *TestWorkloadSpec) {
	*out = *in
	if in.ClusterTopologiesRefs != nil {
		in, out := &in.ClusterTopologiesRefs, &out.ClusterTopologiesRefs
		*out = make([]ClusterTopologyRef, len(*in))
		copy(*out, *in)
	}
	if in.Workloads != nil {
		in, out := &in.Workloads, &out.Workloads
		*out = make([]TestWorkloadItemSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.TeardownTestClusterTopology != nil {
		in, out := &in.TeardownTestClusterTopology, &out.TeardownTestClusterTopology
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestWorkloadSpec.
func (in *TestWorkloadSpec) DeepCopy() *TestWorkloadSpec {
	if in == nil {
		return nil
	}
	out := new(TestWorkloadSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TestWorkloadStatus) DeepCopyInto(out *TestWorkloadStatus) {
	*out = *in
	if in.Results != nil {
		in, out := &in.Results, &out.Results
		*out = make(map[string]TestWorkloadResult, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TestWorkloadStatus.
func (in *TestWorkloadStatus) DeepCopy() *TestWorkloadStatus {
	if in == nil {
		return nil
	}
	out := new(TestWorkloadStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiDBCluster) DeepCopyInto(out *TiDBCluster) {
	*out = *in
	out.Version = in.Version
	out.ServerConfigs = in.ServerConfigs
	if in.TiDB != nil {
		in, out := &in.TiDB, &out.TiDB
		*out = make([]TiDBSpec, len(*in))
		copy(*out, *in)
	}
	if in.TiKV != nil {
		in, out := &in.TiKV, &out.TiKV
		*out = make([]TiKVSpec, len(*in))
		copy(*out, *in)
	}
	if in.PD != nil {
		in, out := &in.PD, &out.PD
		*out = make([]PDSpec, len(*in))
		copy(*out, *in)
	}
	if in.Monitor != nil {
		in, out := &in.Monitor, &out.Monitor
		*out = make([]PrometheusSpec, len(*in))
		copy(*out, *in)
	}
	if in.Grafana != nil {
		in, out := &in.Grafana, &out.Grafana
		*out = make([]GrafanaSpec, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiDBCluster.
func (in *TiDBCluster) DeepCopy() *TiDBCluster {
	if in == nil {
		return nil
	}
	out := new(TiDBCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiDBClusterVersion) DeepCopyInto(out *TiDBClusterVersion) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiDBClusterVersion.
func (in *TiDBClusterVersion) DeepCopy() *TiDBClusterVersion {
	if in == nil {
		return nil
	}
	out := new(TiDBClusterVersion)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiDBSpec) DeepCopyInto(out *TiDBSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiDBSpec.
func (in *TiDBSpec) DeepCopy() *TiDBSpec {
	if in == nil {
		return nil
	}
	out := new(TiDBSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiKVSpec) DeepCopyInto(out *TiKVSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiKVSpec.
func (in *TiKVSpec) DeepCopy() *TiKVSpec {
	if in == nil {
		return nil
	}
	out := new(TiKVSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkflowStepItem) DeepCopyInto(out *WorkflowStepItem) {
	*out = *in
	if in.TestResourceRequest != nil {
		in, out := &in.TestResourceRequest, &out.TestResourceRequest
		*out = new(TestResourceSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.TestClusterTopology != nil {
		in, out := &in.TestClusterTopology, &out.TestClusterTopology
		*out = new(TestClusterTopology)
		(*in).DeepCopyInto(*out)
	}
	if in.TestWorkload != nil {
		in, out := &in.TestWorkload, &out.TestWorkload
		*out = new(TestWorkload)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkflowStepItem.
func (in *WorkflowStepItem) DeepCopy() *WorkflowStepItem {
	if in == nil {
		return nil
	}
	out := new(WorkflowStepItem)
	in.DeepCopyInto(out)
	return out
}
