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

package allocator

import (
	"path"
	"strings"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/PingCAP-QE/Naglfar/pkg/ref"
	"github.com/PingCAP-QE/Naglfar/pkg/util"
)

type DiskAllocator struct{}

type DiskResource struct {
	Size      util.BytesSize     `json:"size"`
	Kind      naglfarv1.DiskKind `json:"kind"`
	MountPath string             `json:"mountPath"`
}

func (a DiskAllocator) Filter(ctx *FilterContext) (err error) {
	coresNeeds := int(ctx.Target.Spec.Disks)

	var machines []*naglfarv1.Machine
	for _, machine := range ctx.Machines {
		machineKey := ref.CreateRef(&machine.ObjectMeta).Key()
		resources := ctx.Relationship.Status.MachineToResources[machineKey]
		disks := restDisks(machine, resources)

		if len(cpuSet) >= coresNeeds {
			machines = append(machines, machine)
		}
	}

	ctx.Machines = machines
	return
}

func (a DiskAllocator) Bind(ctx *BindContext) (err error) {
	machineKey := ref.CreateRef(&ctx.Machine.ObjectMeta).Key()
	resources := ctx.Relationship.Status.MachineToResources[machineKey]
	ctx.Binding.CPUSet = pickCpuSet(restCpuSet(ctx.Machine, resources), ctx.Target.Spec.Cores)
	return
}

func restDisks(machine *naglfarv1.Machine, resources naglfarv1.ResourceRefList) map[string]*DiskResource {
	disks := make(map[string]*DiskResource)
	exclusiveSet := make(map[string]bool)

	for _, device := range machine.Spec.ExclusiveDisks {
		exclusiveSet[device] = true
	}

	for device, disk := range machine.Status.Info.StorageDevices {
		if _, ok := exclusiveSet[device]; ok {
			diskResource := &DiskResource{
				Size:      disk.Total.Sub(disk.Used),
				MountPath: disk.MountPoint,
			}

			if strings.HasPrefix(path.Base(device), "nvme") {
				diskResource.Kind = naglfarv1.NVMEKind
			} else {
				diskResource.Kind = naglfarv1.OtherKind
			}

			disks[device] = diskResource
		}
	}

	for _, resourceRef := range resources {
		for device := range resourceRef.Binding.Disks {
			delete(disks, device)
		}
	}

	return disks
}
