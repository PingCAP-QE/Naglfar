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

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stdcopy"
)

func (client *Client) Exec(name string, config dockerTypes.ExecConfig) (stdout, stderr bytes.Buffer, err error) {
	exec, err := client.ContainerExecCreate(client.Ctx, name, config)
	if err != nil {
		return
	}

	resp, err := client.ContainerExecAttach(client.Ctx, exec.ID, config)
	if err != nil {
		return
	}
	defer resp.Close()

	_, err = stdcopy.StdCopy(&stdout, &stderr, resp.Reader)
	return
}
