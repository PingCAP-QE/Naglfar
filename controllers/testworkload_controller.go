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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
)

// TestWorkloadReconciler reconciles a TestWorkload object
type TestWorkloadReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testworkloads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testworkloads/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testclustertopologies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testclustertopologies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *TestWorkloadReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("testworkload", req.NamespacedName)

	workload := new(naglfarv1.TestWorkload)
	if err := r.Get(ctx, req.NamespacedName, workload); err != nil {
		log.Error(err, "unable to fetch testworkload")
		// maybe resource deleted
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	switch workload.Status.State {
	case "":
		workload.Status.State = naglfarv1.TestWorkloadStatePending
		err := r.Status().Update(ctx, workload)
		return ctrl.Result{}, err
	case naglfarv1.TestWorkloadStatePending:
		return r.reconcilePending(ctx, workload)
	case naglfarv1.TestWorkloadStateRunning:
		return r.reconcileRunning(workload)
	case naglfarv1.TestWorkloadStateFail:
		return r.reconcileFail(workload)
	case naglfarv1.TestWorkloadStateFinish:
		return r.reconcileFinish(workload)
	}
	// your logic here
	return ctrl.Result{}, nil
}

func (r *TestWorkloadReconciler) reconcilePending(ctx context.Context, workload *naglfarv1.TestWorkload) (ctrl.Result, error) {
	ns := workload.Namespace
	var clusterTopologies map[types.NamespacedName]struct{}
	for _, item := range workload.Spec.ClusterTopologiesRefs {
		clusterTopologies[types.NamespacedName{
			Namespace: workload.Namespace,
			Name:      item.Name,
		}] = struct{}{}
	}
	allReady, err := r.checkTopologiesReady(ctx, clusterTopologies)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !allReady {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}
	var installedCount = 0
	for _, item := range workload.Spec.Workloads {
		resourceRequest := new(naglfarv1.TestResourceRequest)
		err := r.Get(ctx, types.NamespacedName{
			Namespace: ns,
			Name:      item.DockerContainer.ResourceRequest.Name,
		}, resourceRequest)
		if err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(nil)
		}
		var testResources naglfarv1.TestResourceList
		if err := r.List(ctx, &testResources, client.InNamespace(ns), client.MatchingFields{resourceOwnerKey: resourceRequest.Name}); err != nil {
			return ctrl.Result{}, err
		}
		var workloadNode *naglfarv1.TestResource
		var workloadNodeName = fmt.Sprintf("%s-%s", item.DockerContainer.ResourceRequest.Name, item.DockerContainer.ResourceRequest.Node)

		for _, testResource := range testResources.Items {
			if testResource.Name == workloadNodeName {
				workloadNode = &testResource
				break
			}
		}
		if workloadNode == nil {
			err := fmt.Errorf("cannot find the resource: %s", workloadNodeName)
			r.Recorder.Event(workload, "Warning", "Install", err.Error())
			return ctrl.Result{}, err
		}
		switch workloadNode.Status.State {
		case naglfarv1.ResourcePending, naglfarv1.ResourceFail:
			// Confusing state, we need to wait and claim down...
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		case naglfarv1.ResourceUninitialized:
			if workloadNode.Spec.Image == "" {
				workloadNode.Spec.Image, workloadNode.Spec.Commands = item.DockerContainer.Image, item.DockerContainer.Command
				if err := r.Update(ctx, workloadNode); err != nil {
					return ctrl.Result{}, err
				}
			} else if workloadNode.Spec.Image != item.DockerContainer.Image {
				err := fmt.Errorf("install the workload %s/%s failed, resource %s has installed a conflict image",
					workload.Name, item.Name, workloadNode.Name)
				r.Recorder.Event(workload, "Warning", "Install", err.Error())
				return ctrl.Result{}, err
			}
		case naglfarv1.ResourceReady, naglfarv1.ResourceFinish:
			installedCount += 1
		default:
			panic(fmt.Sprintf("it's a bug, we forget to process the `%s` state", workloadNode.Status.State))
		}
	}
	if installedCount == len(workload.Spec.Workloads) {
		workload.Status.State = naglfarv1.TestWorkloadStateRunning
		if err := r.Status().Update(ctx, workload); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Event(workload, "Normal", "Install", "all workload has been installed")
	}
	// otherwise, we are still pending
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (r *TestWorkloadReconciler) reconcileRunning(workload *naglfarv1.TestWorkload) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *TestWorkloadReconciler) reconcileFail(workload *naglfarv1.TestWorkload) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *TestWorkloadReconciler) reconcileFinish(workload *naglfarv1.TestWorkload) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *TestWorkloadReconciler) checkTopologiesReady(ctx context.Context, clusterTopologies map[types.NamespacedName]struct{}) (bool, error) {
	for objectKey := range clusterTopologies {
		topology := new(naglfarv1.TestClusterTopology)
		if err := r.Get(ctx, objectKey, topology); err != nil {
			return false, err
		}
		if topology.Status.State != naglfarv1.ClusterTopologyStateReady {
			return false, nil
		}
	}
	return true, nil
}

func (r *TestWorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naglfarv1.TestWorkload{}).
		Complete(r)
}
