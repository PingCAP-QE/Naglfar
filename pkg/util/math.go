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

package util

import (
	"math/rand"
	"time"
)

func Rand(n int) int {
	rand.Seed(int64(time.Now().UnixNano()))
	return rand.Intn(n)
}

func RandOne(strSlice []string) string {
	return strSlice[Rand(len(strSlice))]
}

func MinDuration(duractions ...time.Duration) time.Duration {
	min := time.Duration(int(^uint(0) >> 1))

	for _, duration := range duractions {
		if duration < min {
			min = duration
		}
	}

	return min
}
