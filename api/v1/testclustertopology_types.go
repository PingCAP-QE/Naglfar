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

package v1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	ClusterTopologyStatePending  ClusterTopologyState = "pending"
	ClusterTopologyStateReady    ClusterTopologyState = "ready"
	ClusterTopologyStateUpdating ClusterTopologyState = "updating"
)

// TODO: add a deploy version spec: clusterName, base version, component versions(for PR and self build version) etc.
type TiDBClusterVersion struct {
	Version string `json:"version"`
	// +optional
	TiDBDownloadURL string `json:"tidbDownloadURL,omitempty"`
	// +optional
	TiKVDownloadURL string `json:"tikvDownloadURL,omitempty"`
	// +optional
	PDDownloadUrl string `json:"pdDownloadURL,omitempty"`
}

type ServerConfigs struct {
	// +optional
	TiDB string `json:"tidb,omitempty"`
	// +optional
	TiKV string `json:"tikv,omitempty"`
	// +optional
	PD string `json:"pd,omitempty"`
}

type TiDBSpec struct {
	Host string `json:"host"`

	// +optional
	Port int `json:"port,omitempty"`
	// +optional
	StatusPort int `json:"statusPort,omitempty" yaml:"status_port,omitempty"`

	DeployDir string `json:"deployDir" yaml:"deploy_dir"`

	// +optional
	LogDir string `json:"logDir,omitempty" yaml:"log_dir,omitempty"`
}

type PDSpec struct {
	Host string `json:"host"`

	// +optional
	ClientPort int `json:"clientPort,omitempty" yaml:"client_port,omitempty"`
	// +optional
	PeerPort int `json:"peerPort,omitempty" yaml:"peer_port,omitempty"`

	DeployDir string `json:"deployDir" yaml:"deploy_dir"`
	DataDir   string `json:"dataDir,omitempty" yaml:"data_dir,omitempty"`

	// +optional
	LogDir string `json:"logDir,omitempty" yaml:"log_dir,omitempty"`
}

type TiKVSpec struct {
	Host string `json:"host"`

	// +optional
	Port int `json:"port,omitempty"`
	// +optional
	StatusPort int `json:"statusPort,omitempty" yaml:"status_port,omitempty"`

	DeployDir string `json:"deployDir" yaml:"deploy_dir"`
	DataDir   string `json:"dataDir" yaml:"data_dir"`

	// +optional
	LogDir string `json:"logDir,omitempty" yaml:"log_dir,omitempty"`
}

type PumpSpec struct {
	Host    string `json:"host"`
	SSHPort int    `json:"sshPort,omitempty"`
	// +optional
	Port int `json:"port,omitempty"`
	// +optional
	DeployDir string `json:"deployDir"`
	DataDir   string `json:"dataDir"`
	Config    string `json:"config,omitempty"`
}

type DrainerSpec struct {
	Host   string `json:"host"`
	Config string `json:"config"`

	// +optional
	SSHPort int `json:"sshPort,omitempty"`
	// +optional
	Port int `json:"port,omitempty"`
	// +optional
	CommitTS int64 `json:"commitTS,omitempty"`
	// +optional
	DeployDir string `json:"deployDir"`
	DataDir   string `json:"dataDir"`
}

type PrometheusSpec struct {
	Host string `json:"host"`
	// +optional
	Port      int    `json:"port,omitempty"`
	DeployDir string `json:"deployDir"`
	DataDir   string `json:"dataDir"`
	// +optional
	LogDir string `json:"logDir,omitempty"`
}

type TiFlashSpec struct {
	Host string `json:"host"`
	// +optional
	TCPPort int `json:"tcpPort,omitempty"`
	// +optional
	HTTPPort int `json:"httpPort,omitempty"`
	// +optional
	ServicePort int `json:"servicePort,omitempty"`
	// +optional
	ProxyPort int `json:"proxyPort,omitempty"`
	// +optional
	ProxyStatusPort int `json:"proxyStatusPort,omitempty"`
	// +optional
	MetricsPort int `json:"metricsPort,omitempty"`

	DeployDir string `json:"deployDir"`
	DataDir   string `json:"dataDir"`

	// +optional
	Config string `json:"config,omitempty"`
	// +optional
	LearnerConfig string `json:"learnerConfig,omitempty"`
	// +optional
	LogDir string `json:"logDir,omitempty"`
}

