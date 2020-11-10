package tiup

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"github.com/creasty/defaults"
	"github.com/go-logr/logr"
	"github.com/pingcap/tiup/pkg/meta"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	sshUtil "github.com/PingCAP-QE/Naglfar/pkg/ssh"
	tiupSpec "github.com/pingcap/tiup/pkg/cluster/spec"
)

// Our controller node and worker nodes share the same insecure_key path
const insecureKeyPath = "/root/insecure_key"
const ContainerImage = "mahjonp/base-image"

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

func setServerConfigs(spec *tiupSpec.Specification, serverConfigs naglfarv1.ServerConfigs) error {
	unmarshalServerConfigToMaps := func(data []byte, object *map[string]interface{}) error {
		err := yaml.Unmarshal(data, object)
		return err
	}
	var (
		tidbConfigs = make(map[string]interface{}, 0)
		tikvConfigs = make(map[string]interface{}, 0)
		pdConfigs   = make(map[string]interface{}, 0)
	)
	for _, item := range []struct {
		object *map[string]interface{}
		config string
	}{{
		&tidbConfigs,
		serverConfigs.TiDB,
	}, {
		&tikvConfigs,
		serverConfigs.TiKV,
	}, {
		&pdConfigs,
		serverConfigs.PD,
	}} {
		err := unmarshalServerConfigToMaps([]byte(item.config), item.object)
		if err != nil {
			return err
		}
	}
	spec.ServerConfigs = tiupSpec.ServerConfigs{TiDB: tidbConfigs, TiKV: tikvConfigs, PD: pdConfigs}
	return nil
}

func BuildSpecification(ctf *naglfarv1.TestClusterTopologySpec, trs []*naglfarv1.TestResource) (spec tiupSpec.Specification, control *naglfarv1.TestResourceStatus, err error) {
	spec.GlobalOptions = tiupSpec.GlobalOptions{
		User:    "root",
		SSHPort: 22,
	}
	if err := setServerConfigs(&spec, ctf.TiDBCluster.ServerConfigs); err != nil {
		err = fmt.Errorf("set serverConfigs failed: %v", err)
		return spec, nil, err
	}
	resourceMaps := make(map[string]*naglfarv1.TestResourceStatus)
	for idx, resource := range trs {
		resourceMaps[resource.Name] = &trs[idx].Status
	}
	control, exist := resourceMaps[ctf.TiDBCluster.Control]
	log.Info("build specification", "resourceMaps", resourceMaps)
	if !exist {
		return spec, nil, fmt.Errorf("control node not found: `%s`", ctf.TiDBCluster.Control)
	}
	for _, item := range ctf.TiDBCluster.TiDB {
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("tidb node not found: `%s`", item.Host)
		}
		spec.TiDBServers = append(spec.TiDBServers, tiupSpec.TiDBSpec{
			Host:            node.ClusterIP,
			SSHPort:         22,
			Port:            0,
			StatusPort:      0,
			DeployDir:       item.DeployDir,
			LogDir:          item.LogDir,
			ResourceControl: meta.ResourceControl{},
		})
	}
	for _, item := range ctf.TiDBCluster.TiKV {
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("tikv node not found: `%s`", item.Host)
		}
		spec.TiKVServers = append(spec.TiKVServers, tiupSpec.TiKVSpec{
			Host:            node.ClusterIP,
			SSHPort:         22,
			Port:            0,
			StatusPort:      0,
			DeployDir:       item.DeployDir,
			DataDir:         item.DataDir,
			LogDir:          item.LogDir,
			ResourceControl: meta.ResourceControl{},
		})
	}
	for _, item := range ctf.TiDBCluster.PD {
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("pd node not found: `%s`", item.Host)
		}
		spec.PDServers = append(spec.PDServers, tiupSpec.PDSpec{
			Host:            node.ClusterIP,
			SSHPort:         22,
			ClientPort:      0,
			PeerPort:        0,
			DeployDir:       item.DeployDir,
			DataDir:         item.DataDir,
			LogDir:          item.LogDir,
			ResourceControl: meta.ResourceControl{},
		})
	}
	for _, item := range ctf.TiDBCluster.Monitor {
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("monitor node not found: `%s`", item.Host)
		}
		spec.Monitors = append(spec.Monitors, tiupSpec.PrometheusSpec{
			Host:            node.ClusterIP,
			SSHPort:         22,
			Port:            0,
			DeployDir:       item.DeployDir,
			DataDir:         item.DataDir,
			LogDir:          item.LogDir,
			ResourceControl: meta.ResourceControl{},
		})
	}
	for _, item := range ctf.TiDBCluster.Grafana {
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("grafana node not found: `%s`", item.Host)
		}
		spec.Grafana = append(spec.Grafana, tiupSpec.GrafanaSpec{
			Host:            node.ClusterIP,
			SSHPort:         22,
			Port:            0,
			DeployDir:       item.DeployDir,
			ResourceControl: meta.ResourceControl{},
		})
	}
	// set default values from tag
	defaults.Set(&spec)
	return
}

