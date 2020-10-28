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
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/appleboy/easyssh-proxy"
	"github.com/go-logr/logr"
	"github.com/ngaut/log"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
)

// MachineReconciler reconciles a Machine object
type MachineReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=machines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=machines/status,verbs=get;update;patch

func (r *MachineReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {
	ctx := context.Background()
	log := r.Log.WithValues("machine", req.NamespacedName)

	machine := new(naglfarv1.Machine)

	if err = r.Get(ctx, req.NamespacedName, machine); err != nil {
		log.Error(err, "unable to fetch Machine")

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		err = client.IgnoreNotFound(err)
		return
	}

	log.Info("machine reconcile", "content", machine)

	if machine.Status.Info == nil {
		machine.Status.Info, err = fetchMachineInfo(machine)
		if err != nil {
			return
		}

		if err = r.Status().Update(ctx, machine); err != nil {
			log.Error(err, "unable to update Machine")
			return
		}

		log.Info("machine updated", "content", machine)
	}

	return
}

func (r *MachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naglfarv1.Machine{}).
		Complete(r)
}

func MakeSSHConfig(spec *naglfarv1.MachineSpec) *easyssh.MakeConfig {
	timeout, _ := time.ParseDuration(spec.Timeout)
	return &easyssh.MakeConfig{
		User:     spec.Username,
		Password: spec.Password,
		Server:   spec.Host,
		Port:     strconv.Itoa(spec.Port),
		Timeout:  timeout,
	}
}

func fetchMachineInfo(machine *naglfarv1.Machine) (*naglfarv1.MachineInfo, error) {
	osStatScript, err := ScriptBox.FindString("os-stat.sh")

	if err != nil {
		return nil, err
	}

	ssh := MakeSSHConfig(&machine.Spec)

	stdout, stderr, done, err := ssh.Run(osStatScript)

	if err != nil {
		log.Error(err, "error in executing os-stat")
		return nil, err
	}

	if !done {
		err = fmt.Errorf("script os-stat.sh not complete")
		return nil, err
	}

	if stderr != "" {
		err = fmt.Errorf(stderr)
		log.Error(err, "command returns an error")
		return nil, err
	}

	info := new(naglfarv1.MachineInfo)

	if err = json.Unmarshal([]byte(stdout), info); err != nil {
		log.Error(err, "fail to unmarshal os-stat result: \"%s\"", stdout)
	}

	return info, nil
}
