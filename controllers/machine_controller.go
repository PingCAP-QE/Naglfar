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

	"github.com/appleboy/easyssh-proxy"
	"github.com/go-logr/logr"
	"github.com/ngaut/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	}

	if len(machine.Status.TestResources) != 0 {
		// release leaked resources
		leakedIndexSet := make(map[int]bool)
		resourceSet := make(map[types.NamespacedName]bool)

		for index, resourceRef := range machine.Status.TestResources {
			resourceName := types.NamespacedName{Namespace: resourceRef.Namespace, Name: resourceRef.Name}
			if resourceSet[resourceName] {
				leakedIndexSet[index] = true
				continue
			}
			resourceSet[resourceName] = true

			resource := new(naglfarv1.TestResource)

			err = r.Get(ctx, resourceName, resource)

			if client.IgnoreNotFound(err) != nil {
				return
			}

			if err != nil {
				// not found
				err = nil
				leakedIndexSet[index] = true
				continue
			}

			hostRef := resource.Status.HostMachine

			if hostRef.UID != machine.UID {
				leakedIndexSet[index] = true
				continue
			}
		}

		if len(leakedIndexSet) > 0 {
			log.Info("resources leaked", "set", leakedIndexSet)
			newResourceList := make([]corev1.ObjectReference, 0, len(machine.Status.TestResources)-len(leakedIndexSet))
			for index, resourceRef := range machine.Status.TestResources {
				if !leakedIndexSet[index] {
					newResourceList = append(newResourceList, resourceRef)
				}
			}
			machine.Status.TestResources = newResourceList
			if err = r.Status().Update(ctx, machine); err != nil {
				log.Error(err, "unable to update Machine")
				return
			}
		}
	}

	return
}

func (r *MachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naglfarv1.Machine{}).
		Complete(r)
}

func MakeSSHConfig(spec *naglfarv1.MachineSpec) *easyssh.MakeConfig {
	timeout, _ := spec.Timeout.Parse()
	return &easyssh.MakeConfig{
		User:     spec.Username,
		Password: spec.Password,
		Server:   spec.Host,
		Port:     strconv.Itoa(spec.SSHPort),
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

	rawInfo := new(naglfarv1.MachineInfo)

	if err = json.Unmarshal([]byte(stdout), rawInfo); err != nil {
		log.Error(err, fmt.Sprintf("fail to unmarshal os-stat result: \"%s\"", stdout))
	}

	return makeMachineInfo(rawInfo)
}

func makeMachineInfo(rawInfo *naglfarv1.MachineInfo) (*naglfarv1.MachineInfo, error) {
	info := rawInfo.DeepCopy()
	memory, err := info.Memory.ToSize()
	if err != nil {
		return nil, err
	}

	info.Memory = naglfarv1.Size(float64(memory))

	for path, device := range info.StorageDevices {
		totalSize, err := device.Total.ToSize()
		if err != nil {
			return nil, err
		}
		device.Total = naglfarv1.Size(float64(totalSize))

		usedSize, err := device.Used.ToSize()
		if err != nil {
			return nil, err
		}
		device.Used = naglfarv1.Size(float64(usedSize))

		info.StorageDevices[path] = device
	}

	return info, nil
}
