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
	"gopkg.in/yaml.v2"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	sshUtil "github.com/PingCAP-QE/Naglfar/pkg/ssh"
	"github.com/PingCAP-QE/Naglfar/pkg/tiup"
	tiupSpec "github.com/pingcap/tiup/components/dm/spec"
)

func setServerConfigs(spec *tiupSpec.Specification, serverConfigs naglfarv1.DMServerConfigs) error {
	unmarshalServerConfigToMaps := func(data []byte, object *map[string]interface{}) error {
		err := yaml.Unmarshal(data, object)
		return err
	}
	var (
		masterConfigs = make(map[string]interface{})
		workerConfigs = make(map[string]interface{})
	)
	for _, item := range []struct {
		object *map[string]interface{}
		config string
	}{{
		&masterConfigs,
		serverConfigs.Master,
	}, {
		&workerConfigs,
		serverConfigs.Worker,
	},
	} {
		err := unmarshalServerConfigToMaps([]byte(item.config), item.object)
		if err != nil {
			return err
		}
	}
	spec.ServerConfigs = tiupSpec.DMServerConfigs{Master: masterConfigs, Worker: workerConfigs}
	return nil
}

func setMasterConfig(spec *tiupSpec.Specification, masterConfig string, index int) error {
	unmarshalMasterConfigToMap := func(data []byte, object *map[string]interface{}) error {
		err := yaml.Unmarshal(data, object)
		return err
	}
	var (
		config = make(map[string]interface{})
	)
	for _, item := range []struct {
		object *map[string]interface{}
		config string
	}{{
		&config,
		masterConfig,
	}} {
		err := unmarshalMasterConfigToMap([]byte(item.config), item.object)
		if err != nil {
			return err
		}
	}
	spec.Masters[index].Config = config
	return nil
}

func setWorkerConfig(spec *tiupSpec.Specification, workerConfig string, index int) error {
	unmarshalWorkerConfigToMap := func(data []byte, object *map[string]interface{}) error {
		err := yaml.Unmarshal(data, object)
		return err
	}
	var (
		config = make(map[string]interface{})
	)
	for _, item := range []struct {
		object *map[string]interface{}
		config string
	}{{
		&config,
		workerConfig,
	}} {
		err := unmarshalWorkerConfigToMap([]byte(item.config), item.object)
		if err != nil {
			return err
		}
	}
	spec.Workers[index].Config = config
	return nil
}

