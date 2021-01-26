package flink

import (
	"fmt"

	"github.com/creasty/defaults"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
)

const (
	BaseImageName = "docker.io/library/flink:"
)

// If dryRun is true, host uses the resource's name
func BuildSpecification(ctf *naglfarv1.TestClusterTopologySpec, trs []*naglfarv1.TestResource, dryRun bool) (
	spec naglfarv1.FlinkCluster, control *naglfarv1.TestResourceStatus, err error) {
	hostName := func(resourceName, clusterIP string) string {
		if dryRun {
			return resourceName
		}
		return clusterIP
	}

	resourceMaps := make(map[string]*naglfarv1.TestResourceStatus)
	for idx, resource := range trs {
		resourceMaps[resource.Name] = &trs[idx].Status
	}

	jobManager, exist := resourceMaps[ctf.FlinkCluster.JobManager.Host]
	if !exist {
		return spec, nil, fmt.Errorf("jobManager node not found: `%v`", ctf.FlinkCluster.JobManager)
	}

	spec.JobManager = naglfarv1.JobManagerSpec{
		Host:    hostName(ctf.FlinkCluster.JobManager.Host, jobManager.ClusterIP),
		WebPort: ctf.FlinkCluster.JobManager.WebPort,
		Config:  ctf.FlinkCluster.JobManager.Config,
	}

	for idx := range ctf.FlinkCluster.TaskManager {
		item := &ctf.FlinkCluster.TaskManager[idx]
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("taskManager node not found: `%s`", item.Host)
		}
		spec.TaskManager = append(spec.TaskManager, naglfarv1.TaskManagerSpec{
			Host:   hostName(item.Host, node.ClusterIP),
			Config: item.Config,
		})
	}

	// set default values from tag
	defaults.Set(&spec)
	return
}
