package cluster

import (
	"bytes"
	"fmt"
	"net/url"
	"path"
	"reflect"
	"strconv"
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
	tiupSpec "github.com/pingcap/tiup/pkg/cluster/spec"
)

func setPumpConfig(spec *tiupSpec.Specification, pumpConfig string, index int) error {
	unmarshalPumpConfigToMap := func(data []byte, object *map[string]interface{}) error {
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
		pumpConfig,
	}} {
		err := unmarshalPumpConfigToMap([]byte(item.config), item.object)
		if err != nil {
			return err
		}
	}
	spec.PumpServers[index].Config = config
	return nil
}

func setTiFlashConfig(spec *tiupSpec.Specification, tiflashConfig string, index int) error {
	unmarshalTiFlashConfigToMap := func(data []byte, object *map[string]interface{}) error {
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
		tiflashConfig,
	}} {
		err := unmarshalTiFlashConfigToMap([]byte(item.config), item.object)
		if err != nil {
			return err
		}
	}
	spec.TiFlashServers[index].Config = config
	return nil
}

func setTiFlashLearnerConfig(spec *tiupSpec.Specification, tiflashLearnerConfig string, index int) error {
	unmarshalTiFlashLearnerConfigToMap := func(data []byte, object *map[string]interface{}) error {
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
		tiflashLearnerConfig,
	}} {
		err := unmarshalTiFlashLearnerConfigToMap([]byte(item.config), item.object)
		if err != nil {
			return err
		}
	}
	spec.TiFlashServers[index].LearnerConfig = config
	return nil
}

func setDrainerConfig(spec *tiupSpec.Specification, drainerConfig string, index int, clusterIPMaps ...map[string]string) error {
	unmarshalDrainerConfigToMap := func(data []byte, object *map[string]interface{}) error {
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
		drainerConfig,
	}} {
		err := unmarshalDrainerConfigToMap([]byte(item.config), item.object)
		if err != nil {
			return err
		}
	}
	// transfer
	if clusterIPMaps != nil {
		clusterIP := clusterIPMaps[0][strings.Split(config["syncer.to.host"].(string), "://")[1]]
		if clusterIP == "" {
			return fmt.Errorf("syncer.to.host is not existed")
		} else {
			config["syncer.to.host"] = clusterIP
		}
	}
	spec.Drainers[index].Config = config
	return nil
}

