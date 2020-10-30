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

		// maybe resource deleted
		err = client.IgnoreNotFound(err)
		return
	}

	log.Info("resource reconcile", "content", resource)

	switch resource.Status.State {
	case "":
		resource.Status.State = naglfarv1.ResourcePending
		err = r.Status().Update(ctx, resource)
		return
	case naglfarv1.ResourcePending:
		return r.reconcileStatePending(ctx, log, resource)
	}

	return
}

func (r *TestResourceReconciler) resourceOverflow(machine *naglfarv1.Machine, newResource *naglfarv1.TestResource) (overflow bool, requeue bool, err error) {
	rest := machine.Available()

	if rest == nil {
		requeue = true
		return
	}

	ctx := context.Background()

	for _, refer := range machine.Status.TestResources {
		resource := new(naglfarv1.TestResource)

		name := types.NamespacedName{Namespace: refer.Namespace, Name: refer.Name}

		log := r.Log.WithValues("testresource", name)

		if err = r.Get(ctx, name, resource); err != nil {
			log.Error(err, "unable to fetch TestResource")
			if apierrors.IsNotFound(err) {
				// ignore error, resource may deleted
				continue
			}
			return
		}

		rest.CPUPercent -= resource.Spec.CPUPercent
		rest.Memory = rest.Memory.Sub(resource.Spec.Memory)

		if len(resource.Status.DiskStat) != 0 {
			for _, stat := range resource.Status.DiskStat {
				if _, ok := rest.Disks[stat.Device]; !ok {
					log.Error(fmt.Errorf("device %s unavialable on machine %s", stat.Device, machine.Name), "data maybe outdated")
					requeue = true
					return
				}

				delete(rest.Disks, stat.Device)
			}
			continue
		}
	}

	for name, disk := range newResource.Spec.Disks {
		for device, diskResource := range rest.Disks {
			if disk.Kind == diskResource.Kind &&
				disk.Size.Unwrap() <= diskResource.Size.Unwrap() {

				delete(rest.Disks, name)
				newResource.Status.DiskStat[name] = naglfarv1.DiskStatus{
					Kind:      disk.Kind,
					Size:      diskResource.Size,
					Device:    device,
					MountPath: disk.MountPath,
				}

				break
			}
		}
	}

	overflow = len(newResource.Status.DiskStat) < len(newResource.Spec.Disks) ||
		rest.Memory.Unwrap() < newResource.Spec.Memory.Unwrap() ||
		rest.CPUPercent < newResource.Spec.CPUPercent

	return
}

func (r *TestResourceReconciler) tryRequestResource(machine *naglfarv1.Machine, newResource *naglfarv1.TestResource) (overflow bool, requeue bool, err error) {
	log := r.Log.WithValues("testresource", newResource.Name)

	overflow, requeue, err = r.resourceOverflow(machine, newResource)

	if overflow || requeue || err != nil {
		return
	}

	resourceRef, err := ref.GetReference(r.Scheme, newResource)
	if err != nil {
		log.Error(err, "unable to make reference to resource", "resource", newResource)
		return
	}

	machine.Status.TestResources = append(machine.Status.TestResources, *resourceRef)

	if err = r.Status().Update(context.TODO(), machine); err != nil {
		if apierrors.IsConflict(err) {
			err = nil
			requeue = true
		}
		return
	}

	// TODO: may cause resource leak; fix it
	machineRef, err := ref.GetReference(r.Scheme, machine)
	if err != nil {
		log.Error(err, "unable to make reference to machine", "machine", machine)
		return
	}

	newResource.Status.HostMachine = machineRef

	return
}

func (r *TestResourceReconciler) reconcileStatePending(ctx context.Context, log logr.Logger, resource *naglfarv1.TestResource) (result ctrl.Result, err error) {
	overflow := false

	defer func() {
		if resource.Status.State != naglfarv1.ResourcePending {

			// TODO: event this error
			_ = r.Status().Update(ctx, resource)
		}
	}()

	if resource.Spec.TestMachineResource != "" {
		machine := new(naglfarv1.Machine)
		if err = r.Get(ctx, types.NamespacedName{Namespace: "default", Name: resource.Spec.TestMachineResource}, machine); err != nil {
			log.Error(err, fmt.Sprintf("unable to fetch Machine %s", resource.Spec.TestMachineResource))
			return
		}

		overflow, result.Requeue, err = r.tryRequestResource(machine, resource)

		if result.Requeue {
			return
		}

		if err == nil && overflow {
			err = fmt.Errorf("no enough resource on machine(%s)", resource.Spec.TestMachineResource)
		}

		if err != nil {
			resource.Status.State = naglfarv1.ResourceFail
		} else {
			resource.Status.State = naglfarv1.ResourceUninitialized
		}

		return
	}

	var machines naglfarv1.MachineList
	options := make([]client.ListOption, 0)
	if resource.Spec.MachineSelector != "" {
		options = append(options, client.MatchingLabels{"type": resource.Spec.MachineSelector})
	}

	if err = r.List(ctx, &machines, options...); err != nil {
		log.Error(err, "unable to list machines")
		return
	}

	for _, machine := range machines.Items {
		overflow, result.Requeue, err = r.tryRequestResource(&machine, resource)

		if result.Requeue {
			return
		}

		if err != nil {
			resource.Status.State = naglfarv1.ResourceFail
			return
		}

		if overflow {
			continue
		}

		resource.Status.State = naglfarv1.ResourceUninitialized
		return
	}

	resource.Status.State = naglfarv1.ResourceFail

	return
}

func (r *TestResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naglfarv1.TestResource{}).
		Complete(r)
}
