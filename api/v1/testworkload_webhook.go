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

	"github.com/docker/docker/api/types/reference"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var testworkloadlog = logf.Log.WithName("testworkload-resource")

func (r *TestWorkload) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-naglfar-pingcap-com-v1-testworkload,mutating=true,failurePolicy=fail,groups=naglfar.pingcap.com,resources=testworkloads,verbs=create;update,versions=v1,name=mtestworkload.kb.io

var _ webhook.Defaulter = &TestWorkload{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *TestWorkload) Default() {
	testworkloadlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-naglfar-pingcap-com-v1-testworkload,mutating=false,failurePolicy=fail,groups=naglfar.pingcap.com,resources=testworkloads,versions=v1,name=vtestworkload.kb.io

var _ webhook.Validator = &TestWorkload{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *TestWorkload) ValidateCreate() error {
	testworkloadlog.Info("validate create", "name", r.Name)
	if err := r.validateWorkload(); err != nil {
		return fmt.Errorf("validate workload error: %s", err)
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *TestWorkload) ValidateUpdate(old runtime.Object) error {
	testworkloadlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *TestWorkload) ValidateDelete() error {
	testworkloadlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *TestWorkload) validateWorkload() error {
	for _, workload := range r.Spec.Workloads {
		if err := validateImage(workload.DockerContainer.Image); err != nil {
			return fmt.Errorf("validate image %s error: %s", workload.DockerContainer.Image, err)
		}
	}
	return nil
}

func validateImage(image string) error {
	_, _, err := reference.Parse(image)
	if err != nil {
		return err
	}
	return nil
}
