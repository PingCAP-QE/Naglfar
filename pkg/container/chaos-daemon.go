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
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
)

const ChaosDaemon = "chaos-daemon"
const ChaosDaemonPort = "31767/tcp"
const ChaosDaemonImage = "docker.io/pingcap/chaos-daemon"

const dockerSocket = "/var/run/docker.sock"
const sysDir = "/sys"

func ChaosDaemonCfg() (*container.Config, *container.HostConfig) {
	mounts := make([]mount.Mount, 0)

	// bind mounts
	for _, m := range []string{dockerSocket, sysDir} {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: m,
			Target: m,
		})
	}

	config := &container.Config{
		Image: ChaosDaemonImage,
		Cmd: strslice.StrSlice{
			"/usr/local/bin/chaos-daemon",
			"--runtime", "docker",
		},
		ExposedPorts: nat.PortSet{ChaosDaemonPort: struct{}{}},
	}

	hostConfig := &container.HostConfig{
		Mounts:          mounts,
		PublishAllPorts: true,
		CapAdd:          strslice.StrSlice{"SYS_PTRACE", "NET_ADMIN", "MKNOD", "SYS_CHROOT", "SYS_ADMIN", "KILL", "IPC_LOCK"},
	}

	return config, hostConfig
}
