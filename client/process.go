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

package client

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/PingCAP-QE/Naglfar/pkg/kubeutil"
)

func (c *Client) CreateProcChaos(ctx context.Context, chaosName string, tasks ...*naglfarv1.ProcChaosTask) error {
	procChaos := &naglfarv1.ProcChaos{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      chaosName,
		},
		Spec: naglfarv1.ProcChaosSpec{
			Request: requestName,
			Tasks:   tasks,
		},
	}

	return kubeutil.Retry(3, time.Second, func() error {
		return c.Create(ctx, procChaos)
	})
}

func (c *Client) DeleteProcChaos(ctx context.Context, chaosName string) error {
	procChaos := &naglfarv1.ProcChaos{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      chaosName,
		},
	}

	return kubeutil.Retry(3, time.Second, func() error {
		return c.Delete(ctx, procChaos)
	})
}
