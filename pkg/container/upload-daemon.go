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
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
)

const UploadDaemon = "upload-daemon"
const UploadDaemonPort = "6666/tcp"
const UploadDaemonImage = "docker.io/cadmusjiang/upload-daemon:latest"
const UploadPath = "/var/naglfar/lib"
const UploadLabel = "upload"
const UploadExternalPort = 31234

func UploadDaemonCfg(upload string) (*container.Config, *container.HostConfig) {
	mounts := make([]mount.Mount, 0)

	config := &container.Config{
		Image: UploadDaemonImage,
		Labels: map[string]string{
			UploadLabel: upload,
		},
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{UploadDaemonPort: []nat.PortBinding{nat.PortBinding{
			HostIP:   "0.0.0.0",
			HostPort: strconv.Itoa(UploadExternalPort),
		}}},
		Binds:           []string{UploadPath + ":" + UploadPath},
		Mounts:          mounts,
		PublishAllPorts: true,
		RestartPolicy: container.RestartPolicy{
			Name: "always",
		},
	}

	return config, hostConfig
}