type GrafanaSpec struct {
	Host string `json:"host"`
	// +optional
	Port      int    `json:"port,omitempty"`
	DeployDir string `json:"deployDir"`
}

type Global struct {
	DeployDir string `json:"deployDir" yaml:"deploy_dir"`
	DataDir   string `json:"dataDir" yaml:"data_dir"`
}

type TiDBCluster struct {
	Version TiDBClusterVersion `json:"version"`

	// +optional
	Global *Global `json:"global,omitempty"`
	// +optional
	ServerConfigs ServerConfigs `json:"serverConfigs,omitempty"`

	// Control machine host
	Control string `json:"control"`

	// TiDB machine hosts
	// +optional
	TiDB []TiDBSpec `json:"tidb,omitempty"`

	// TiKV machine hosts
	// +optional
	TiKV []TiKVSpec `json:"tikv,omitempty"`

	// PD machine hosts
	// +optional
	PD []PDSpec `json:"pd"`

	// +optional
	Pump []PumpSpec `json:"pump,omitempty"`

	// +optional
	Drainer []DrainerSpec `json:"drainer,omitempty"`

	// +optional
	Monitor []PrometheusSpec `json:"monitor,omitempty"`

	// +optional
	Grafana []GrafanaSpec `json:"grafana,omitempty"`

	// +optional
	TiFlash []TiFlashSpec `json:"tiflash,omitempty"`
}

func (c *TiDBCluster) AllHosts() map[string]struct{} {
	components := []string{TiDBField, TiKVField, PDField, MonitorField, GrafanaField, TiFlashField}
	result := map[string]struct{}{
		c.Control: {},
	}

	val := reflect.ValueOf(*c)
	for i := 0; i < val.Type().NumField(); i++ {
		if checkIn(components, val.Type().Field(i).Name) {
			field := val.Field(i)
			for j := 0; j < field.Len(); j++ {
				result[field.Index(j).FieldByName("Host").String()] = struct{}{}
			}
		}
	}
	return result
}

type JobManagerSpec struct {
	Host string `json:"host"`
	// +optional
	WebPort int `json:"webPort"`
	// +optional
	Config string `json:"config,omitempty"`
}

type TaskManagerSpec struct {
	Host string `json:"host"`
	// +optional
	Config string `json:"config,omitempty"`
}

type FlinkCluster struct {
	Version string `json:"version"`

	JobManager JobManagerSpec `json:"jobManager,omitempty"`
	// +optional
	TaskManager []TaskManagerSpec `json:"taskManager,omitempty"`
}

func (c *FlinkCluster) AllHosts() map[string]struct{} {
	result := map[string]struct{}{
		c.JobManager.Host: {},
	}
	for i := 0; i < len(c.TaskManager); i++ {
		result[c.TaskManager[i].Host] = struct{}{}
	}
	return result
}

type DMServerConfigs struct {
	// +optional
	Master string `json:"master,omitempty"`
	// +optional
	Worker string `json:"worker,omitempty"`
}

type AlertManagerSpec struct {
	Host string `json:"host"`

	// +optional
	WebPort int `json:"webPort,omitempty"`
	// +optional
	ClusterPort int `json:"clusterPort,omitempty"`

	DeployDir string `json:"deployDir"`
	DataDir   string `json:"dataDir"`

	// +optional
	LogDir string `json:"logDir,omitempty"`
}

type MasterSpec struct {
	Host string `json:"host"`

	// +optional
	Port int `json:"Port,omitempty" yaml:"port,omitempty"`
	// +optional
	PeerPort int `json:"peerPort,omitempty" yaml:"peer_port,omitempty"`

	DeployDir string `json:"deployDir" yaml:"deploy_dir"`
	DataDir   string `json:"dataDir" yaml:"data_dir"`

	// +optional
	Config string `json:"config,omitempty"`
	// +optional
	LogDir string `json:"logDir,omitempty"`
}

