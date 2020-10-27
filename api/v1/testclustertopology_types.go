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

type TiDBCluster struct {

	// +optional
	TiDB []string `json:"tidb"`

	// +optional
	TiKV []string `json:"tikv"`

	// +optional
	PD []string `json:"pd"`

	// +optional
	Monitor []string `json:"monitor"`
}

// TestClusterTopologySpec defines the desired state of TestClusterTopology
type TestClusterTopologySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Enum=tidb-cluster
	Type string `json:"type"`

	// +optional
	ResourceRequest string `json:"resourceRequest,omitempty"`

	// +optional
	TiDBCluster *TiDBCluster `json:"tidbCluster,omitempty"`
}

// TestClusterTopologyStatus defines the observed state of TestClusterTopology
type TestClusterTopologyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Enum=Pending;Ready
	// default Pending
	State string `json:"state,omitempty"`
}

// +kubebuilder:object:root=true

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
