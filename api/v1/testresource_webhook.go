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
	"path"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	mountPrefix = "/mnt"
)

// log is for logging in this package.
var testresourcelog = logf.Log.WithName("testresource-resource")

func (r *TestResource) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-naglfar-pingcap-com-v1-testresource,mutating=true,failurePolicy=fail,groups=naglfar.pingcap.com,resources=testresources,verbs=create;update,versions=v1,name=mtestresource.kb.io

var _ webhook.Defaulter = &TestResource{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *TestResource) Default() {
	testresourcelog.Info("default", "name", r.Name)

	for name, disk := range r.Spec.Disks {
		if disk.MountPath == "" {
			disk.MountPath = path.Join(mountPrefix, name)
		}

		if disk.Size == "" {
			disk.Size = disk.Size.Zero()
		}

		r.Spec.Disks[name] = disk
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-naglfar-pingcap-com-v1-testresource,mutating=false,failurePolicy=fail,groups=naglfar.pingcap.com,resources=testresources,versions=v1,name=vtestresource.kb.io

var _ webhook.Validator = &TestResource{}

func (r *TestResourceSpec) ValidateCreate() error {
	if _, err := r.Memory.ToSize(); err != nil {
		return fmt.Errorf("invalid memory size: %s", err.Error())
	}

	for _, disk := range r.Disks {
		if _, err := disk.Size.ToSize(); disk.Size != "" && err != nil {
			return fmt.Errorf("invalid disk size: %s", err.Error())
		}
	}

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *TestResource) ValidateCreate() error {
	testresourcelog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return r.Spec.ValidateCreate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *TestResource) ValidateUpdate(old runtime.Object) error {
	testresourcelog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *TestResource) ValidateDelete() error {
	testresourcelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
