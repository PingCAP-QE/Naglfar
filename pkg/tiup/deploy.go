package tiup

import (
	"bytes"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

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

const (
	// Our controller node and worker nodes share the same insecure_key path
	insecureKeyPath = "/root/insecure_key"
	ContainerImage  = "docker.io/mahjonp/base-image:latest"
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

// If dryRun is true, host uses the resource's name
func BuildSpecification(ctf *naglfarv1.TestClusterTopologySpec, trs []*naglfarv1.TestResource, dryRun bool) (
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
		spec.Grafana = append(spec.Grafana, tiupSpec.GrafanaSpec{
			Host:            hostName(item.Host, node.ClusterIP),
			SSHPort:         22,
			Port:            item.Port,
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
	specification, control, err := BuildSpecification(ctf, trs, false)
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
	if err := c.deployCluster(log, clusterName, version.Version); IgnoreClusterDuplicated(err) != nil {
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

func (c *ClusterManager) UninstallCluster(clusterName string) error {
	client, err := sshUtil.NewSSHClient("root", insecureKeyPath, c.control.HostIP, c.control.SSHPort)
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
			return ErrClusterNotExist{clusterName: clusterName}
		}
		return fmt.Errorf("cannot run remote command `%s`: %s", cmd, err)
	}
	return nil
}

func (c *ClusterManager) writeTopologyFileOnControl(out []byte) error {
	clientConfig, _ := auth.PrivateKey("root", insecureKeyPath, ssh.InsecureIgnoreHostKey())
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
	client, err := sshUtil.NewSSHClient("root", insecureKeyPath, c.control.HostIP, c.control.SSHPort)
	if err != nil {
		return err
	}
	defer client.Close()
	cmd := fmt.Sprintf("/root/.tiup/bin/tiup cluster deploy -y %s %s /root/topology.yaml -i %s", clusterName, version, insecureKeyPath)
	stdStr, errStr, err := client.RunCommand(cmd)
	if err != nil {
		log.Error(err, "run command on remote failed",
			"host", fmt.Sprintf("%s@%s:%d", "root", c.control.HostIP, c.control.SSHPort),
			"command", cmd,
			"stdout", stdStr,
			"stderr", errStr)
		if strings.Contains(errStr, "specify another cluster name") {
			return ErrClusterDuplicated{clusterName: clusterName}
		}
		return fmt.Errorf("deploy cluster failed(%s): %s", err, errStr)
	}
	return nil
}

func (c *ClusterManager) startCluster(clusterName string) error {
	client, err := sshUtil.NewSSHClient("root", insecureKeyPath, c.control.HostIP, c.control.SSHPort)
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

func (c *ClusterManager) shouldPatch(version naglfarv1.TiDBClusterVersion) bool {
	return len(version.TiDBDownloadURL) != 0 || len(version.PDDownloadUrl) != 0 || len(version.TiKVDownloadURL) != 0
}

func (c *ClusterManager) patch(clusterName string, version naglfarv1.TiDBClusterVersion) error {
	type component struct {
		componentName string
		downloadURL   string
	}
	client, err := sshUtil.NewSSHClient("root", insecureKeyPath, c.control.HostIP, c.control.SSHPort)
	if err != nil {
		return err
	}
	defer client.Close()
	commands := []string{"set -ex", "rm -rf components && mkdir components"}
	var patchComponents []component
	var patchComponentNames []string
	if len(version.TiDBDownloadURL) != 0 {
		patchComponents = append(patchComponents, component{
			componentName: "tidb-server",
			downloadURL:   version.TiDBDownloadURL,
		})
		patchComponentNames = append(patchComponentNames, "tidb-server")
	}
	if len(version.TiKVDownloadURL) != 0 {
		patchComponents = append(patchComponents, component{
			componentName: "tikv-server",
			downloadURL:   version.TiKVDownloadURL,
		})
		patchComponentNames = append(patchComponentNames, "tikv-server")
	}
	if len(version.PDDownloadUrl) != 0 {
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
		commands = append(commands,
			fmt.Sprintf(`curl -O %s`, downloadURL))
		commands = append(commands, c.GenUnzipCommand(path.Base(u.Path), "components"))
		commands = append(commands, fmt.Sprintf("rm -rf components/%s", component.componentName))
		commands = append(commands, fmt.Sprintf("mv components/bin/%s components/%s", component.componentName, component.componentName))
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
