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

package container

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"

	"github.com/PingCAP-QE/Naglfar/pkg/script"
)

const statScript = "os-stat.sh"

const MachineStatImage = "docker.io/alexeiled/nsenter"
const MachineStat = "naglfar-machine-stat"

func MachineStatCfg() (*container.Config, *container.HostConfig) {
	osStatScript, err := script.ScriptBox.FindString(statScript)

	if err != nil {
		panic(fmt.Sprintf("fail to find %s: %s", statScript, err.Error()))
	}

	config := &container.Config{
		Image: MachineStatImage,
		Cmd:   strslice.StrSlice{"-m", "--target", "1", "--", "sh", "-c", osStatScript},
	}

	hostConfig := &container.HostConfig{
		Privileged: true,
		PidMode:    "host",
	}

	return config, hostConfig
}
