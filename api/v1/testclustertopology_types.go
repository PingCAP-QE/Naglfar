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
	StatusPort int `json:"statusPort,omitempty"`

	DeployDir string `json:"deployDir"`
	// +optional
	DataDir string `json:"dataDir,omitempty"`
	// +optional
	LogDir string `json:"logDir,omitempty"`
}

type PDSpec struct {
	Host string `json:"host"`

	// +optional
	ClientPort int `json:"clientPort,omitempty"`
	// +optional
	PeerPort int `json:"peerPort,omitempty"`

	DeployDir string `json:"deployDir"`
	DataDir   string `json:"dataDir"`
	// +optional
	LogDir string `json:"logDir,omitempty"`
}

type TiKVSpec struct {
	Host string `json:"host"`
	// +optional
	Port int `json:"port,omitempty"`
	// +optional
	StatusPort int    `json:"statusPort,omitempty"`
	DeployDir  string `json:"deployDir"`
	DataDir    string `json:"dataDir"`
	// +optional
	LogDir string `json:"logDir,omitempty"`
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

type GrafanaSpec struct {
	Host string `json:"host"`
	// +optional
	Port      int    `json:"port,omitempty"`
	DeployDir string `json:"deployDir"`
}

type TiDBCluster struct {
	Version TiDBClusterVersion `json:"version"`

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
}

func (c *TiDBCluster) AllHosts() map[string]struct{} {
	result := map[string]struct{}{
		c.Control: {},
	}
	for _, item := range c.TiDB {
		result[item.Host] = struct{}{}
	}
	for _, item := range c.TiKV {
		result[item.Host] = struct{}{}
	}
	for _, item := range c.PD {
		result[item.Host] = struct{}{}
	}
	for _, item := range c.Monitor {
		result[item.Host] = struct{}{}
	}
	for _, item := range c.Grafana {
		result[item.Host] = struct{}{}
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
}

// +kubebuilder:validation:Enum=pending;ready;updating
type ClusterTopologyState string

// TestClusterTopologyStatus defines the observed state of TestClusterTopology
type TestClusterTopologyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	PreServerConfigs *ServerConfigs      `json:"preServerConfigs,omitempty"`
	PreVersion       *TiDBClusterVersion `json:"preVersion,omitempty"`
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