func setServerConfigs(spec *tiupSpec.Specification, serverConfigs naglfarv1.ServerConfigs) error {
	unmarshalServerConfigToMaps := func(data []byte, object *map[string]interface{}) error {
		err := yaml.Unmarshal(data, object)
		return err
	}
	var (
		tidbConfigs = make(map[string]interface{})
		tikvConfigs = make(map[string]interface{})
		pdConfigs   = make(map[string]interface{})
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

func IsServerConfigModified(pre naglfarv1.ServerConfigs, cur naglfarv1.ServerConfigs) bool {
	return !reflect.DeepEqual(pre, cur)
}

func IsClusterConfigModified(pre *naglfarv1.TiDBCluster, cur *naglfarv1.TiDBCluster) bool {
	return !reflect.DeepEqual(pre, cur)
}

func IsScaleIn(pre *naglfarv1.TiDBCluster, cur *naglfarv1.TiDBCluster) bool {
	return len(pre.TiDB) > len(cur.TiDB) || len(pre.PD) > len(cur.PD) || len(pre.TiKV) > len(cur.TiKV)
}

func IsScaleOut(pre *naglfarv1.TiDBCluster, cur *naglfarv1.TiDBCluster) bool {
	return len(pre.TiDB) < len(cur.TiDB) || len(pre.PD) < len(cur.PD) || len(pre.TiKV) < len(cur.TiKV)
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
	if ctf.TiDBCluster.Global != nil {
		spec.GlobalOptions.DataDir = ctf.TiDBCluster.Global.DataDir
		spec.GlobalOptions.DeployDir = ctf.TiDBCluster.Global.DeployDir
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
	if !exist {
		return spec, nil, fmt.Errorf("control node not found: `%s`", ctf.TiDBCluster.Control)
	}
	for _, item := range ctf.TiDBCluster.TiDB {
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("tidb node not found: `%s`", item.Host)
		}
		spec.TiDBServers = append(spec.TiDBServers, tiupSpec.TiDBSpec{
			Host:            hostName(item.Host, node.ClusterIP),
			SSHPort:         22,
			Port:            item.Port,
			StatusPort:      item.StatusPort,
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
			Host:            hostName(item.Host, node.ClusterIP),
			SSHPort:         22,
			Port:            item.Port,
			StatusPort:      item.StatusPort,
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
			Host:            hostName(item.Host, node.ClusterIP),
			SSHPort:         22,
			ClientPort:      item.ClientPort,
			PeerPort:        item.PeerPort,
			DeployDir:       item.DeployDir,
			DataDir:         item.DataDir,
			LogDir:          item.LogDir,
			ResourceControl: meta.ResourceControl{},
		})
	}
	for index, item := range ctf.TiDBCluster.Pump {
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("pump node not found: `%s`", item.Host)
		}
		spec.PumpServers = append(spec.PumpServers, tiupSpec.PumpSpec{
			Host:      hostName(item.Host, node.ClusterIP),
			Port:      item.Port,
			SSHPort:   item.SSHPort,
			DeployDir: item.DeployDir,
			DataDir:   item.DataDir,
		})
		if err := setPumpConfig(&spec, ctf.TiDBCluster.Pump[index].Config, index); err != nil {
			err = fmt.Errorf("set PumpConfigs failed: %v", err)
			return spec, nil, err
		}
	}
	for index, item := range ctf.TiDBCluster.Drainer {
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("drainer node not found: `%s`", item.Host)
		}
		spec.Drainers = append(spec.Drainers, tiupSpec.DrainerSpec{
			Host:      hostName(item.Host, node.ClusterIP),
			Port:      item.Port,
			SSHPort:   item.SSHPort,
			CommitTS:  item.CommitTS,
			DeployDir: item.DeployDir,
			DataDir:   item.DataDir,
		})
		if err := setDrainerConfig(&spec, ctf.TiDBCluster.Drainer[index].Config, index, clusterIPMaps...); err != nil {
			err = fmt.Errorf("set DrainerConfigs failed: %v", err)
			return spec, nil, err
		}
	}
	for _, item := range ctf.TiDBCluster.Monitor {
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
	for _, item := range ctf.TiDBCluster.Grafana {
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
	for _, item := range ctf.TiDBCluster.CDC {
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("cdc node not found: `%s`", item.Host)
		}
		spec.CDCServers = append(spec.CDCServers, tiupSpec.CDCSpec{
			Host:            hostName(item.Host, node.ClusterIP),
			SSHPort:         22,
			Port:            item.Port,
			DeployDir:       item.DeployDir,
			LogDir:          item.LogDir,
			ResourceControl: meta.ResourceControl{},
		})
	}

	for index, item := range ctf.TiDBCluster.TiFlash {
		node, exist := resourceMaps[item.Host]
		if !exist {
			return spec, nil, fmt.Errorf("tiflash node not found: `%s`", item.Host)
		}
		spec.TiFlashServers = append(spec.TiFlashServers, tiupSpec.TiFlashSpec{
			Host:                 hostName(item.Host, node.ClusterIP),
			SSHPort:              22,
			TCPPort:              item.TCPPort,
			HTTPPort:             item.HTTPPort,
			FlashServicePort:     item.ServicePort,
			FlashProxyPort:       item.ProxyPort,
			FlashProxyStatusPort: item.ProxyStatusPort,
			StatusPort:           item.ProxyStatusPort,
			DeployDir:            item.DeployDir,
			DataDir:              item.DataDir,
			LogDir:               item.LogDir,
			ResourceControl:      meta.ResourceControl{},
		})
		if err := setTiFlashConfig(&spec, ctf.TiDBCluster.TiFlash[index].Config, index); err != nil {
			err = fmt.Errorf("set TiFlashConfigs failed: %v", err)
			return spec, nil, err
		}
		if err := setTiFlashLearnerConfig(&spec, ctf.TiDBCluster.TiFlash[index].LearnerConfig, index); err != nil {
			err = fmt.Errorf("set TiFlashLearnerConfigs failed: %v", err)
			return spec, nil, err
		}
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

func (c *ClusterManager) InstallCluster(log logr.Logger, clusterName string, version naglfarv1.TiDBClusterVersion) error {
	outfile, err := yaml.Marshal(c.spec)
	if err != nil {
		return err
	}
	if err := c.writeTopologyFileOnControl(outfile); err != nil {
		return err
	}
	if err := c.deployCluster(log, clusterName, version.Version); tiup.IgnoreClusterDuplicated(err) != nil {
		return err
	}
	if err := c.startCluster(clusterName); err != nil {
		return err
	}
	if c.shouldPatch(version) {
		return c.patch(clusterName, version)
	}
	return nil
}

func (c *ClusterManager) UpdateCluster(log logr.Logger, clusterName string, ct *naglfarv1.TestClusterTopology) error {
	type Meta struct {
		User        string `json:"user,omitempty"`
		TiDBVersion string `yaml:"tidb_version" json:"tidb_version,omitempty"`
		Topology    *tiupSpec.Specification
	}
	var meta Meta
	meta.User = "root"
	meta.TiDBVersion = ct.Spec.TiDBCluster.Version.Version
	meta.Topology = c.spec
	outfile, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	if err := c.writeTemporaryTopologyMetaOnControl(outfile, clusterName); err != nil {
		return err
	}

	roles := c.diffServerConfigs(ct.Status.PreTiDBCluster.ServerConfigs, ct.Spec.TiDBCluster.ServerConfigs)

	log.Info("RequestTopology is modified.", "changed roles", roles)

	if err := c.reloadCluster(clusterName, []string{}, roles); err != nil {
		return err
	}
	return nil
}

func (c *ClusterManager) ScaleInCluster(log logr.Logger, clusterName string, ct *naglfarv1.TestClusterTopology, clusterIPMaps map[string]string) error {
	nodes := c.getScaleInNode(*ct.Status.PreTiDBCluster, *ct.Spec.TiDBCluster, clusterIPMaps)

	log.Info("RequestTopology is modified.", "scale-in nodes", nodes)

	var nodesStr string
	for i := 0; i < len(nodes); i++ {
		nodesStr += " --node " + nodes[i]
	}

	client, err := sshUtil.NewSSHClient("root", tiup.InsecureKeyPath, c.control.HostIP, c.control.SSHPort)
	if err != nil {
		return err
	}
	defer client.Close()
	cmd := fmt.Sprintf("/root/.tiup/bin/tiup cluster scale-in %s %s -y", clusterName, nodesStr)
	stdStr, errStr, err := client.RunCommand(cmd)
	if err != nil {
		log.Error(err, "run command on remote failed",
			"host", fmt.Sprintf("%s@%s:%d", "root", c.control.HostIP, c.control.SSHPort),
			"command", cmd,
			"stdout", stdStr,
			"stderr", errStr)
		return fmt.Errorf("scale-in cluster failed(%s): %s", err, errStr)
	}
	return nil
}

func (c *ClusterManager) PruneCluster(log logr.Logger, clusterName string, ct *naglfarv1.TestClusterTopology) error {
	client, err := sshUtil.NewSSHClient("root", tiup.InsecureKeyPath, c.control.HostIP, c.control.SSHPort)
	if err != nil {
		return err
	}
	defer client.Close()
	cmd := fmt.Sprintf("/root/.tiup/bin/tiup cluster prune %s -y", clusterName)
	stdStr, errStr, err := client.RunCommand(cmd)
	if err != nil {
		log.Error(err, "run command on remote failed",
			"host", fmt.Sprintf("%s@%s:%d", "root", c.control.HostIP, c.control.SSHPort),
			"command", cmd,
			"stdout", stdStr,
			"stderr", errStr)
		return fmt.Errorf("prune cluster failed(%s): %s", err, errStr)
	}
	return nil
}

func (c *ClusterManager) GetNodeStatusList(log logr.Logger, clusterName string, status string) ([]string, error) {
	client, err := sshUtil.NewSSHClient("root", tiup.InsecureKeyPath, c.control.HostIP, c.control.SSHPort)
	if err != nil {
		return []string{}, err
	}
	defer client.Close()
	cmd := fmt.Sprintf("/root/.tiup/bin/tiup cluster display %s", clusterName)
	stdStr, errStr, err := client.RunCommand(cmd)
	if err != nil {
		log.Error(err, "run command on remote failed",
			"host", fmt.Sprintf("%s@%s:%d", "root", c.control.HostIP, c.control.SSHPort),
			"command", cmd,
			"stdout", stdStr,
			"stderr", errStr)
		return []string{}, fmt.Errorf("scale-in cluster failed(%s): %s", err, errStr)
	}
	lines := strings.Split(stdStr, "\n")
	var list []string
	for i := 0; i < len(lines); i++ {
		if strings.Contains(lines[i], status) {
			parts := strings.Split(lines[i], " ")
			list = append(list, parts[0])
		}
	}
	return list, nil
}

func (c *ClusterManager) ScaleOutCluster(log logr.Logger, clusterName string, ct *naglfarv1.TestClusterTopology, clusterIPMaps map[string]string) error {
	type ScaleOut struct {
		TiDB []naglfarv1.TiDBSpec `yaml:"tidb_servers,omitempty"`

		TiKV []naglfarv1.TiKVSpec `yaml:"tikv_servers,omitempty"`

		PD []naglfarv1.PDSpec `yaml:"pd_servers,omitempty"`
	}

	var scaleOut ScaleOut
	tidbs, pds, tikvs := c.getScaleOutRoles(*ct.Status.PreTiDBCluster, *ct.Spec.TiDBCluster, clusterIPMaps)
	var scaleOutStr string
	scaleOutStr += "tidbs: "
	for i := 0; i < len(tidbs); i++ {
		scaleOutStr += tidbs[i].Host + ":" + strconv.Itoa(tidbs[i].Port) + " "
	}
	scaleOutStr += "pds: "
	for i := 0; i < len(pds); i++ {
		scaleOutStr += pds[i].Host + ":" + strconv.Itoa(pds[i].ClientPort) + " "
	}
	scaleOutStr += "tikvs: "
	for i := 0; i < len(tikvs); i++ {
		scaleOutStr += tikvs[i].Host + ":" + strconv.Itoa(tikvs[i].Port) + " "
	}
	log.Info("RequestTopology is modified.", "scale-out nodes: ", scaleOutStr)
	scaleOut.TiDB = tidbs
	scaleOut.PD = pds
	scaleOut.TiKV = tikvs
	outfile, err := yaml.Marshal(scaleOut)

	if err := c.writeScaleOutFileOnControl(outfile); err != nil {
		return err
	}

	if err != nil {
		return err
	}
	client, err := sshUtil.NewSSHClient("root", tiup.InsecureKeyPath, c.control.HostIP, c.control.SSHPort)
	if err != nil {
		return err
	}
	defer client.Close()
	cmd := fmt.Sprintf("/root/.tiup/bin/tiup cluster scale-out %s /root/scale-out.yaml -i %s -y", clusterName, tiup.InsecureKeyPath)
	stdStr, errStr, err := client.RunCommand(cmd)
	if err != nil {
		log.Error(err, "run command on remote failed",
			"host", fmt.Sprintf("%s@%s:%d", "root", c.control.HostIP, c.control.SSHPort),
			"command", cmd,
			"stdout", stdStr,
			"stderr", errStr)
		return fmt.Errorf("scale-out cluster failed(%s): %s", err, errStr)
	}
	return nil
}

func (c *ClusterManager) UninstallCluster(clusterName string) error {
	client, err := sshUtil.NewSSHClient("root", tiup.InsecureKeyPath, c.control.HostIP, c.control.SSHPort)
	if err != nil {
		return err
	}
	defer client.Close()
	cmd := fmt.Sprintf("/root/.tiup/bin/tiup cluster destroy -y %s", clusterName)
	stdStr, errStr, err := client.RunCommand(cmd)
	if err != nil {
		c.log.Error(err, "run command on remote failed",
			"host", fmt.Sprintf("%s@%s:%d", "root", c.control.HostIP, c.control.SSHPort),
			"command", cmd,
			"stdStr", stdStr,
			"stderr", errStr)
		// catch Error: tidb cluster `xxx` not exists
		if strings.Contains(errStr, "not exists") {
			return tiup.ErrClusterNotExist{ClusterName: clusterName}
		}
		return fmt.Errorf("cannot run remote command `%s`: %s", cmd, err)
	}
	return nil
}

func (c *ClusterManager) diffServerConfigs(pre naglfarv1.ServerConfigs, cur naglfarv1.ServerConfigs) []string {
	var roles []string
	if !reflect.DeepEqual(pre.TiDB, cur.TiDB) {
		roles = append(roles, "tidb")
	}
	if !reflect.DeepEqual(pre.PD, cur.PD) {
		roles = append(roles, "pd")
	}
	if !reflect.DeepEqual(pre.TiKV, cur.TiKV) {
		roles = append(roles, "tikv")
	}
	return roles
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

func (c *ClusterManager) writeScaleOutFileOnControl(out []byte) error {
	clientConfig, err := auth.PrivateKey("root", tiup.InsecureKeyPath, tiup.InsecureIgnoreHostKey())
	if err != nil {
		return fmt.Errorf("generate client privatekey failed: %s", err)
	}
	client := scp.NewClient(fmt.Sprintf("%s:%d", c.control.HostIP, c.control.SSHPort), &clientConfig)
	err = client.Connect()
	if err != nil {
		return fmt.Errorf("couldn't establish a connection to the remote server: %s", err)
	}
	if err := client.Copy(bytes.NewReader(out), "/root/scale-out.yaml", "0655", int64(len(out))); err != nil {
		return fmt.Errorf("error while copying file: %s", err)
	}
	return nil
}

func (c *ClusterManager) writeTemporaryTopologyMetaOnControl(out []byte, clusterName string) error {
	clientConfig, _ := auth.PrivateKey("root", tiup.InsecureKeyPath, tiup.InsecureIgnoreHostKey())
	client := scp.NewClient(fmt.Sprintf("%s:%d", c.control.HostIP, c.control.SSHPort), &clientConfig)
	err := client.Connect()
	if err != nil {
		return fmt.Errorf("couldn't establish a connection to the remote server: %s", err)
	}

	if err := client.Copy(bytes.NewReader(out), "/tmp/meta.yaml", "0655", int64(len(out))); err != nil {
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
	cmd := fmt.Sprintf("/root/.tiup/bin/tiup cluster deploy -y %s %s /root/topology.yaml -i %s", clusterName, version, tiup.InsecureKeyPath)
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
	cmd := fmt.Sprintf("/root/.tiup/bin/tiup cluster start %s", clusterName)
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

func (c *ClusterManager) reloadCluster(clusterName string, nodes []string, roles []string) error {
	client, err := sshUtil.NewSSHClient("root", tiup.InsecureKeyPath, c.control.HostIP, c.control.SSHPort)
	if err != nil {
		return err
	}
	defer client.Close()

	var roleStr string
	for i := 0; i < len(roles); i++ {
		roleStr += " -R " + roles[i]
	}
	moveCmd := fmt.Sprintf("mv /tmp/meta.yaml /root/.tiup/storage/cluster/clusters/" + clusterName + "/meta.yaml")
	reloadCmd := fmt.Sprintf("/root/.tiup/bin/tiup cluster reload %s %s --ignore-config-check", clusterName, roleStr)
	combineCmd := moveCmd + ";" + reloadCmd
	cmd := fmt.Sprintf(`flock -n /tmp/naglfar.tiup.lock -c "%s"`, combineCmd)
	stdStr, errStr, err := client.RunCommand(cmd)
	if err != nil {
		c.log.Error(err, "run command on remote failed",
			"host", fmt.Sprintf("%s@%s:%d", "root", c.control.HostIP, c.control.SSHPort),
			"command", cmd,
			"stdStr", stdStr,
			"stderr", errStr)
		return fmt.Errorf("cannot run remote command `%s`: %s", cmd, err)
	}
	return nil
}

func (c *ClusterManager) shouldPatch(version naglfarv1.TiDBClusterVersion) bool {
	return version.TiDBDownloadURL != "" || version.PDDownloadUrl != "" || version.TiKVDownloadURL != ""
}

func (c *ClusterManager) patch(clusterName string, version naglfarv1.TiDBClusterVersion) error {
	type component struct {
		componentName string
		downloadURL   string
	}
	client, err := sshUtil.NewSSHClient("root", tiup.InsecureKeyPath, c.control.HostIP, c.control.SSHPort)
	if err != nil {
		return err
	}
	defer client.Close()
	commands := []string{"set -ex", "rm -rf components && mkdir components"}
	var patchComponents []component
	var patchComponentNames []string
	if version.TiDBDownloadURL != "" {
		patchComponents = append(patchComponents, component{
			componentName: "tidb-server",
			downloadURL:   version.TiDBDownloadURL,
		})
		patchComponentNames = append(patchComponentNames, "tidb-server")
	}
	if version.TiKVDownloadURL != "" {
		patchComponents = append(patchComponents, component{
			componentName: "tikv-server",
			downloadURL:   version.TiKVDownloadURL,
		})
		patchComponentNames = append(patchComponentNames, "tikv-server")
	}
	if version.PDDownloadUrl != "" {
		patchComponents = append(patchComponents, component{
			componentName: "pd-server",
			downloadURL:   version.PDDownloadUrl,
		})
		patchComponentNames = append(patchComponentNames, "pd-server")
	}
	for _, component := range patchComponents {
		downloadURL := component.downloadURL
		u, err := url.Parse(downloadURL)
		if err != nil {
			return err
		}
		commands = append(commands, fmt.Sprintf(`curl -O %s`, downloadURL), c.GenUnzipCommand(path.Base(u.Path), "components"), fmt.Sprintf("rm -rf components/%s", component.componentName), fmt.Sprintf("mv components/bin/%s components/%s", component.componentName, component.componentName))
	}
	commands = append(commands, "cd components && tar zcf patch.tar.gz "+strings.Join(patchComponentNames, " "))
	for _, component := range patchComponentNames {
		commands = append(commands, fmt.Sprintf("/root/.tiup/bin/tiup cluster patch %s patch.tar.gz -R %s",
			clusterName, strings.Split(component, "-server")[0]))
	}
	cmd := fmt.Sprintf(`flock -n /tmp/naglfar.tiup.lock -c "%s"`, strings.Join(commands, "\n"))
	stdStr, errStr, err := client.RunCommand(cmd)
	if err != nil {
		c.log.Error(err, "run command on remote failed",
			"host", fmt.Sprintf("%s@%s:%d", "root", c.control.HostIP, c.control.SSHPort),
			"command", cmd,
			"stdout", stdStr,
			"stderr", errStr)
		return fmt.Errorf("patch cluster failed(%s): %s", err, errStr)
	}
	return nil
}

func (c *ClusterManager) GenUnzipCommand(filePath string, destDir string) string {
	if strings.HasSuffix(filePath, ".zip") {
		return "unzip -d " + destDir + " " + filePath
	}
	if strings.HasSuffix(filePath, ".tar.gz") {
		return "tar -zxf " + filePath + " -C " + destDir
	}
	return "tar -xf " + filePath + " -C " + destDir
}

func (c *ClusterManager) getScaleInNode(pre naglfarv1.TiDBCluster, cur naglfarv1.TiDBCluster, clusterIPMaps map[string]string) []string {
	var result []string
	for i := 0; i < len(pre.TiDB); i++ {
		var isExisted bool
		for j := 0; j < len(cur.TiDB); j++ {
			if reflect.DeepEqual(pre.TiDB[i], cur.TiDB[j]) {
				isExisted = true
			}
		}
		if !isExisted {
			if pre.TiDB[i].Port == 0 {
				pre.TiDB[i].Port = 4000
			}
			node := clusterIPMaps[pre.TiDB[i].Host] + ":" + strconv.Itoa(pre.TiDB[i].Port)
			result = append(result, node)
		}
	}
	for i := 0; i < len(pre.PD); i++ {
		var isExisted bool
		for j := 0; j < len(cur.PD); j++ {
			if reflect.DeepEqual(pre.PD[i], cur.PD[j]) {
				isExisted = true
			}
		}
		if !isExisted {
			if pre.PD[i].ClientPort == 0 {
				pre.PD[i].ClientPort = 2379
			}
			node := clusterIPMaps[pre.PD[i].Host] + ":" + strconv.Itoa(pre.PD[i].ClientPort)
			result = append(result, node)
		}
	}
	for i := 0; i < len(pre.TiKV); i++ {
		var isExisted bool
		for j := 0; j < len(cur.TiKV); j++ {
			if reflect.DeepEqual(pre.TiKV[i], cur.TiKV[j]) {
				isExisted = true
			}
		}
		if !isExisted {
			if pre.TiKV[i].Port == 0 {
				pre.TiKV[i].Port = 20160
			}
			node := clusterIPMaps[pre.TiKV[i].Host] + ":" + strconv.Itoa(pre.TiKV[i].Port)
			result = append(result, node)
		}
	}
	return result
}

func (c *ClusterManager) getScaleOutRoles(pre naglfarv1.TiDBCluster, cur naglfarv1.TiDBCluster, clusterIPMaps map[string]string) ([]naglfarv1.TiDBSpec, []naglfarv1.PDSpec, []naglfarv1.TiKVSpec) {

	var tidbs []naglfarv1.TiDBSpec
	for i := 0; i < len(cur.TiDB); i++ {
		var isExisted bool
		for j := 0; j < len(pre.TiDB); j++ {
			if reflect.DeepEqual(cur.TiDB[i], pre.TiDB[j]) {
				isExisted = true
			}
		}
		if !isExisted {
			tmp := cur.TiDB[i]
			tmp.Host = clusterIPMaps[tmp.Host]
			if tmp.Port == 0 {
				tmp.Port = 4000
			}
			if tmp.StatusPort == 0 {
				tmp.StatusPort = 10080
			}
			tidbs = append(tidbs, tmp)
		}
	}

	var pds []naglfarv1.PDSpec
	for i := 0; i < len(cur.PD); i++ {
		var isExisted bool
		for j := 0; j < len(pre.PD); j++ {
			if reflect.DeepEqual(cur.PD[i], pre.PD[j]) {
				isExisted = true
			}
		}
		if !isExisted {
			tmp := cur.PD[i]
			tmp.Host = clusterIPMaps[tmp.Host]

			if tmp.ClientPort == 0 {
				tmp.ClientPort = 2379
			}
			if tmp.PeerPort == 0 {
				tmp.PeerPort = 2380
			}
			pds = append(pds, tmp)
		}
	}

	var tikvs []naglfarv1.TiKVSpec
	for i := 0; i < len(cur.TiKV); i++ {
		var isExisted bool
		for j := 0; j < len(pre.TiKV); j++ {
			if reflect.DeepEqual(cur.TiKV[i], pre.TiKV[j]) {
				isExisted = true
			}
		}
		if !isExisted {
			tmp := cur.TiKV[i]
			tmp.Host = clusterIPMaps[tmp.Host]

			if tmp.Port == 0 {
				tmp.Port = 20160
			}
			if tmp.StatusPort == 0 {
				tmp.StatusPort = 20180
			}
			tikvs = append(tikvs, tmp)
		}
	}
	return tidbs, pds, tikvs
}
