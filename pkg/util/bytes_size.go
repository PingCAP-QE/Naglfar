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

package util

import (
	"fmt"

	"github.com/docker/go-units"
)

type BytesSize string

func (b BytesSize) ToSize() (int64, error) {
	return units.RAMInBytes(string(b))
}

func (b BytesSize) Unwrap() int64 {
	size, err := b.ToSize()
	if err != nil {
		panic(fmt.Sprintf("BytesSize(%s) is invalid", b))
	}
	return size
}

func (b BytesSize) Zero() BytesSize {
	return Size(0)
}

func (b BytesSize) Sub(other BytesSize) BytesSize {
	return Size(float64(b.Unwrap() - other.Unwrap()))
}

func Size(size float64) BytesSize {
	return BytesSize(units.BytesSize(size))
}
