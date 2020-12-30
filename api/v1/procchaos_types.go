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

	"github.com/PingCAP-QE/Naglfar/pkg/util"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ProcChaosTask struct {
	Pattern string `json:"pattern"`

	Nodes []string `json:"nodes"`

	// KillAll means kill all process matching pattern
	// +optional
	KillAll bool `json:"killAll,omitempty"`

	// +optional
	Period util.Duration `json:"period,omitempty"`
}

type ProcChaosState struct {
	// +optional
	KilledTime util.Time `json:"killeTime,omitempty"`

	// +optional
	KilledNode string `json:"killedNode,omitempty"`
}

// ProcChaosSpec defines the desired state of ProcChaos
type ProcChaosSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Request is the name of TestResourceRequest
	Request string `json:"request"`

	Tasks []*ProcChaosTask `json:"tasks"`
}

// ProcChaosStatus defines the observed state of ProcChaos
type ProcChaosStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	States []*ProcChaosState `json:"states"`
}

// +kubebuilder:object:root=true

// ProcChaos is the Schema for the procchaos API
// +kubebuilder:subresource:status
type ProcChaos struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProcChaosSpec   `json:"spec,omitempty"`
	Status ProcChaosStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProcChaosList contains a list of ProcChaos
type ProcChaosList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProcChaos `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProcChaos{}, &ProcChaosList{})
}
