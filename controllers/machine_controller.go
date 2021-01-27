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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/PingCAP-QE/Naglfar/pkg/container"
	dockerutil "github.com/PingCAP-QE/Naglfar/pkg/docker-util"
	"github.com/PingCAP-QE/Naglfar/pkg/ref"
	"github.com/PingCAP-QE/Naglfar/pkg/util"
)

const machineFinalizer = "machine.naglfar.pingcap.com"

// MachineReconciler reconciles a Machine object
type MachineReconciler struct {
	Ctx context.Context
	client.Client
	Log     logr.Logger
	Eventer record.EventRecorder
	Scheme  *runtime.Scheme
}

// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=machines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=machines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=relationships,verbs=get;list
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=relationships/status,verbs=get;update;patch

func (r *MachineReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {
	log := r.Log.WithValues("machine", req.NamespacedName)

	relation, err := r.getRelationship()
	if err != nil {
		return
	}

	if relation.Status.MachineToResources == nil {
		result.Requeue = true
		result.RequeueAfter = time.Second
		log.Info("relationship not ready")
		return
	}

	machine := new(naglfarv1.Machine)

	if err = r.Get(r.Ctx, req.NamespacedName, machine); err != nil {
		log.Error(err, "unable to fetch Machine")

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		err = client.IgnoreNotFound(err)
		return
	}

	log.Info("machine reconcile", "content", machine)

	if machine.ObjectMeta.DeletionTimestamp.IsZero() && !util.StringsContains(machine.ObjectMeta.Finalizers, machineFinalizer) {
		machine.ObjectMeta.Finalizers = append(machine.ObjectMeta.Finalizers, machineFinalizer)
		err = r.Update(r.Ctx, machine)
		return
	}

	if !machine.ObjectMeta.DeletionTimestamp.IsZero() && util.StringsContains(machine.ObjectMeta.Finalizers, machineFinalizer) {
		if machine.Status.State != naglfarv1.MachineShutdown {
			machine.Status.State = naglfarv1.MachineShutdown
			err = r.Status().Update(r.Ctx, machine)
			return
		}
	}

	switch machine.Status.State {
	case naglfarv1.MachineStarting:
		return r.reconcileStarting(log, machine)
	case naglfarv1.MachineReady:
		return r.reconcileRunning(log, machine)
	case naglfarv1.MachineShutdown:
		return r.reconcileShutdown(log, machine)
	default:
		machine.Status.State = naglfarv1.MachineStarting
		err = r.Status().Update(r.Ctx, machine)
		return
	}
}

func (r *MachineReconciler) reconcileStarting(log logr.Logger, machine *naglfarv1.Machine) (result ctrl.Result, err error) {
	dockerClient, err := dockerutil.MakeClient(r.Ctx, machine)
	if err != nil {
		return
	}
	defer dockerClient.Close()

	machine.Status.ChaosPort, err = r.tryLock(machine, dockerClient)
	if err != nil {
		r.Eventer.Event(machine, "Warning", "Lock", err.Error())
		return
	}

	machine.Status.Info, err = r.fetchMachineInfo(log, machine, dockerClient)
	if err != nil {
		r.Eventer.Event(machine, "Warning", "FetchInfo", err.Error())
		return
	}

	machine.Status.State = naglfarv1.MachineReady

	if err = r.Status().Update(r.Ctx, machine); err != nil {
		log.Error(err, "unable to update Machine")
	}
	return
}

func (r *MachineReconciler) reconcileRunning(log logr.Logger, machine *naglfarv1.Machine) (ctrl.Result, error) {
	if machine.Status.UploadPort == 0 {
		dockerClient, err := dockerutil.MakeClient(r.Ctx, machine)
		if err != nil {
			return ctrl.Result{}, err
		}
		defer dockerClient.Close()
		machine.Status.UploadPort, err = r.createUploadDaemon(machine, dockerClient)
		if err != nil {
			r.Eventer.Event(machine, "Warning", "upload-daemon", err.Error())
			return ctrl.Result{}, err
		}
		err = r.Status().Update(r.Ctx, machine)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	relation, err := r.getRelationship()
	if err != nil {
		return ctrl.Result{}, err
	}

	machineKey := ref.CreateRef(&machine.ObjectMeta).Key()

	if relation.Status.MachineToResources[machineKey] == nil {
		relation.Status.MachineToResources[machineKey] = make(naglfarv1.ResourceRefList, 0)
		err = r.Status().Update(r.Ctx, relation)
	}

	return ctrl.Result{}, err
}

func (r *MachineReconciler) reconcileShutdown(log logr.Logger, machine *naglfarv1.Machine) (result ctrl.Result, err error) {
	relation, err := r.getRelationship()
	if err != nil {
		return
	}

	machineRef := ref.CreateRef(&machine.ObjectMeta)
	machineKey := machineRef.Key()

	if resourceList := relation.Status.MachineToResources[machineKey]; len(resourceList) != 0 {
		result.Requeue = true
		result.RequeueAfter = time.Duration(10 * time.Second)
		return
	}

	dockerClient, err := dockerutil.MakeClient(r.Ctx, machine)
	if err != nil {
		return
	}
	defer dockerClient.Close()

	err = r.Unlock(machine, dockerClient)
	if err != nil {
		return
	}

	err = r.deleteUploadDaemon(machine, dockerClient)
	if err != nil {
		return
	}

	delete(relation.Status.MachineToResources, machineKey)
	err = r.Status().Update(r.Ctx, relation)
	if err != nil {
		return
	}

	machine.ObjectMeta.Finalizers = util.StringsRemove(machine.ObjectMeta.Finalizers, machineFinalizer)
	err = r.Update(r.Ctx, machine)
	return
}

func (r *MachineReconciler) getRelationship() (*naglfarv1.Relationship, error) {
	var relation naglfarv1.Relationship
	err := r.Get(r.Ctx, relationshipName, &relation)
	if apierrors.IsNotFound(err) {
		r.Log.Error(err, fmt.Sprintf("relationship(%s) not found", relationshipName))
		err = nil
	}
	return &relation, err
}

func (r *MachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naglfarv1.Machine{}).
		Complete(r)
}

func (r *MachineReconciler) fetchMachineInfo(log logr.Logger, machine *naglfarv1.Machine, dockerClient *dockerutil.Client) (info *naglfarv1.MachineInfo, err error) {
	cfg, hostCfg := container.MachineStatCfg()

	stdout, err := dockerClient.JustExec(container.MachineStat, container.MachineStatImage, dockerutil.RunOptions{
		Config:     cfg,
		HostConfig: hostCfg,
	})

	if err != nil {
		return
	}

	rawInfo := new(naglfarv1.MachineInfo)

	if err = json.Unmarshal(stdout.Bytes(), &rawInfo); err != nil {
		log.Error(err, fmt.Sprintf("fail to unmarshal os-stat result: \"%s\"", stdout.String()))
	}

	info, err = makeMachineInfo(rawInfo)
	return
}

func makeMachineInfo(rawInfo *naglfarv1.MachineInfo) (*naglfarv1.MachineInfo, error) {
	info := rawInfo.DeepCopy()
	memory, err := info.Memory.ToSize()
	if err != nil {
		return nil, err
	}

	info.Memory = util.Size(float64(memory))

	for path, device := range info.StorageDevices {
		totalSize, err := device.Total.ToSize()
		if err != nil {
			return nil, err
		}
		device.Total = util.Size(float64(totalSize))

		usedSize, err := device.Used.ToSize()
		if err != nil {
			return nil, err
		}
		device.Used = util.Size(float64(usedSize))

		info.StorageDevices[path] = device
	}

	return info, nil
}

func (r *MachineReconciler) tryLock(machine *naglfarv1.Machine, dockerClient *dockerutil.Client) (chaosPort int, err error) {
	cfg, hostCfg := container.ChaosDaemonCfg(string(machine.UID))
	stat, err := dockerClient.Run(container.ChaosDaemon, container.ChaosDaemonImage, dockerutil.RunOptions{
		Config:     cfg,
		HostConfig: hostCfg,
	})

	if err != nil {
		return
	}

	if stat.Config == nil || stat.Config.Labels == nil {
		err = fmt.Errorf("invalid lock container: %s", stat.Name)
	}

	uid, ok := stat.Config.Labels[container.LockerLabel]
	if !ok {
		err = fmt.Errorf("container %s has no label `%s`", stat.Name, container.LockerLabel)
		return
	}

	if uid != string(machine.UID) {
		err = fmt.Errorf("machine(%s) locked by other naglfar system: UID(%s)", machine.Name, machine.UID)
		return
	}

	if ports, ok := stat.NetworkSettings.Ports[container.ChaosDaemonPort]; ok && len(ports) > 0 {
		chaosPort, _ = strconv.Atoi(ports[0].HostPort)
	}

	return
}

func (r *MachineReconciler) createUploadDaemon(machine *naglfarv1.Machine, dockerClient *dockerutil.Client) (uploadPort int, err error) {
	cfg, hostCfg := container.UploadDaemonCfg(string(machine.UID))
	stat, err := dockerClient.Run(container.UploadDaemon, container.UploadDaemonImage, dockerutil.RunOptions{
		Config:     cfg,
		HostConfig: hostCfg,
	})

	if err != nil {
		return
	}

	if stat.Config == nil || stat.HostConfig == nil {
		err = fmt.Errorf("invalid occupy container: %s", stat.Name)
	}

	port := strconv.Itoa(container.UploadDaemonInternalPort) + "/tcp"
	if ports, ok := stat.NetworkSettings.Ports[nat.Port(port)]; ok && len(ports) > 0 {
		uploadPort, err = strconv.Atoi(ports[0].HostPort)
		if err != nil {
			return
		}
	}
	if err != nil {
		return 0, err
	}
	if uploadPort == 0 {
		panic(fmt.Sprintf("there's a bug, uploadPort should be zero"))
	}
	return
}

func (r *MachineReconciler) Unlock(machine *naglfarv1.Machine, dockerClient *dockerutil.Client) error {
	info, err := dockerClient.ContainerInspect(r.Ctx, container.ChaosDaemon)

	if err != nil {
		if !docker.IsErrNotFound(err) {
			return err
		}

		// locker not found
		return nil
	}

	if info.Config == nil || info.Config.Labels == nil || info.Config.Labels[container.LockerLabel] != string(machine.UID) {
		// locker released
		return nil
	}

	return dockerClient.ContainerRemove(r.Ctx, container.ChaosDaemon, dockerTypes.ContainerRemoveOptions{Force: true})
}

func (r *MachineReconciler) deleteUploadDaemon(machine *naglfarv1.Machine, dockerClient *dockerutil.Client) error {
	info, err := dockerClient.ContainerInspect(r.Ctx, container.UploadDaemon)

	if err != nil {
		if !docker.IsErrNotFound(err) {
			return err
		}

		// upload not found
		return nil
	}

	if info.Config == nil || info.HostConfig == nil {
		// upload daemon released
		return nil
	}

	return dockerClient.ContainerRemove(r.Ctx, container.UploadDaemon, dockerTypes.ContainerRemoveOptions{Force: true})
}
