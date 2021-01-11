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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var testresourcerequestlog = logf.Log.WithName("testresourcerequest-resource")

func (r *TestResourceRequest) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-naglfar-pingcap-com-v1-testresourcerequest,mutating=true,failurePolicy=fail,groups=naglfar.pingcap.com,resources=testresourcerequests,verbs=create;update,versions=v1,name=mtestresourcerequest.kb.io

var _ webhook.Defaulter = &TestResourceRequest{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *TestResourceRequest) Default() {
	testresourcerequestlog.Info("default", "name", r.Name)
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-naglfar-pingcap-com-v1-testresourcerequest,mutating=false,failurePolicy=fail,groups=naglfar.pingcap.com,resources=testresourcerequests,versions=v1,name=vtestresourcerequest.kb.io

var _ webhook.Validator = &TestResourceRequest{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *TestResourceRequest) ValidateCreate() error {
	testresourcerequestlog.Info("validate create", "name", r.Name)
	for _, resourceSpec := range r.Spec.Items {
		if err := resourceSpec.Spec.ValidateCreate(); err != nil {
			return err
		}
	}
	// check items
	err := r.checkItems()
	return err
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *TestResourceRequest) ValidateUpdate(old runtime.Object) error {
	testresourcerequestlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

func (r *TestResourceRequest) checkItems() error {
	machinesLogicNames := make(map[string]struct{})
	for _, item := range r.Spec.Machines {
		machinesLogicNames[item.Name] = struct{}{}
	}
	for _, item := range r.Spec.Items {
		if item.Spec.Machine != "" {
			if _, exist := machinesLogicNames[item.Spec.Machine]; !exist {
				return fmt.Errorf("no exist machine item: %s", item.Spec.Machine)
			}
		}
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *TestResourceRequest) ValidateDelete() error {
	testresourcerequestlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