type ClusterManager struct {
	log     logr.Logger
	spec    *tiupSpec.Specification
	control *naglfarv1.TestResourceStatus
}

func MakeClusterManager(log logr.Logger, ctf *naglfarv1.TestClusterTopologySpec, trs []*naglfarv1.TestResource) (*ClusterManager, error) {
	specification, control, err := BuildSpecification(ctf, trs)
	if err != nil {
		return nil, err
	}
	return &ClusterManager{
		log:     log,
		spec:    &specification,
		control: control.DeepCopy(),
	}, nil
}

func (c *ClusterManager) InstallCluster(log logr.Logger, clusterName string, version naglfarv1.TiDBClusterVersion) error {
	outfile, err := yaml.Marshal(c.spec)
	if err != nil {
		return err
	}
	if err := c.writeTopologyFileOnControl(outfile); err != nil {
		return err
	}
	if err := c.deployCluster(log, clusterName, version.Version); err != nil {
		return err
	}
	return c.startCluster(clusterName)
}

func (c *ClusterManager) UninstallCluster(clusterName string) error {
	ssh := sshUtil.MakeSSHKeyConfig(c.control.Username, insecureKeyPath, c.control.HostIP, c.control.SSHPort)
	cmd := fmt.Sprintf("/root/.tiup/bin/tiup cluster destroy -y %s", clusterName)
	_, errStr, _, err := ssh.Run(cmd)
	if err != nil {
		c.log.Error(err, "run command on remote failed",
			"host", fmt.Sprintf("%s@%s:%d", c.control.Username, c.control.HostIP, c.control.SSHPort),
			"command", cmd,
			"stderr", errStr)

		// catch Error: tidb cluster `xxx` not exists
		if strings.Contains(errStr, "not exists") {
			return ErrClusterNotExist{clusterName: clusterName}
		}
		return fmt.Errorf("cannot run remote command `%s`: %s", cmd, err)
	}
	return nil
}

func (c *ClusterManager) writeTopologyFileOnControl(out []byte) error {
	log := c.log.WithName("clusterManager").WithName("writeTopologyFileOnControl")

	clientConfig, _ := auth.PrivateKey("root", insecureKeyPath, ssh.InsecureIgnoreHostKey())
	client := scp.NewClient(fmt.Sprintf("%s:%d", c.control.HostIP, c.control.SSHPort), &clientConfig)
	err := client.Connect()
	if err != nil {
		return fmt.Errorf("couldn't establish a connection to the remote server: %s", err)
	}
	log.V(2).Info("inspect the topology config generated", "value", string(out))
	if err := client.Copy(bytes.NewReader(out), "/root/topology.yaml", "0655", int64(len(out))); err != nil {
		return fmt.Errorf("error while copying file: %s", err)
	}
	return nil
}

func (c *ClusterManager) deployCluster(log logr.Logger, clusterName string, version string) error {
	ssh := sshUtil.MakeSSHKeyConfig(c.control.Username, insecureKeyPath, c.control.HostIP, c.control.SSHPort)
	cmd := fmt.Sprintf("/root/.tiup/bin/tiup cluster deploy -y %s %s /root/topology.yaml -i %s", clusterName, version, insecureKeyPath)
	_, errStr, _, err := ssh.Run(cmd)
	if err != nil {
		log.Error(err, "run command on remote failed",
			"host", fmt.Sprintf("%s@%s:%d", c.control.Username, c.control.HostIP, c.control.SSHPort),
			"command", cmd,
			"stderr", errStr)
		if strings.Contains(errStr, "specify another cluster name") {
			return ErrClusterDuplicated{clusterName: clusterName}
		}
		return fmt.Errorf("cannot run remote command `%s`: %s", cmd, err)
	}
	return nil
}

func (c *ClusterManager) startCluster(clusterName string) error {
	ssh := sshUtil.MakeSSHKeyConfig(c.control.Username, insecureKeyPath, c.control.HostIP, c.control.SSHPort)
	cmd := fmt.Sprintf("/root/.tiup/bin/tiup cluster start %s", clusterName)
	_, errStr, _, err := ssh.Run(cmd)
	if err != nil {
		c.log.Error(err, "run command on remote failed",
			"host", fmt.Sprintf("%s@%s:%d", c.control.Username, c.control.HostIP, c.control.SSHPort),
			"command", cmd,
			"stderr", errStr)
		return fmt.Errorf("cannot run remote command `%s`: %s", cmd, err)
	}
	return nil
}
