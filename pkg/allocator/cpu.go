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
	"fmt"
	"sort"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/PingCAP-QE/Naglfar/pkg/ref"
)

type CpuAllocator struct{}

func (a CpuAllocator) Filter(ctx *FilterContext) (err error) {
	coresNeeds := int(ctx.Target.Spec.Cores)

	var machines []*naglfarv1.Machine
	for _, machine := range ctx.Machines {
		machineKey := ref.CreateRef(&machine.ObjectMeta).Key()
		resources := ctx.Relationship.Status.MachineToResources[machineKey]
		cpuSet := restCpuSet(machine, resources)

		if len(cpuSet) >= coresNeeds {
			machines = append(machines, machine)
		}
	}

	ctx.Machines = machines
	return
}

func (a CpuAllocator) Bind(ctx *BindContext) (err error) {
	machineKey := ref.CreateRef(&ctx.Machine.ObjectMeta).Key()
	resources := ctx.Relationship.Status.MachineToResources[machineKey]
	ctx.Binding.CPUSet = pickCpuSet(restCpuSet(ctx.Machine, resources), ctx.Target.Spec.Cores)
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

func pickCpuSet(cpuSet []int, i int32) (set []int) {
	if len(cpuSet) < int(i) {
		return
	}
	for index := 0; index < int(i); index++ {
		set = append(set, cpuSet[index])
	}
	return
}
