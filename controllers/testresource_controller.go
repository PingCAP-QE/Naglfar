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

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ref "k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
)

// TestResourceReconciler reconciles a TestResource object
type TestResourceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=machines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=machines/status,verbs=get;update;patch

func (r *TestResourceReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {
	ctx := context.Background()
	log := r.Log.WithValues("testresource", req.NamespacedName)

	resource := new(naglfarv1.TestResource)

	if err = r.Get(ctx, req.NamespacedName, resource); err != nil {
		log.Error(err, "unable to fetch TestResource")

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		err = client.IgnoreNotFound(err)
		return
	}

	log = r.Log.WithValues("state", resource.Status.State)

	switch resource.Status.State {
	case naglfarv1.ResourcePending:
		return r.reconcileStatePending(ctx, log, resource)
	}

	return
}

func (r *TestResourceReconciler) reconcileStatePending(ctx context.Context, log logr.Logger, resource *naglfarv1.TestResource) (result ctrl.Result, err error) {
	resourceOverflow := func(machine *naglfarv1.Machine) (overflow bool, err error) {
		rest := machine.Status.Available.DeepCopy()

		for _, refer := range machine.Status.TestResources {
			resource := new(naglfarv1.TestResource)

			name := types.NamespacedName{Namespace: refer.Namespace, Name: refer.Name}

			log := r.Log.WithValues("testresource", name)

			if err = r.Get(ctx, name, resource); err != nil {
				log.Error(err, "unable to fetch TestResource")
				if apierrors.IsNotFound(err) {
					// machine updated, ignored
					err = nil
					continue
				}

				return
			}

			// TODO: save resource after machine saving

			rest.CPUPercent = rest.CPUPercent - resource.Spec.CPUPercent
			rest.Memory = rest.Memory.Sub(resource.Spec.Memory)

			if len(resource.Status.DiskStat) != 0 {
				for _, stat := range resource.Status.DiskStat {
					if _, ok := rest.Disks[stat.Device]; !ok {
						err = fmt.Errorf("device %s unavialable on machine %s", stat.Device, machine.Name)
						return
					}

					delete(rest.Disks, stat.Device)
				}
				continue
			}

			for name, disk := range resource.Spec.Disks {
				for device, diskResource := range rest.Disks {
					if disk.Kind == diskResource.Kind &&
						disk.Size.Unwrap() <= diskResource.Size.Unwrap() {
						resource.Status.DiskStat[name] = naglfarv1.DiskStatus{
							Kind:      disk.Kind,
							Size:      diskResource.Size,
							Device:    device,
							MountPath: disk.MountPath,
						}
					}
				}
			}

			if len(resource.Spec.Disks) != len(resource.Status.DiskStat) {
				overflow = true
				return
			}
		}

		overflow = rest.Memory.Unwrap() <= 0 || rest.CPUPercent <= 0

		return
	}

	tryRequestResource := func(machine *naglfarv1.Machine) (overflow bool, result ctrl.Result, err error) {
		resourceRef, err := ref.GetReference(r.Scheme, resource)
		if err != nil {
			log.Error(err, "unable to make reference to resource", "resource", resource)
			return
		}

		machine.Status.TestResources = append(machine.Status.TestResources, *resourceRef)

		if overflow, err = resourceOverflow(machine); overflow || err != nil {
			return
		}

		if err = r.Status().Update(ctx, machine); err != nil {
			if apierrors.IsConflict(err) {
				err = nil
				result.Requeue = true
			}
			return
		}

		return
	}

	if resource.Spec.TestMachineResource != "" {
		overflow := false
		machine := new(naglfarv1.Machine)
		if err = r.Get(ctx, types.NamespacedName{Namespace: "default", Name: resource.Spec.TestMachineResource}, machine); err != nil {
			log.Error(err, fmt.Sprintf("unable to fetch Machine %s", resource.Spec.TestMachineResource))

			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			err = client.IgnoreNotFound(err)
			return
		}

		overflow, result, err = tryRequestResource(machine)

		if overflow {
			err = fmt.Errorf("resource of machine(%s) ran out of", machine.Name)
		}

		return
	}

	return ctrl.Result{}, nil
}

func (r *TestResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naglfarv1.TestResource{}).
		Complete(r)
}
