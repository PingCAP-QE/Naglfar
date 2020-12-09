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

type WorkflowStepItem struct {
	Name string `json:"name"`

	// +optional
	TestResourceRequest *TestResourceSpec `json:"testResourceRequest,omitempty"`

	// +optional
	TestClusterTopology *TestClusterTopology `json:"testClusterTopology,omitempty"`

	// +optional
	TestWorkload *TestWorkload `json:"testWorkload,omitempty"`
}

// TestWorkflowSpec defines the desired state of TestWorkflow
type TestWorkflowSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	Steps []WorkflowStepItem `json:"steps,omitempty"`
}

// TestWorkflowStatus defines the observed state of TestWorkflow
type TestWorkflowStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// TestWorkflow is the Schema for the testworkflows API
type TestWorkflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestWorkflowSpec   `json:"spec,omitempty"`
	Status TestWorkflowStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TestWorkflowList contains a list of TestWorkflow
type TestWorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestWorkflow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TestWorkflow{}, &TestWorkflowList{})
}