type WorkerSpec struct {
	Host string `json:"host"`
	// +optional
	Port int `json:"port,omitempty"`

	DeployDir string `json:"deployDir" yaml:"deploy_dir"`
	DataDir   string `json:"dataDir" yaml:"data_dir"`

	// +optional
	LogDir string `json:"logDir,omitempty"`
	// +optional
	Config string `json:"config,omitempty"`
}

type DMCluster struct {
	Version string `json:"version,omitempty"`

	// +optional
	Global *Global `json:"global,omitempty"`

	// +optional
	ServerConfigs DMServerConfigs `json:"serverConfigs,omitempty"`

	// Control machine host
	Control string `json:"control"`

	// +optional
	Master []MasterSpec `json:"master,omitempty"`

	// +optional
	Worker []WorkerSpec `json:"worker,omitempty"`

	// +optional
	AlertManager []AlertManagerSpec `json:"alertManager,omitempty"`

	// +optional
	Monitor []PrometheusSpec `json:"monitor,omitempty"`

	// +optional
	Grafana []GrafanaSpec `json:"grafana,omitempty"`
}

func (c *DMCluster) AllHosts() map[string]struct{} {
	components := []string{MasterField, WorkerField, MonitorField, GrafanaField, AlertManagerField}
	result := map[string]struct{}{
		c.Control: {},
	}

	val := reflect.ValueOf(*c)
	for i := 0; i < val.Type().NumField(); i++ {
		if checkIn(components, val.Type().Field(i).Name) {
			field := val.Field(i)
			for j := 0; j < field.Len(); j++ {
				result[field.Index(j).FieldByName("Host").String()] = struct{}{}
			}
		}
	}
	return result
}

// TestClusterTopologySpec defines the desired state of TestClusterTopology
type TestClusterTopologySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ResourceRequest cannot be empty if the tidbCluster field is set
	// +optional
	ResourceRequest string `json:"resourceRequest,omitempty"`

	// +optional
	TiDBCluster *TiDBCluster `json:"tidbCluster,omitempty"`

	// +optional
	FlinkCluster *FlinkCluster `json:"flinkCluster,omitempty"`

	// +optional
	DMCluster *DMCluster `json:"dmCluster,omitempty"`
}

type TiDBClusterInfo struct {
	//optional
	IsScale bool `json:"isScale,omitempty"`

	// tikvs in pendingOffline state, like [172.16.0.1:20160]
	// +optional
	PendingOfflineList []string `json:"pendingOfflineList,omitempty"`

	// tikvs in offline state, like [172.16.0.1:20160]
	// +optional
	OfflineList []string `json:"offlineList,omitempty"`
}

// +kubebuilder:validation:Enum=pending;ready;updating
type ClusterTopologyState string

// TestClusterTopologyStatus defines the observed state of TestClusterTopology
type TestClusterTopologyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// PreTiDBCluster save tidb cluster configuration before update and scale-in/out,
	// It is used to compare with the new configuration to see if the TestClusterTopology
	// has been changed and what has been changed
	PreTiDBCluster *TiDBCluster `json:"preTiDBCluster,omitempty"`

	// Info provide some information to help users understand the internal state of the cluster
	// +optional
	TiDBClusterInfo TiDBClusterInfo `json:"tiDBCluster,omitempty"`
	// default Pending
	State ClusterTopologyState `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName="tct"
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="the state of cluster topology"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TestClusterTopology is the Schema for the testclustertopologies API
type TestClusterTopology struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestClusterTopologySpec   `json:"spec,omitempty"`
	Status TestClusterTopologyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TestClusterTopologyList contains a list of TestClusterTopology
type TestClusterTopologyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestClusterTopology `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TestClusterTopology{}, &TestClusterTopologyList{})
}
