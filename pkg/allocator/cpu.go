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
	"sort"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/PingCAP-QE/Naglfar/pkg/ref"
)

var CpuAllocate AllocateFunc = func(ctx AllocContext) (err error) {
	coresNeeds := int(ctx.Target.Spec.Cores)

	machines := make(BindingTable)
	for machine, binding := range ctx.Machines {
		machineKey := ref.CreateRef(&machine.ObjectMeta).Key()
		resources := ctx.Relationship.Status.MachineToResources[machineKey]
		cpuSet := restCpuSet(machine, resources)

		if len(cpuSet) >= coresNeeds {
			newBinding := binding.DeepCopy()
			newBinding.CPUSet = pickCpuSet(cpuSet, int32(coresNeeds))
			machines[machine] = newBinding
		}
	}

	ctx.Machines = machines
	return
}

func restCpuSet(machine *naglfarv1.Machine, resources naglfarv1.ResourceRefList) []int {
	cpuSet := deleteCPUSet(makeCPUSet(machine.Status.Info.Threads), makeCPUSet(machine.Spec.Reserve.Cores))
	for _, resourceRef := range resources {
		cpuSet = deleteCPUSet(cpuSet, resourceRef.Binding.CPUSet)
	}
	return cpuSet
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

func pickCpuSet(cpuSet []int, i int32) (set []int) {
	if len(cpuSet) < int(i) {
		return
	}
	for index := 0; index < int(i); index++ {
		set = append(set, cpuSet[index])
	}
	return
}
