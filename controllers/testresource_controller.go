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
	"strconv"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	dockerutil "github.com/PingCAP-QE/Naglfar/pkg/docker-util"
	"github.com/PingCAP-QE/Naglfar/pkg/ref"
	"github.com/PingCAP-QE/Naglfar/pkg/util"
)

const resourceFinalizer = "testresource.naglfar.pingcap.com"

const clusterNetwork = "naglfar-overlay"

var relationshipName = types.NamespacedName{
	Namespace: "default",
	Name:      "machine-testresource",
}

func refsRemove(list naglfarv1.ResourceRefList, resource ref.Ref) naglfarv1.ResourceRefList {
	newList := make(naglfarv1.ResourceRefList, 0, len(list)-1)
	for _, elem := range list {
		if elem.Ref == resource {
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

func pickCpuSet(cpuSet []int, i int32) (set []int) {
	if len(cpuSet) < int(i) {
		return
	}
	for index := 0; index < int(i); index++ {
		set = append(set, cpuSet[index])
	}
	return
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
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=machines,verbs=get
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=machines/status,verbs=get
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=relationships,verbs=get;list
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=relationships/status,verbs=get;update;patch
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
	if resource.ObjectMeta.DeletionTimestamp.IsZero() && !util.StringsContains(resource.ObjectMeta.Finalizers, resourceFinalizer) {
		resource.ObjectMeta.Finalizers = append(resource.ObjectMeta.Finalizers, resourceFinalizer)
		err = r.Update(r.Ctx, resource)
		return
	}

	if !resource.ObjectMeta.DeletionTimestamp.IsZero() && util.StringsContains(resource.ObjectMeta.Finalizers, resourceFinalizer) {
		var relation *naglfarv1.Relationship

		if relation, err = r.getRelationship(); err != nil {
			return
		}

		resourceRef := ref.CreateRef(&resource.ObjectMeta)
		resourceKey := resourceRef.Key()

		if machineRef, ok := relation.Status.ResourceToMachine[resourceKey]; ok {
			var machine naglfarv1.Machine
			if err = r.Get(r.Ctx, machineRef.Namespaced(), &machine); err != nil {
				// TODO: deal with not found
				return
			}

			result.Requeue, err = r.finalize(resource, &machine)

			if result.Requeue || err != nil {
				return
			}

			machineKey := ref.CreateRef(&machine.ObjectMeta).Key()
			relation.Status.MachineToResources[machineKey] = refsRemove(relation.Status.MachineToResources[machineKey], resourceRef)
			delete(relation.Status.ResourceToMachine, resourceKey)
			if err = r.Status().Update(r.Ctx, relation); err != nil {
				return
			}
		}

		resource.ObjectMeta.Finalizers = util.StringsRemove(resource.ObjectMeta.Finalizers, resourceFinalizer)
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
	case naglfarv1.ResourceDestroy:
		return r.reconcileStateDestroy(log, resource)
	default:
		return
	}
}

func (r *TestResourceReconciler) getRelationship() (*naglfarv1.Relationship, error) {
	var relation naglfarv1.Relationship
	err := r.Get(r.Ctx, relationshipName, &relation)
	if apierrors.IsNotFound(err) {
		r.Log.Error(err, fmt.Sprintf("relationship(%s) not found", relationshipName))
		err = nil
	}
	return &relation, err
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
	dockerClient, err := dockerutil.MakeClient(r.Ctx, machine)
	if err != nil {
		return
	}
	defer dockerClient.Close()

	if err = r.removeContainer(resource, dockerClient); err != nil {
		err = client.IgnoreNotFound(err)
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

func (r *TestResourceReconciler) resourceOverflow(rest *naglfarv1.AvailableResource, newResource *naglfarv1.TestResource) (*naglfarv1.ResourceBinding, bool) {
	if rest == nil {
		return nil, true
	}

	binding := &naglfarv1.ResourceBinding{
		CPUSet: pickCpuSet(rest.IdleCPUSet, newResource.Spec.Cores),
		Memory: newResource.Spec.Memory,
		Disks:  make(map[string]naglfarv1.DiskBinding),
	}

	for name, disk := range newResource.Spec.Disks {
		for device, diskResource := range rest.Disks {
			if disk.Kind == diskResource.Kind &&
				disk.Size.Unwrap() <= diskResource.Size.Unwrap() {

				delete(rest.Disks, name)
				binding.Disks[name] = naglfarv1.DiskBinding{
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

	overflow := len(binding.Disks) < len(newResource.Spec.Disks) ||
		rest.Memory.Unwrap() < newResource.Spec.Memory.Unwrap() ||
		len(binding.CPUSet) < int(newResource.Spec.Cores)

	if overflow {
		return nil, overflow
	}

	return binding, false
}

func (r *TestResourceReconciler) requestResource(resource *naglfarv1.TestResource) (hostIP string, err error) {
	relation, err := r.getRelationship()
	if err != nil {
		return
	}

	resourceRef := ref.CreateRef(&resource.ObjectMeta)
	resourceKey := resourceRef.Key()
	machine := new(naglfarv1.Machine)

	machineRef, success := relation.Status.ResourceToMachine[resourceKey]

	if success {
		err = r.Get(r.Ctx, machineRef.Namespaced(), machine)
		if err == nil {
			hostIP = machine.Spec.Host
		}
		return
	}

	return
}

func (r *TestResourceReconciler) reconcileStatePending(_ logr.Logger, resource *naglfarv1.TestResource) (result ctrl.Result, err error) {
	ip, err := r.requestResource(resource)
	if err != nil {
		return
	}

	if ip != "" {
		resource.Status.HostIP = ip
		resource.Status.State = naglfarv1.ResourceUninitialized
		err = r.Status().Update(r.Ctx, resource)
		return
	}
	return ctrl.Result{RequeueAfter: time.Second}, nil
}

func (r *TestResourceReconciler) getResourceBinding(resourceRef ref.Ref) (binding *naglfarv1.ResourceBinding, err error) {
	relation, err := r.getRelationship()
	if err != nil {
		return
	}

	resourceKey := resourceRef.Key()

	machine := relation.Status.ResourceToMachine[resourceKey]

	return &machine.Binding, nil
}

func (r *TestResourceReconciler) getHostMachine(resourceRef ref.Ref) (*naglfarv1.Machine, error) {
	var machine naglfarv1.Machine

	relation, err := r.getRelationship()
	if err != nil {
		return nil, err
	}

	resourceKey := resourceRef.Key()

	machineRef := relation.Status.ResourceToMachine[resourceKey]

	err = r.Get(r.Ctx, machineRef.Namespaced(), &machine)

	return &machine, err
}

func (r *TestResourceReconciler) createContainer(resource *naglfarv1.TestResource, dockerClient *dockerutil.Client) (err error) {
	containerName := resource.ContainerName()

	binding, err := r.getResourceBinding(ref.CreateRef(&resource.ObjectMeta))

	if err != nil {
		return
	}

	config, hostConfig := resource.ContainerConfig(binding)
	if err = dockerClient.PullImageByPolicy(config.Image, resource.Status.ImagePullPolicy); err != nil {
		r.Eventer.Event(resource, "Warning", "pullImage", fmt.Sprintf("pulling image %s failed: %s", config.Image, err.Error()))
		return
	}
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

func (r *TestResourceReconciler) createCleaner(resource *naglfarv1.TestResource, dockerClient *dockerutil.Client) (err error) {
	containerName := resource.ContainerCleanerName()

	binding, err := r.getResourceBinding(ref.CreateRef(&resource.ObjectMeta))

	if err != nil {
		return
	}

	config, hostConfig := resource.ContainerCleanerConfig(binding)

	if err = dockerClient.PullImageByPolicy(config.Image, resource.Status.ImagePullPolicy); err != nil {
		r.Eventer.Event(resource, "Warning", "pullImage", fmt.Sprintf("pulling image %s failed: %s", config.Image, err.Error()))
		return
	}

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

func (r *TestResourceReconciler) reconcileStateUninitialized(log logr.Logger, resource *naglfarv1.TestResource) (result ctrl.Result, err error) {
	if resource.Status.Image == "" {
		return
	}

	machine, err := r.getHostMachine(ref.CreateRef(&resource.ObjectMeta))
	if err != nil {
		return
	}

	dockerClient, err := dockerutil.MakeClient(r.Ctx, machine)
	if err != nil {
		return
	}
	defer dockerClient.Close()

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
		_, ok := stats.NetworkSettings.Networks[clusterNetwork]
		if !ok {
			err = dockerClient.NetworkConnect(r.Ctx, clusterNetwork, containerName, nil)
			if err == nil {
				result.Requeue = true
			}
			return
		}
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
		network, ok := stats.NetworkSettings.Networks[clusterNetwork]
		if !ok {
			err = fmt.Errorf("network configuration error, miss %s network", clusterNetwork)
			return
		}
		resource.Status.ClusterIP = network.IPAddress
		resource.Status.State = naglfarv1.ResourceReady
		if ports, ok := stats.NetworkSettings.Ports[naglfarv1.SSHPort]; ok && len(ports) > 0 {
			resource.Status.SSHPort, _ = strconv.Atoi(ports[0].HostPort)
		}
		for port, hostPorts := range stats.NetworkSettings.Ports {
			if len(resource.Status.PortBindings) == 0 {
				resource.Status.PortBindings = fmt.Sprintf("%s:%s", port, hostPorts[0].HostPort)
			} else {
				resource.Status.PortBindings = fmt.Sprintf("%s,%s:%s", resource.Status.PortBindings, port, hostPorts[0].HostPort)
			}
		}
	}

	if !timeIsZero(stats.State.FinishedAt) {
		resource.Status.State = naglfarv1.ResourceFinish
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
	machine, err := r.getHostMachine(ref.CreateRef(&resource.ObjectMeta))
	if err != nil {
		return
	}

	dockerClient, err := machine.DockerClient()
	if err != nil {
		return
	}
	defer dockerClient.Close()

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

func (r *TestResourceReconciler) reconcileStateDestroy(log logr.Logger, resource *naglfarv1.TestResource) (result ctrl.Result, err error) {
	machine, err := r.getHostMachine(ref.CreateRef(&resource.ObjectMeta))
	if err != nil {
		return
	}
	result.Requeue, err = r.finalize(resource, machine)
	if result.Requeue || err != nil {
		return
	}
	r.Eventer.Event(resource, "Normal", "uninstall", "uninstall resource successfully")
	// clear all container spec
	resource.Status.ResourceContainerSpec = naglfarv1.ResourceContainerSpec{}
	resource.Status.State = naglfarv1.ResourceUninitialized
	err = r.Status().Update(r.Ctx, resource)
	return
}

func (r *TestResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naglfarv1.TestResource{}).
		Complete(r)
}
