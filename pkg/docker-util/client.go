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
	"context"

	docker "github.com/docker/docker/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
)

type Client struct {
	*docker.Client
	Ctx context.Context
}

func MakeClient(ctx context.Context, machine *naglfarv1.Machine) (*Client, error) {
	rawClient, err := machine.DockerClient()
	if err != nil {
		return nil, err
	}

	client := &Client{
		Client: rawClient,
		Ctx:    ctx,
	}

	return client, nil
}
