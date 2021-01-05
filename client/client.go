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
	"fmt"
	"github.com/PingCAP-QE/Naglfar/pkg/tiup/cluster"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/PingCAP-QE/Naglfar/controllers"
	"github.com/PingCAP-QE/Naglfar/pkg/kubeutil"
	tiupSpec "github.com/pingcap/tiup/pkg/cluster/spec"
)

type Client struct {
	client.Client
}

var (
	namespace        string
	requestName      string
	testWorkloadName string
	workloadItemName string

	// k8s
	scheme *runtime.Scheme
)

func (c *Client) GetTiDBClusterTopology(ctx context.Context, clusterName string) (*tiupSpec.Specification, error) {
	originCluster := os.Getenv(clusterName)
	if len(originCluster) == 0 {
		originCluster = clusterName
	}
	topology, err := c.getClusterTopology(ctx, originCluster)
	if err != nil {
		return nil, err
	}
	if topology.Spec.TiDBCluster == nil {
		panic(fmt.Sprintf("incorrect cluster topology used, please check the cluster: %s, %s", clusterName, originCluster))
	}
	resources, err := c.getTestResources(ctx)
	if err != nil {
		return nil, err
	}
	spec, _, err := cluster.BuildSpecification(&topology.Spec, resources, false)
	return &spec, err
}

func (c *Client) getTestResources(ctx context.Context) ([]*naglfarv1.TestResource, error) {
	var resourceList naglfarv1.TestResourceList
	var resources []*naglfarv1.TestResource
	err := kubeutil.Retry(3, time.Second, func() error {
		if err := c.List(ctx, &resourceList, client.InNamespace(namespace)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for idx := range resourceList.Items {
		resources = append(resources, &resourceList.Items[idx])
	}
	return resources, nil
}

func (c *Client) getClusterTopology(ctx context.Context, clusterName string) (*naglfarv1.TestClusterTopology, error) {
	var ct naglfarv1.TestClusterTopology
	ct.Namespace = namespace
	ct.Name = clusterName
	key, err := client.ObjectKeyFromObject(&ct)
	if err != nil {
		return nil, err
	}
	err = kubeutil.Retry(3, time.Second, func() error {
		if err := c.Get(ctx, key, &ct); err != nil {
			return err
		}
		return nil
	})
	return &ct, err
}

func (c *Client) getTestWorkload(ctx context.Context) (*naglfarv1.TestWorkload, error) {
	var workload naglfarv1.TestWorkload
	workload.Namespace = namespace
	workload.Name = testWorkloadName
	key, err := client.ObjectKeyFromObject(&workload)
	if err != nil {
		return nil, err
	}
	err = kubeutil.Retry(3, time.Second, func() error {
		if err := c.Get(ctx, key, &workload); err != nil {
			return err
		}
		return nil
	})
	return &workload, err
}

// NewClient creates the Naglfar client
func NewClient(kubeconfig string) (*Client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	cli, err := client.New(config, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, nil
	}
	return &Client{
		cli,
	}, nil
}

func init() {
	namespace = os.Getenv(controllers.NaglfarClusterNs)
	requestName = os.Getenv(controllers.NaglfarTestResourceRequestName)
	testWorkloadName = os.Getenv(controllers.NaglfarTestWorkloadName)
	workloadItemName = os.Getenv(controllers.NaglfarTestWorkloadItem)

	scheme = runtime.NewScheme()
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	utilruntime.Must(kubescheme.AddToScheme(scheme))
	utilruntime.Must(naglfarv1.AddToScheme(scheme))
}
