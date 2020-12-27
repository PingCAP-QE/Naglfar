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
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/PingCAP-QE/Naglfar/pkg/script"
	"github.com/PingCAP-QE/Naglfar/pkg/util"
)

const findProcScript = "find-proc.sh"

const tinyDuration = time.Microsecond

var findProc, _ = script.ScriptBox.FindString(findProcScript)

// ProcChaosReconciler reconciles a ProcChaos object
type ProcChaosReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Ctx    context.Context
}

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

	for states := len(procChaos.Status.States); states < len(procChaos.Spec.Tasks); states++ {
		state := new(naglfarv1.ProcChaosState)
		task := procChaos.Spec.Tasks[states]

		if task.Period != "" {
			state.KilledTime = util.NewTime(time.Now())
			procChaos.Status.States = append(procChaos.Status.States, state)
			continue
		}

		state.KilledNode, err = r.killProc(log, task)
		if err != nil {
			return
		}
		state.KilledTime = util.NewTime(time.Now())
		procChaos.Status.States = append(procChaos.Status.States, state)

		if err = r.Status().Update(r.Ctx, procChaos); err == nil {
			result.RequeueAfter = tinyDuration
		}
		return
	}

	for index, task := range procChaos.Spec.Tasks {
		state := procChaos.Status.States[index]

		_, _ = task, state

		// TODO: do period kill
	}

	return ctrl.Result{}, nil
}

func (r *ProcChaosReconciler) killProc(log logr.Logger, task *naglfarv1.ProcChaosTask) (node string, err error) {
	return
}

func (r *ProcChaosReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naglfarv1.ProcChaos{}).
		Complete(r)
}
