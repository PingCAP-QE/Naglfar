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

const (
	TestResourceRequestPending TestResourceRequestState = "pending"
	TestResourceRequestReady   TestResourceRequestState = "ready"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ResourceRequestItem struct {
	// Specifies the name of a node.
	// For example, we can use n1, n2, n3 to refer to different resources
	Name string           `json:"name"`
	Spec TestResourceSpec `json:"spec"`
}

// TestResourceRequestSpec defines the desired state of TestResourceRequest
type TestResourceRequestSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	Items []*ResourceRequestItem `json:"items,omitempty"`
}

// +kubebuilder:validation:Enum=pending;ready
type TestResourceRequestState string

// TestResourceRequestStatus defines the observed state of TestResourceRequest
type TestResourceRequestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// default Pending
	State TestResourceRequestState `json:"state"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TestResourceRequest is the Schema for the testresourcerequests API
type TestResourceRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestResourceRequestSpec   `json:"spec,omitempty"`
	Status TestResourceRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TestResourceRequestList contains a list of TestResourceRequest
type TestResourceRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestResourceRequest `json:"items"`
}

func (r *TestResourceRequest) ConstructTestResource(idx int) *TestResource {
	return &TestResource{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        r.Spec.Items[idx].Name,
			Namespace:   r.Namespace,
		},
		Spec: *r.Spec.Items[idx].Spec.DeepCopy(),
	}
}

func init() {
	SchemeBuilder.Register(&TestResourceRequest{}, &TestResourceRequestList{})
}
