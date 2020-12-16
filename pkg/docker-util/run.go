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

package dockerutil

import (
	"bytes"
	"fmt"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	docker "github.com/docker/docker/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
)

type RunOptions struct {
	Policy           naglfarv1.PullImagePolicy
	Config           *container.Config
	HostConfig       *container.HostConfig
	NetworkingConfig *network.NetworkingConfig
}

func (client *Client) Run(name, image string, options RunOptions) (stat dockerTypes.ContainerJSON, err error) {
	if options.Policy == "" {
		options.Policy = naglfarv1.PullPolicyIfNotPresent
	}

	if err = client.PullImageByPolicy(image, options.Policy); err != nil {
		return
	}

	if stat, err = client.ContainerInspect(client.Ctx, name); err != nil {
		if !docker.IsErrNotFound(err) {
			return
		}

		_, err = client.ContainerCreate(client.Ctx, options.Config, options.HostConfig, options.NetworkingConfig, name)
		if err != nil {
			return
		}

		stat, err = client.ContainerInspect(client.Ctx, name)
		if err != nil {
			return
		}
	}

	if TimeIsZero(stat.State.StartedAt) {
		err = client.ContainerStart(client.Ctx, name, dockerTypes.ContainerStartOptions{})
	}

	return
}

func (client *Client) JustExec(name, image string, options RunOptions) (stdout bytes.Buffer, err error) {
	stat, err := client.Run(name, image, options)

	if err != nil {
		return
	}

	exitCode, err := client.ContainerWait(client.Ctx, stat.ID)
	if err != nil {
		return
	}

	if exitCode != 0 {
		err = fmt.Errorf("container %s exit with %d", stat.ID, exitCode)
		return
	}

	stdout, _, err = client.Logs(stat.Name, dockerTypes.ContainerLogsOptions{
		ShowStdout: true,
	})

	if err != nil {
		return
	}

	err = client.ContainerRemove(client.Ctx, stat.ID, dockerTypes.ContainerRemoveOptions{
		Force: true,
	})

	return
}
