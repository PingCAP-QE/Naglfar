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
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ref "k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
)

const resourceFinalizer = "testresource.naglfar.pingcap.com"

func stringsContains(list []string, target string) bool {
	for _, elem := range list {
		if elem == target {
			return true
		}
	}
	return false
}

func stringsRemove(list []string, target string) []string {
	newList := make([]string, 0, len(list)-1)
	for _, elem := range list {
		if target != elem {
			newList = append(newList, elem)
		}
	}
	return newList
}

func resourcesRemove(list []corev1.ObjectReference, resource *naglfarv1.TestResource) []corev1.ObjectReference {
	newList := make([]corev1.ObjectReference, 0, len(list)-1)
	for _, elem := range list {
		if elem.Kind == resource.Kind && elem.Namespace == resource.Namespace && elem.Name == resource.Name {
			continue
		}
		newList = append(newList, elem)
	}
	return newList
}

func timeIsZero(timeStr string) bool {
	datatime, err := time.Parse(time.RFC3339, timeStr)
	return err == nil && datatime.IsZero()
}

// TestResourceReconciler reconciles a TestResource object
type TestResourceReconciler struct {
	client.Client
	Ctx     context.Context
	Log     logr.Logger
	Eventer record.EventRecorder
	Scheme  *runtime.Scheme
}

// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=machines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=machines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

