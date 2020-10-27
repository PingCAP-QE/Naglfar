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

type ClusterTopologyRef struct {
	Name      string `json:"name"`
	AliasName string `json:"aliasName"`
}

type ResourceRequestRef struct {
	Name string `json:"name"`
	Node string `json:"node"`
}

type DockerContainer struct {
	Name            string             `json:"name"`
	ResourceRequest ResourceRequestRef `json:"resourceRequest"`
	Image           string             `json:"image"`
	// +optional
	Command []string `json:"command,omitempty"`
}

type TestWorkloadItem struct {
	Name string `json:"name"`

	//JobTemplate *batchv1beta1.JobTemplateSpec `json:"jobTemplate,omitempty"`

	// +optional
	DockerContainer *DockerContainer `json:"dockerContainer,omitempty"`
}

// TestWorkloadSpec defines the desired state of TestWorkload
type TestWorkloadSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ClusterTopologiesRefs references
	// +optional
	ClusterTopologiesRefs []ClusterTopologyRef `json:"clusterTopologies,omitempty"`

	// Workloads specifies workload templates
	Workloads []TestWorkloadItem `json:"workloads"`
}

// TestWorkloadStatus defines the observed state of TestWorkload
type TestWorkloadStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

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