// If dryRun is true, host uses the resource's name
func BuildSpecification(ctf *naglfarv1.TestClusterTopologySpec, trs []*naglfarv1.TestResource, dryRun bool, clusterIPMaps ...map[string]string) (
	spec tiupSpec.Specification, control *naglfarv1.TestResourceStatus, err error) {
	hostName := func(resourceName, clusterIP string) string {
		if dryRun {
			return resourceName
		}
		return clusterIP
	}
	spec.GlobalOptions = tiupSpec.GlobalOptions{
		User:    "root",
		SSHPort: 22,
	}
	if ctf.DMCluster.Global != nil {
		spec.GlobalOptions.DataDir = ctf.DMCluster.Global.DataDir
		spec.GlobalOptions.DeployDir = ctf.DMCluster.Global.DeployDir
	}
	if err := setServerConfigs(&spec, ctf.DMCluster.ServerConfigs); err != nil {
		err = fmt.Errorf("set serverConfigs failed: %v", err)
		return spec, nil, err
	}
	resourceMaps := make(map[string]*naglfarv1.TestResourceStatus)
	for idx, resource := range trs {
		resourceMaps[resource.Name] = &trs[idx].Status
	}
	control, exist := resourceMaps[ctf.DMCluster.Control]
	if !exist {
		return spec, nil, fmt.Errorf("control node not found: `%s`", ctf.DMCluster.Control)
	}
	for index, item := range ctf.DMCluster.Master {
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("master node not found: `%s`", item.Host)
		}
		spec.Masters = append(spec.Masters, tiupSpec.MasterSpec{
			Host:            hostName(item.Host, node.ClusterIP),
			SSHPort:         22,
			PeerPort:        item.PeerPort,
			Port:            item.Port,
			DataDir:         item.DataDir,
			DeployDir:       item.DeployDir,
			LogDir:          item.LogDir,
			ResourceControl: meta.ResourceControl{},
		})
		if err := setMasterConfig(&spec, item.Config, index); err != nil {
			err = fmt.Errorf("set MasterConfigs failed: %v", err)
			return spec, nil, err
		}
	}
	for index, item := range ctf.DMCluster.Worker {
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("worker node not found: `%s`", item.Host)
		}
		spec.Workers = append(spec.Workers, tiupSpec.WorkerSpec{
			Host:            hostName(item.Host, node.ClusterIP),
			SSHPort:         22,
			Port:            item.Port,
			DeployDir:       item.DeployDir,
			LogDir:          item.LogDir,
			ResourceControl: meta.ResourceControl{},
		})
		if err := setWorkerConfig(&spec, item.Config, index); err != nil {
			err = fmt.Errorf("set WorkerConfigs failed: %v", err)
			return spec, nil, err
		}
	}

	for _, item := range ctf.DMCluster.Monitor {
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("monitor node not found: `%s`", item.Host)
		}
		spec.Monitors = append(spec.Monitors, tiupSpec.PrometheusSpec{
			Host:            hostName(item.Host, node.ClusterIP),
			SSHPort:         22,
			Port:            item.Port,
			DeployDir:       item.DeployDir,
			DataDir:         item.DataDir,
			LogDir:          item.LogDir,
			ResourceControl: meta.ResourceControl{},
		})
	}
	for _, item := range ctf.DMCluster.Grafana {
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("grafana node not found: `%s`", item.Host)
		}
		spec.Grafanas = append(spec.Grafanas, tiupSpec.GrafanaSpec{
			Host:            hostName(item.Host, node.ClusterIP),
			SSHPort:         22,
			Port:            item.Port,
			DeployDir:       item.DeployDir,
			ResourceControl: meta.ResourceControl{},
		})
	}

	for _, item := range ctf.DMCluster.AlertManager {
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("alertManager node not found: `%s`", item.Host)
		}
		spec.Alertmanagers = append(spec.Alertmanagers, tiupSpec.AlertmanagerSpec{
			Host:            hostName(item.Host, node.ClusterIP),
			SSHPort:         22,
			ClusterPort:     item.ClusterPort,
			WebPort:         item.WebPort,
			DeployDir:       item.DeployDir,
			DataDir:         item.DataDir,
			LogDir:          item.LogDir,
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

func MakeClusterManager(log logr.Logger, ctf *naglfarv1.TestClusterTopologySpec, trs []*naglfarv1.TestResource, clusterIPMaps ...map[string]string) (*ClusterManager, error) {
	var specification tiupSpec.Specification
	var control *naglfarv1.TestResourceStatus
	var err error
	specification, control, err = BuildSpecification(ctf, trs, false, clusterIPMaps...)
	if err != nil {
		return nil, err
	}
	return &ClusterManager{
		log:     log,
		spec:    &specification,
		control: control.DeepCopy(),
	}, nil
}

func (c *ClusterManager) InstallCluster(log logr.Logger, clusterName string, version string) error {
	outfile, err := yaml.Marshal(c.spec)
	if err != nil {
		return err
	}
	if err := c.writeTopologyFileOnControl(outfile); err != nil {
		return err
	}
	if err := c.deployCluster(log, clusterName, version); tiup.IgnoreClusterDuplicated(err) != nil {
		return err
	}
	if err := c.startCluster(clusterName); err != nil {
		return err
	}
	return nil
}

func (c *ClusterManager) UninstallCluster(clusterName string) error {
	client, err := sshUtil.NewSSHClient("root", tiup.InsecureKeyPath, c.control.HostIP, c.control.SSHPort)
	if err != nil {
		return err
	}
	defer client.Close()
	cmd := fmt.Sprintf("/root/.tiup/bin/tiup dm destroy -y %s", clusterName)
	stdStr, errStr, err := client.RunCommand(cmd)
	if err != nil {
		c.log.Error(err, "run command on remote failed",
			"host", fmt.Sprintf("%s@%s:%d", "root", c.control.HostIP, c.control.SSHPort),
			"command", cmd,
			"stdStr", stdStr,
			"stderr", errStr)
		// catch Error: dm cluster `xxx` not exists
		if strings.Contains(errStr, "not exists") {
			return tiup.ErrClusterNotExist{ClusterName: clusterName}
		}
		return fmt.Errorf("cannot run remote command `%s`: %s", cmd, err)
	}
	return nil
}

func (c *ClusterManager) writeTopologyFileOnControl(out []byte) error {
	clientConfig, _ := auth.PrivateKey("root", tiup.InsecureKeyPath, tiup.InsecureIgnoreHostKey())
	client := scp.NewClient(fmt.Sprintf("%s:%d", c.control.HostIP, c.control.SSHPort), &clientConfig)
	err := client.Connect()
	if err != nil {
		return fmt.Errorf("couldn't establish a connection to the remote server: %s", err)
	}
	if err := client.Copy(bytes.NewReader(out), "/root/topology.yaml", "0655", int64(len(out))); err != nil {
		return fmt.Errorf("error while copying file: %s", err)
	}
	return nil
}

func (c *ClusterManager) deployCluster(log logr.Logger, clusterName string, version string) error {
	client, err := sshUtil.NewSSHClient("root", tiup.InsecureKeyPath, c.control.HostIP, c.control.SSHPort)
	if err != nil {
		return err
	}
	defer client.Close()
	cmd := fmt.Sprintf("/root/.tiup/bin/tiup dm deploy -y %s %s /root/topology.yaml -i %s", clusterName, version, tiup.InsecureKeyPath)
	stdStr, errStr, err := client.RunCommand(cmd)
	if err != nil {
		log.Error(err, "run command on remote failed",
			"host", fmt.Sprintf("%s@%s:%d", "root", c.control.HostIP, c.control.SSHPort),
			"command", cmd,
			"stdout", stdStr,
			"stderr", errStr)
		if strings.Contains(errStr, "specify another cluster name") {
			return tiup.ErrClusterDuplicated{ClusterName: clusterName}
		}
		return fmt.Errorf("deploy cluster failed(%s): %s", err, errStr)
	}
	return nil
}

func (c *ClusterManager) startCluster(clusterName string) error {
	client, err := sshUtil.NewSSHClient("root", tiup.InsecureKeyPath, c.control.HostIP, c.control.SSHPort)
	if err != nil {
		return err
	}
	defer client.Close()
	cmd := fmt.Sprintf("/root/.tiup/bin/tiup dm start %s", clusterName)
	stdout, errStr, err := client.RunCommand(cmd)
	if err != nil {
		c.log.Error(err, "run command on remote failed",
			"host", fmt.Sprintf("%s@%s:%d", "root", c.control.HostIP, c.control.SSHPort),
			"command", cmd,
			"stdout", stdout,
			"stderr", errStr)
		return fmt.Errorf("start cluster failed: %s", err)
	}
	return nil
}
