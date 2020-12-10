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

import naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"

type BindingTable map[*naglfarv1.Machine][]*naglfarv1.ResourceBinding

type AllocContext struct {
	Machines     BindingTable
	Request      *naglfarv1.TestResourceRequest
	Relationship *naglfarv1.Relationship
	Target       *naglfarv1.TestResource
}

type Allocator interface {
	Alloc(*AllocContext) error
}

type Combination []Allocator

func Combine(allocators ...Allocator) Combination {
	return allocators
}

func (c Combination) Alloc(ctx *AllocContext) (err error) {
	for _, alloc := range c {
		err = alloc.Alloc(ctx)
		if err != nil {
			break
		}
	}
	return
}
