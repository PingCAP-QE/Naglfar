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
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var procchaoslog = logf.Log.WithName("procchaos-resource")

func (r *ProcChaos) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-naglfar-pingcap-com-v1-procchaos,mutating=false,failurePolicy=fail,groups=naglfar.pingcap.com,resources=procchaos,versions=v1,name=vprocchaos.kb.io

var _ webhook.Validator = &ProcChaos{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ProcChaos) ValidateCreate() error {
	procchaoslog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	if len(r.Spec.Tasks) == 0 {
		return fmt.Errorf("tasks cannot be empty")
	}

	for _, task := range r.Spec.Tasks {
		if _, err := task.Period.Parse(); err != nil {
			return err
		}
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ProcChaos) ValidateUpdate(old runtime.Object) error {
	procchaoslog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	if !reflect.DeepEqual(r.Spec.Tasks, old.(*ProcChaos).Spec.Tasks) {
		return fmt.Errorf("cannot update tasks")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ProcChaos) ValidateDelete() error {
	procchaoslog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
