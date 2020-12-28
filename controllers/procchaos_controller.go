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
	"strings"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	dockerutil "github.com/PingCAP-QE/Naglfar/pkg/docker-util"
	"github.com/PingCAP-QE/Naglfar/pkg/ref"
	"github.com/PingCAP-QE/Naglfar/pkg/script"
	"github.com/PingCAP-QE/Naglfar/pkg/util"
)

const findProcScript = "find-proc.sh"

const tinyDuration = time.Microsecond

// ProcChaosReconciler reconciles a ProcChaos object
type ProcChaosReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Ctx    context.Context
}

// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=machines,verbs=get
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=machines/status,verbs=get
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=relationships,verbs=get
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=relationships/status,verbs=get
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresources,verbs=get
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresources/status,verbs=get
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresourcerequests,verbs=get
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresourcerequests/status,verbs=get
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=procchaos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=procchaos/status,verbs=get;update;patch

func (r *ProcChaosReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {
	log := r.Log.WithValues("procchaos", req.NamespacedName)

	// your logic here

	var procChaos *naglfarv1.ProcChaos
	if err = r.Get(r.Ctx, req.NamespacedName, procChaos); err != nil {
		log.Error(err, "unable to fetch ProcChaos")

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		err = client.IgnoreNotFound(err)
		return
	}

	var request *naglfarv1.TestResourceRequest
	if err = r.Get(r.Ctx, types.NamespacedName{Namespace: req.Namespace, Name: procChaos.Spec.Request}, request); err != nil {
		log.Error(err, fmt.Sprintf("unable to fetch Request(%s)", procChaos.Spec.Request))
		err = client.IgnoreNotFound(err)
		return
	}

	if request.Status.State != naglfarv1.TestResourceRequestReady {
		log.Info(fmt.Sprintf("Request(%s) not ready", procChaos.Spec.Request))
		result.Requeue = true
		result.RequeueAfter = time.Second
		return
	}

	var killAndUpdate = func(task *naglfarv1.ProcChaosTask, state *naglfarv1.ProcChaosState) {
		state.KilledNode, err = r.killProc(log, task, request)
		if err != nil {
			return
		}
		state.KilledTime = util.NewTime(time.Now())

		if err = r.Status().Update(r.Ctx, procChaos); err == nil {
			result.Requeue = true
			result.RequeueAfter = tinyDuration
		}
	}

	for states := len(procChaos.Status.States); states < len(procChaos.Spec.Tasks); states++ {
		state := new(naglfarv1.ProcChaosState)
		task := procChaos.Spec.Tasks[states]
		procChaos.Status.States = append(procChaos.Status.States, state)

		if task.Period != "" {
			state.KilledTime = util.NewTime(time.Now())
			continue
		}

		killAndUpdate(task, state)
		return
	}

	var durations []time.Duration
	for index, task := range procChaos.Spec.Tasks {
		if task.Period == "" {
			continue
		}

		state := procChaos.Status.States[index]
		duration := time.Now().Sub(state.KilledTime.Unwrap()) - task.Period.Unwrap()
		if duration <= 0 {
			killAndUpdate(task, state)
			return
		}
		durations = append(durations, duration)
	}

	result.RequeueAfter = util.MinDuration(durations...)
	return
}

func (r *ProcChaosReconciler) killProc(log logr.Logger, task *naglfarv1.ProcChaosTask, request *naglfarv1.TestResourceRequest) (node string, err error) {
	node = util.RandOne(task.Nodes)
	var resource *naglfarv1.TestResource
	var relation *naglfarv1.Relationship
	var machine *naglfarv1.Machine

	for _, item := range request.Spec.Items {
		if node == item.Name {
			if err = r.Get(r.Ctx, types.NamespacedName{Namespace: request.Namespace, Name: node}, resource); err != nil {
				log.Error(err, fmt.Sprintf("cannot get resource(%s) in namespace(%s)", node, request.Namespace))
				return
			}
		}
	}

	if resource == nil {
		err = fmt.Errorf("node(%s) is not exist", node)
		return
	}

	if resource.Status.State != naglfarv1.MachineReady {
		err = fmt.Errorf("Resource(%s) not ready", node)
		log.Error(err, "wait a while")
		return
	}

	if err = r.Get(r.Ctx, relationshipName, relation); err != nil {
		log.Error(err, fmt.Sprintf("cannot get relationship(%s)", relationshipName))
		return
	}

	resourceRef := ref.CreateRef(&resource.ObjectMeta)
	resourceKey := resourceRef.Key()

	machineRef, exist := relation.Status.ResourceToMachine[resourceKey]

	if !exist {
		err = fmt.Errorf("Resource(%s) is not bound with any machine", node)
		return
	}

	if err = r.Get(r.Ctx, machineRef.Namespaced(), machine); err != nil {
		log.Error(err, fmt.Sprintf("cannot get machine(%s)", machineRef.Key()))
		return
	}

	dockerClient, err := dockerutil.MakeClient(r.Ctx, machine)
	if err != nil {
		log.Error(err, fmt.Sprintf("cannot make docker client from machine(%s)", machineRef.Key()))
		return
	}
	defer dockerClient.Close()

	findProc, err := script.ScriptBox.FindString(findProcScript)
	if err != nil {
		return
	}

	stdout, _, err := dockerClient.Exec(resource.ContainerName(), dockerTypes.ExecConfig{
		AttachStdout: true,
		Env:          []string{fmt.Sprintf("PROC=%s", task.Pattern)},
		Cmd:          []string{"sh", "-c", findProc},
	})
	if err != nil {
		return
	}

	var procs []string

	err = json.Unmarshal(stdout.Bytes(), &procs)
	if err != nil {
		return
	}

	if !task.KillAll {
		procs = []string{util.RandOne(procs)}
	}

	procList := strings.Join(procs, " ")

	stdout, stderr, err := dockerClient.Exec(resource.ContainerName(), dockerTypes.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"sh", "-c", fmt.Sprintf("kill -9 %s", procList)},
	})

	if err != nil {
		return
	}

	log.Info(fmt.Sprintf("kill %s, %s: stderr(%s)", procList, stdout.String(), stderr.String()))

	return
}

func (r *ProcChaosReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naglfarv1.ProcChaos{}).
		Complete(r)
}
