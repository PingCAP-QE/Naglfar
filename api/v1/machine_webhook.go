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
	"fmt"
	"time"

	"github.com/docker/go-units"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var machinelog = logf.Log.WithName("machine-resource")

func (r *Machine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-naglfar-pingcap-com-v1-machine,mutating=true,failurePolicy=fail,groups=naglfar.pingcap.com,resources=machines,verbs=create;update,versions=v1,name=mmachine.kb.io

var _ webhook.Defaulter = &Machine{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Machine) Default() {
	machinelog.Info("default", "name", r.Name)

	if r.Spec.SSHPort == 0 {
		r.Spec.SSHPort = 22
	}

	if r.Spec.DockerPort == 0 {
		if r.Spec.DockerTLS {
			r.Spec.DockerPort = 2376
		} else {
			r.Spec.DockerPort = 2375
		}
	}

	if r.Spec.Timeout == "" {
		r.Spec.Timeout = HumanDuration(10 * time.Second)
	}

	if r.Spec.Reserve == nil {
		r.Spec.Reserve = new(ReserveResources)
	}

	if r.Spec.Reserve.Cores == 0 {
		r.Spec.Reserve.Cores = 1
	}

	if r.Spec.Reserve.Memory == "" {
		r.Spec.Reserve.Memory = Size(1 * units.GiB)
	}
	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-naglfar-pingcap-com-v1-machine,mutating=false,failurePolicy=fail,groups=naglfar.pingcap.com,resources=machines,versions=v1,name=vmachine.kb.io

var _ webhook.Validator = &Machine{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Machine) ValidateCreate() error {
	machinelog.Info("validate create", "name", r.Name)

	if r.Spec.SSHPort < 0 {
		return fmt.Errorf("invalid port %d", r.Spec.SSHPort)
	}

	if r.Spec.DockerPort < 0 {
		return fmt.Errorf("invalid port %d", r.Spec.DockerPort)
	}

	if reserve := r.Spec.Reserve; reserve != nil {
		if reserve.Cores < 0 {
			return fmt.Errorf("invalid cpu cores %d", reserve.Cores)
		}

		if _, err := reserve.Memory.ToSize(); reserve.Memory != "" && err != nil {
			return fmt.Errorf("invalid memory size: %s", err.Error())
		}
	}

	if _, err := r.Spec.Timeout.Parse(); err != nil {
		return fmt.Errorf("fail to parse timeout(%s): %s", r.Spec.Timeout, err.Error())
	}

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Machine) ValidateUpdate(old runtime.Object) error {
	machinelog.Info("validate update", "name", r.Name)

	if _, err := r.Status.Info.Memory.ToSize(); err != nil {
		return fmt.Errorf("invalid memory size: %s", err.Error())
	}

	for _, device := range r.Status.Info.StorageDevices {
		if _, err := device.Total.ToSize(); err != nil {
			return fmt.Errorf("invalid storage size: %s", err.Error())
		}

		if _, err := device.Used.ToSize(); err != nil {
			return fmt.Errorf("invalid storage size: %s", err.Error())
		}
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Machine) ValidateDelete() error {
	machinelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
