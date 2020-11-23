/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	TestWorkloadStatePending TestWorkloadState = "pending"
	TestWorkloadStateRunning                   = "running"
	TestWorkloadStateFinish                    = "finish"
)

type ClusterTopologyRef struct {
	Name string `json:"name"`
	// +optional
	AliasName string `json:"aliasName,omitempty"`
}

type ResourceRequestRef struct {
	Name string `json:"name"`
	Node string `json:"node"`
}

type DockerContainerSpec struct {
	ResourceRequest ResourceRequestRef `json:"resourceRequest"`
	Image           string             `json:"image"`
	// +optional
	ImagePullPolicy PullImagePolicy `json:"imagePullPolicy,omitempty"`
	// +optional
	Command []string `json:"command,omitempty"`
}

type TestWorkloadItemSpec struct {
	Name string `json:"name"`

	// +optional
	DockerContainer *DockerContainerSpec `json:"dockerContainer,omitempty"`
}

// TestWorkloadSpec defines the desired state of TestWorkload
type TestWorkloadSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ClusterTopologiesRefs references
	// +optional
	ClusterTopologiesRefs []ClusterTopologyRef `json:"clusterTopologies,omitempty"`

	// Workloads specifies workload templates
	Workloads []TestWorkloadItemSpec `json:"workloads"`

	// +optional
	TeardownTestClusterTopology []string `json:"teardownTestClusterTopology,omitempty"`
}

// +kubebuilder:validation:Enum=pending;running;finish;fail
type TestWorkloadState string

type TestWorkloadResult struct {
	// plain text format
	// +optional
	PlainText string `json:"plainText,omitempty"`
}

// TestWorkloadStatus defines the observed state of TestWorkload
type TestWorkloadStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// default pending
	// +optional
	State TestWorkloadState `json:"state"`

	// Save the results passed through
	// +optional
	Results map[string]TestWorkloadResult `json:"results,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName="tw"
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="the state of workload"

// TestWorkload is the Schema for the testworkloads API
type TestWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestWorkloadSpec   `json:"spec,omitempty"`
	Status TestWorkloadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TestWorkloadList contains a list of TestWorkload
type TestWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TestWorkload{}, &TestWorkloadList{})
}