func (r *TestResourceReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {
	log := r.Log.WithValues("testresource", req.NamespacedName)

	resource := new(naglfarv1.TestResource)

	if err = r.Get(r.Ctx, req.NamespacedName, resource); err != nil {
		log.Error(err, "unable to fetch TestResource")

		// maybe resource deleted
		err = client.IgnoreNotFound(err)
		return
	}

	log.Info("resource reconcile", "content", resource)

	if resource.ObjectMeta.DeletionTimestamp.IsZero() && !stringsContains(resource.ObjectMeta.Finalizers, resourceFinalizer) {
		resource.ObjectMeta.Finalizers = append(resource.ObjectMeta.Finalizers, resourceFinalizer)
		err = r.Update(r.Ctx, resource)
		return
	}

	if !resource.ObjectMeta.DeletionTimestamp.IsZero() && stringsContains(resource.ObjectMeta.Finalizers, resourceFinalizer) {
		var machine *naglfarv1.Machine

		machine, err = r.checkHostMachine(log, resource)

		if err != nil {
			return
		}

		if machine != nil {
			result.Requeue, err = r.finalize(resource, machine)
			if err != nil || result.Requeue {
				return
			}

			machine.Status.TestResources = resourcesRemove(machine.Status.TestResources, resource)
			err = r.Status().Update(r.Ctx, machine)

			if err != nil {
				return
			}
		}

		resource.ObjectMeta.Finalizers = stringsRemove(resource.ObjectMeta.Finalizers, resourceFinalizer)
		err = r.Update(r.Ctx, resource)
		return
	}

	switch resource.Status.State {
	case "":
		resource.Status.State = naglfarv1.ResourcePending
		err = r.Status().Update(r.Ctx, resource)
		return
	case naglfarv1.ResourcePending:
		return r.reconcileStatePending(log, resource)
	case naglfarv1.ResourceUninitialized:
		return r.reconcileStateUninitialized(log, resource)
	case naglfarv1.ResourceFail:
		return r.reconcileStateFail(log, resource)
	case naglfarv1.ResourceReady:
		return r.reconcileStateReady(log, resource)
	case naglfarv1.ResourceFinish:
		return r.reconcileStateFinish(log, resource)
	default:
		return
	}
}

func (r *TestResourceReconciler) removeContainer(resource *naglfarv1.TestResource, dockerClient docker.APIClient) (err error) {
	containerName := resource.ContainerName()
	_, err = dockerClient.ContainerInspect(r.Ctx, containerName)

	if err != nil {
		// ignore not found
		if docker.IsErrContainerNotFound(err) {
			err = nil
		}
		return
	}

	err = dockerClient.ContainerRemove(r.Ctx, containerName, dockerTypes.ContainerRemoveOptions{Force: true})
	return
}

func (r *TestResourceReconciler) finalize(resource *naglfarv1.TestResource, machine *naglfarv1.Machine) (requeue bool, err error) {
	dockerClient, err := docker.NewClient(machine.DockerURL(), machine.Spec.DockerVersion, nil, nil)
	if err != nil {
		return
	}

	if err = r.removeContainer(resource, dockerClient); err != nil {
		return
	}

	cleanerName := resource.ContainerCleanerName()
	stats, err := dockerClient.ContainerInspect(r.Ctx, cleanerName)

	if err != nil {
		if !docker.IsErrContainerNotFound(err) {
			return
		}

		err = r.createCleaner(resource, dockerClient)
		if err == nil {
			requeue = true
		}
		return
	}

	if timeIsZero(stats.State.StartedAt) {
		err = dockerClient.ContainerStart(r.Ctx, cleanerName, dockerTypes.ContainerStartOptions{})
		if err == nil {
			requeue = true
		}
		return
	}

	code, err := dockerClient.ContainerWait(r.Ctx, cleanerName)

	if err != nil {
		return
	}

	if code != 0 {
		r.Eventer.Eventf(resource, "Warning", "Clean", "fail to clean container: exit(%d)", code)
	}

	err = dockerClient.ContainerRemove(r.Ctx, cleanerName, dockerTypes.ContainerRemoveOptions{})

	return
}

func (r *TestResourceReconciler) resourceOverflow(machine *naglfarv1.Machine, newResource *naglfarv1.TestResource) (overflow bool, requeue bool, err error) {
	rest := machine.Available()

	if rest == nil {
		requeue = true
		return
	}

	for _, refer := range machine.Status.TestResources {
		resource := new(naglfarv1.TestResource)

		name := types.NamespacedName{Namespace: refer.Namespace, Name: refer.Name}

		log := r.Log.WithValues("testresource", name)

		if err = r.Get(r.Ctx, name, resource); err != nil {
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

	newResource.Status.DiskStat = make(map[string]naglfarv1.DiskStatus)

	for name, disk := range newResource.Spec.Disks {
		for device, diskResource := range rest.Disks {
			if disk.Kind == diskResource.Kind &&
				disk.Size.Unwrap() <= diskResource.Size.Unwrap() {

				delete(rest.Disks, name)
				newResource.Status.DiskStat[name] = naglfarv1.DiskStatus{
					Kind:       disk.Kind,
					Size:       diskResource.Size,
					Device:     device,
					OriginPath: diskResource.MountPath,
					MountPath:  disk.MountPath,
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

	machineRef, err := ref.GetReference(r.Scheme, machine)
	if err != nil {
		log.Error(err, "unable to make reference to machine", "machine", machine)
		return
	}

	machine.Status.TestResources = append(machine.Status.TestResources, *resourceRef)
	newResource.Status.HostMachine = machineRef

	return
}

func (r *TestResourceReconciler) updatePendingResource(machine *naglfarv1.Machine, resource *naglfarv1.TestResource) (requeue bool) {
	log := r.Log.WithValues("testresource", resource.Name)
	if resource.Status.State != naglfarv1.ResourcePending {
		if err := r.Status().Update(r.Ctx, resource); err != nil {
			log.Info("fail to update, maybe conflict", "testresource", types.NamespacedName{Namespace: resource.Name, Name: resource.Name})
			return true
		}
	}

	if resource.Status.State == naglfarv1.ResourceUninitialized {
		if err := r.Status().Update(r.Ctx, machine); err != nil {
			log.Info("fail to update, maybe conflict", "machine", machine.Name)
			return true
		}
	}

	return false
}

func (r *TestResourceReconciler) reconcileStatePending(log logr.Logger, resource *naglfarv1.TestResource) (result ctrl.Result, err error) {
	machine := new(naglfarv1.Machine)
	var machines []naglfarv1.Machine

	defer func() {
		if !result.Requeue {
			result.Requeue = r.updatePendingResource(machine, resource)
		}
	}()

	if resource.Spec.TestMachineResource != "" {
		if err = r.Get(r.Ctx, types.NamespacedName{Namespace: "default", Name: resource.Spec.TestMachineResource}, machine); err != nil {
			log.Error(err, fmt.Sprintf("unable to fetch Machine %s", resource.Spec.TestMachineResource))
			return
		}

		machines = append(machines, *machine)
	} else {
		var machineList naglfarv1.MachineList
		options := make([]client.ListOption, 0)
		if resource.Spec.MachineSelector != "" {
			options = append(options, client.MatchingLabels{"type": resource.Spec.MachineSelector})
		}

		if err = r.List(r.Ctx, &machineList, options...); err != nil {
			log.Error(err, "unable to list machines")
			return
		}

		machines = machineList.Items
	}

	for _, *machine = range machines {
		var overflow bool
		overflow, result.Requeue, err = r.tryRequestResource(machine, resource)

		if result.Requeue || err != nil {
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

func (r *TestResourceReconciler) checkHostMachine(log logr.Logger, targetResource *naglfarv1.TestResource) (machine *naglfarv1.Machine, err error) {
	if targetResource.Status.HostMachine != nil {
		host := targetResource.Status.HostMachine
		hostname := types.NamespacedName{Namespace: host.Namespace, Name: host.Name}
		hostMachine := new(naglfarv1.Machine)

		err = r.Get(r.Ctx, hostname, hostMachine)

		if client.IgnoreNotFound(err) != nil {
			log.Error(err, fmt.Sprintf("unable to fetch Machine %s", hostname))
			return
		}

		if err != nil {
			// not found
			err = nil
			return
		}

		for _, resourceRef := range hostMachine.Status.TestResources {
			if resourceRef.UID == targetResource.UID {
				machine = hostMachine
				return
			}
		}
	}

	return
}

func (r *TestResourceReconciler) createContainer(resource *naglfarv1.TestResource, dockerClient docker.APIClient) (err error) {
	containerName := resource.ContainerName()
	config, hostConfig := resource.ContainerConfig()

	resp, err := dockerClient.ContainerCreate(r.Ctx, config, hostConfig, nil, containerName)
	if err != nil {
		r.Eventer.Event(resource, "Warning", "ContainerCreate", err.Error())
		return
	}

	for _, warning := range resp.Warnings {
		r.Eventer.Event(resource, "Warning", "ContainerCreate", warning)
	}

	return
}

func (r *TestResourceReconciler) createCleaner(resource *naglfarv1.TestResource, dockerClient docker.APIClient) (err error) {
	containerName := resource.ContainerCleanerName()
	config, hostConfig := resource.ContainerCleanerConfig()

	resp, err := dockerClient.ContainerCreate(r.Ctx, config, hostConfig, nil, containerName)
	if err != nil {
		r.Eventer.Event(resource, "Warning", "CleanerCreate", err.Error())
		return
	}

	for _, warning := range resp.Warnings {
		r.Eventer.Event(resource, "Warning", "CleanerCreate", warning)
	}

	return
}

func (r *TestResourceReconciler) getMachineOrRollback(log logr.Logger, resource *naglfarv1.TestResource) (machine *naglfarv1.Machine, rollback bool, err error) {
	machine, err = r.checkHostMachine(log, resource)
	if err != nil {
		return
	}

	if machine == nil {
		resource.Status.HostMachine = nil
		resource.Status.State = naglfarv1.ResourcePending
		err = r.Status().Update(r.Ctx, resource)
		rollback = true
	}
	return
}

func (r *TestResourceReconciler) reconcileStateUninitialized(log logr.Logger, resource *naglfarv1.TestResource) (result ctrl.Result, err error) {
	machine, rollabck, err := r.getMachineOrRollback(log, resource)
	if err != nil || rollabck {
		return
	}

	if resource.Spec.Image == "" {
		return
	}

	dockerClient, err := docker.NewClient(machine.DockerURL(), machine.Spec.DockerVersion, nil, nil)
	if err != nil {
		return
	}

	containerName := resource.ContainerName()
	var stats dockerTypes.ContainerJSON
	stats, err = dockerClient.ContainerInspect(r.Ctx, containerName)

	if err != nil {
		if !docker.IsErrContainerNotFound(err) {
			return
		}

		err = r.createContainer(resource, dockerClient)
		if err == nil {
			result.Requeue = true
		}
		return
	}

	if timeIsZero(stats.State.StartedAt) {
		err = dockerClient.ContainerStart(r.Ctx, containerName, dockerTypes.ContainerStartOptions{})
		if err == nil {
			result.Requeue = true
		}
		return
	}

	if stats.State.Restarting {
		result.Requeue = true
		result.RequeueAfter = time.Second
		return
	}

	if stats.State.Running {
		resource.Status.State = naglfarv1.ResourceReady
	}

	if !timeIsZero(stats.State.FinishedAt) {
		resource.Status.State = naglfarv1.ResourceFinish
	}

	if stats.State.OOMKilled {
		resource.Status.State = naglfarv1.ResourceFail
	}

	err = r.Status().Update(r.Ctx, resource)
	return
}

// TODO: complete reconcileStateFail
func (r *TestResourceReconciler) reconcileStateFail(log logr.Logger, resource *naglfarv1.TestResource) (result ctrl.Result, err error) {
	return
}

// TODO: complete reconcileStateReady
func (r *TestResourceReconciler) reconcileStateReady(log logr.Logger, resource *naglfarv1.TestResource) (result ctrl.Result, err error) {
	machine, rollabck, err := r.getMachineOrRollback(log, resource)
	if err != nil || rollabck {
		return
	}

	dockerClient, err := docker.NewClient(machine.DockerURL(), machine.Spec.DockerVersion, nil, nil)
	if err != nil {
		return
	}

	containerName := resource.ContainerName()
	stats, err := dockerClient.ContainerInspect(r.Ctx, containerName)
	if err != nil {
		if !docker.IsErrContainerNotFound(err) {
			return
		}

		// not found
		resource.Status.State = naglfarv1.ResourceUninitialized
		err = r.Status().Update(r.Ctx, resource)
		return
	}

	if stats.State.Restarting {
		resource.Status.State = naglfarv1.ResourceUninitialized
	}

	if !timeIsZero(stats.State.FinishedAt) {
		resource.Status.State = naglfarv1.ResourceFinish
	}

	if stats.State.OOMKilled {
		resource.Status.State = naglfarv1.ResourceFail
	}

	if resource.Status.State != naglfarv1.ResourceReady {
		err = r.Status().Update(r.Ctx, resource)
	} else {
		result.Requeue = true
		result.RequeueAfter = time.Second
	}

	return
}

// TODO: complete reconcileStateFinish
func (r *TestResourceReconciler) reconcileStateFinish(log logr.Logger, resource *naglfarv1.TestResource) (result ctrl.Result, err error) {
	// TODO: collect logs
	return
}

func (r *TestResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naglfarv1.TestResource{}).
		Complete(r)
}
