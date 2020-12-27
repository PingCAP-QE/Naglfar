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

package util

import (
	"fmt"
	"time"
)

const layout = time.RFC3339

type Time string

func (r Time) Parse() (time.Time, error) {
	return time.Parse(layout, string(r))
}

func (r Time) Unwrap() time.Time {
	t, err := r.Parse()
	if err != nil {
		panic(fmt.Sprintf("Time(%s) is invalid", r))
	}
	return t
}

func Format(t time.Time) Time {
	return Time(t.Format(layout))
}
