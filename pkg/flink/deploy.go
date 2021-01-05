package flink

import (
	"context"
	"fmt"
	"time"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/creasty/defaults"
	"github.com/go-logr/logr"
)

const (
	// Our controller node and worker nodes share the same insecure_key path
	insecureKeyPath = "/root/insecure_key"
	BaseImageName   = "docker.io/library/flink:"
	sshTimeout      = 10 * time.Minute
)

type ErrClusterDuplicated struct {
	clusterName string
}

type ErrClusterNotExist struct {
	clusterName string
}

func (e ErrClusterDuplicated) Error() string {
	return fmt.Sprintf("cluster name %s is duplicated", e.clusterName)
}

func (e ErrClusterNotExist) Error() string {
	return fmt.Sprintf("cluster name %s is not exist", e.clusterName)
}

// IgnoreClusterDuplicated returns nil on ClusterDuplicated errors
// All other values that are not NotFound errors or nil are returned unmodified.
func IgnoreClusterDuplicated(err error) error {
	if _, ok := err.(ErrClusterDuplicated); ok {
		return nil
	}
	return err
}

// IgnoreClusterNotExist returns nil on IgnoreClusterNotExist errors
// All other values that are not NotFound errors or nil are returned unmodified.
func IgnoreClusterNotExist(err error) error {
	if _, ok := err.(ErrClusterNotExist); ok {
		return nil
	}
	return err
}

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
			return spec, nil, fmt.Errorf("tidb node not found: `%s`", item.Host)
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

type ClusterManager struct {
	log     logr.Logger
	spec    *naglfarv1.FlinkCluster
	control *naglfarv1.TestResourceStatus
}

func MakeClusterManager(log logr.Logger, ctf *naglfarv1.TestClusterTopologySpec, trs []*naglfarv1.TestResource, clusterIPMaps ...map[string]string) (*ClusterManager, error) {
	var specification naglfarv1.FlinkCluster
	var control *naglfarv1.TestResourceStatus
	var err error
	specification, control, err = BuildSpecification(ctf, trs, false)
	if err != nil {
		return nil, err
	}
	return &ClusterManager{
		log:     log,
		spec:    &specification,
		control: control.DeepCopy(),
	}, nil
}

func (c *ClusterManager) InstallCluster(ctx context.Context, log logr.Logger) error {
	log.Info("flink cluster is installed")
	return nil
}
