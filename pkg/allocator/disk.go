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

var DiskAllocate AllocateFunc = func(ctx AllocContext) (err error) {
	diskNeeds := ctx.Target.Spec.Disks

	machines := make(BindingTable)
	for machine, binding := range ctx.Machines {
		newBinding := binding.DeepCopy()
		machineKey := ref.CreateRef(&machine.ObjectMeta).Key()
		resources := ctx.Relationship.Status.MachineToResources[machineKey]
		disks := restDisks(machine, resources)

		for name, disk := range diskNeeds {
			for device, diskResource := range disks {
				if disk.Kind == diskResource.Kind &&
					disk.Size.Unwrap() <= diskResource.Size.Unwrap() {

					delete(disks, name)
					newBinding.Disks[name] = naglfarv1.DiskBinding{
						Kind:       disk.Kind,
						Size:       diskResource.Size,
						Device:     device,
						OriginPath: diskResource.MountPath,
						MountPath:  disk.MountPath,
					}

					break
				}
			}
		}

		if len(newBinding.Disks) == len(diskNeeds) {
			machines[machine] = newBinding
		}
	}

	ctx.Machines = machines
	return
}

type DiskResource struct {
	Size      util.BytesSize
	Kind      naglfarv1.DiskKind
	MountPath string
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
