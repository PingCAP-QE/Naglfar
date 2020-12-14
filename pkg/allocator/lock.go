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

import "github.com/PingCAP-QE/Naglfar/pkg/ref"

var LockAllocate AllocateFunc = func(ctx AllocContext) (err error) {
	machines := make(BindingTable)
	requestRef := ref.CreateRef(&ctx.Request.ObjectMeta)

	for machine, binding := range ctx.Machines {
		machineRef := ref.CreateRef(&machine.ObjectMeta)

		if locker, ok := ctx.Relationship.Status.MachineLocks[machineRef.Key()]; !ok || locker == requestRef {
			machines[machine] = binding
		}
	}

	ctx.Machines = machines
	return nil
}
