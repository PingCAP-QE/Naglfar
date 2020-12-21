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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/PingCAP-QE/Naglfar/pkg/ref"
	"github.com/PingCAP-QE/Naglfar/pkg/util"
)

const (
	testWorkloadFinalizer   = "testworkload.naglfar.pingcap.com"
	NaglfarClusterNs        = "NAGLFAR_CLUSTER_NS"
	NaglfarTestWorkloadName = "NAGLFAR_TESTWORKLOAD_NAME"
	NaglfarTestWorkloadItem = "NAGLFAR_TESTWORKLOAD_WORKLOAD_ITEM"
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

	if workload.ObjectMeta.DeletionTimestamp.IsZero() {
		if !util.StringsContains(workload.ObjectMeta.Finalizers, testWorkloadFinalizer) {
			workload.ObjectMeta.Finalizers = append(workload.ObjectMeta.Finalizers, testWorkloadFinalizer)
			if err := r.Update(ctx, workload); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if util.StringsContains(workload.ObjectMeta.Finalizers, testWorkloadFinalizer) {
			if err := r.uninstallWorkload(ctx, workload); err != nil {
				return ctrl.Result{}, err
			}
		}
		workload.Finalizers = util.StringsRemove(workload.Finalizers, testWorkloadFinalizer)
		if err := r.Update(ctx, workload); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	switch workload.Status.State {
	case "":
		workload.Status.State = naglfarv1.TestWorkloadStatePending
		err := r.Status().Update(ctx, workload)
		return ctrl.Result{}, err
	case naglfarv1.TestWorkloadStatePending:
		return r.reconcilePending(ctx, workload)
	case naglfarv1.TestWorkloadStateRunning:
		return r.reconcileRunning(ctx, workload)
	case naglfarv1.TestWorkloadStateFinish:
		return r.reconcileFinish(workload)
	}
	return ctrl.Result{}, nil
}

// 1. check all dependent topologies are installed(maybe we can relax this supposed condition)
// 2. set the workload container configuration on specified resource nodes: image, commands etc
// 3. poll the state of workload resource nodes, if all workloads have started, set itself to `running`
func (r *TestWorkloadReconciler) reconcilePending(ctx context.Context, workload *naglfarv1.TestWorkload) (ctrl.Result, error) {
	clusterTopologies := make(map[types.NamespacedName]struct{})
	for _, item := range workload.Spec.ClusterTopologiesRefs {
		clusterTopologies[types.NamespacedName{
			Namespace: workload.Namespace,
			Name:      item.Name,
		}] = struct{}{}
	}
	topologies, allReady, err := r.checkTopologiesReady(ctx, clusterTopologies)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !allReady {
		r.Recorder.Event(workload, "Warning", "Precondition", "not all clusters are ready")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	var resourceList naglfarv1.TestResourceList
	if err := r.List(ctx, &resourceList, client.InNamespace(workload.Namespace)); err != nil {
		return ctrl.Result{}, err
	}
	var installedCount = 0
	for _, item := range workload.Spec.Workloads {
		workloadNode, err := r.getWorkloadRequestNode(ctx, workload.Namespace, item.DockerContainer)
		if err != nil {
			return ctrl.Result{}, err
		}
		if workloadNode == nil {
			err := fmt.Errorf("cannot find the resource %s", item.DockerContainer.ResourceRequest.Name)
			r.Recorder.Event(workload, "Warning", "Precondition", err.Error())
			return ctrl.Result{}, err
		}
		if workloadNode.Status.ClaimRef != nil && *workloadNode.Status.ClaimRef != ref.CreateRef(&workload.ObjectMeta) {
			r.Recorder.Eventf(workload, "Warning", "Precondition", "node %s is occupied by %s",
				ref.CreateRef(&workloadNode.ObjectMeta).Key(),
				workloadNode.Status.ClaimRef.Key())
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		switch workloadNode.Status.State {
		case naglfarv1.ResourceDestroy:
			return ctrl.Result{RequeueAfter: time.Second}, nil
		case naglfarv1.ResourcePending, naglfarv1.ResourceFail:
			panic(fmt.Sprintf("there's a bug, it shouldn't see the `%s` state", workloadNode.Status.State))
		case naglfarv1.ResourceUninitialized:
			if workloadNode.Status.Image == "" {
				self := ref.CreateRef(&workload.ObjectMeta)
				// claim occupy
				workloadNode.Status.ClaimRef = &self

				r.setContainerSpec(&workloadNode.Status, workload, &item)
				topologyEnvs, err := r.buildTopologyEnvs(&workload.Spec, topologies, resourceList)
				if err != nil {
					r.Recorder.Event(workload, "Warning", "Precondition", err.Error())
					return ctrl.Result{}, err
				}
				workloadNode.Status.Envs = append(workloadNode.Status.Envs, topologyEnvs...)
				if err := r.Status().Update(ctx, workloadNode); err != nil {
					return ctrl.Result{}, err
				}
				r.Recorder.Event(workload, "Normal", "Install", fmt.Sprintf("preparing the workload: %s", item.Name))
			} else if workloadNode.Status.Image != item.DockerContainer.Image {
				err := fmt.Errorf("resource %s has installed a conflict image", workloadNode.Name)
				r.Recorder.Event(workload, "Warning", "Precondition", err.Error())
				return ctrl.Result{}, err
			}
		case naglfarv1.ResourceReady, naglfarv1.ResourceFinish:
			installedCount += 1
		default:
			panic(fmt.Sprintf("there's a bug, forget to process the `%s` state", workloadNode.Status.State))
		}
	}
	// TODO: we can record installed workloads on the `status` field
	if installedCount == len(workload.Spec.Workloads) {
		workload.Status.State = naglfarv1.TestWorkloadStateRunning
		if err := r.Status().Update(ctx, workload); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Event(workload, "Normal", "Install", "all workload has been installed")
		return ctrl.Result{}, nil
	}
	// otherwise, we are still pending
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

// 1. poll the state of workload resource nodes, if all workloads have finished, set itself to `finish`
func (r *TestWorkloadReconciler) reconcileRunning(ctx context.Context, workload *naglfarv1.TestWorkload) (ctrl.Result, error) {
	var finishedCount = 0
	for _, item := range workload.Spec.Workloads {
		workloadNode, err := r.getWorkloadRequestNode(ctx, workload.Namespace, item.DockerContainer)
		if err != nil {
			return ctrl.Result{}, err
		}
		if workloadNode == nil {
			err := fmt.Errorf("cannot find the resource %s", item.DockerContainer.ResourceRequest.Name)
			r.Recorder.Event(workload, "Warning", "Inspect", err.Error())
			return ctrl.Result{}, err
		}
		switch workloadNode.Status.State {
		case naglfarv1.ResourcePending, naglfarv1.ResourceFail, naglfarv1.ResourceUninitialized:
			panic(fmt.Sprintf("there's a bug, it shouldn't see the `%s` state", workloadNode.Status.State))
		case naglfarv1.ResourceReady:
			// no nothing
		case naglfarv1.ResourceFinish:
			finishedCount += 1
		default:
			panic(fmt.Sprintf("it's a bug, we forget to process the `%s` state", workloadNode.Status.State))
		}
	}
	if finishedCount == len(workload.Spec.Workloads) {
		workload.Status.State = naglfarv1.TestWorkloadStateFinish
		if err := r.Status().Update(ctx, workload); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Event(workload, "Normal", "Finish", "all workload has been finished")
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (r *TestWorkloadReconciler) reconcileFinish(workload *naglfarv1.TestWorkload) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *TestWorkloadReconciler) setContainerSpec(containerSpec *naglfarv1.TestResourceStatus, testWorkload *naglfarv1.TestWorkload, workloadSpec *naglfarv1.TestWorkloadItemSpec) {
	var envs []string
	envs = append(envs, fmt.Sprintf("%s=%s", NaglfarClusterNs, testWorkload.Namespace))
	envs = append(envs, fmt.Sprintf("%s=%s", NaglfarTestWorkloadName, testWorkload.Name))
	envs = append(envs, fmt.Sprintf("%s=%s", NaglfarTestWorkloadItem, workloadSpec.Name))
	for _, item := range testWorkload.Spec.ClusterTopologiesRefs {
		envs = append(envs, fmt.Sprintf("%s=%s", item.AliasName, item.Name))
	}

	containerSpec.Image, containerSpec.ImagePullPolicy = workloadSpec.DockerContainer.Image, workloadSpec.DockerContainer.ImagePullPolicy
	containerSpec.Command, containerSpec.Envs = workloadSpec.DockerContainer.Command, envs
	return
}

func (r *TestWorkloadReconciler) getWorkloadRequestNode(ctx context.Context, ns string, workloadSpec *naglfarv1.DockerContainerSpec) (*naglfarv1.TestResource, error) {
	resourceRequest := new(naglfarv1.TestResourceRequest)
	err := r.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      workloadSpec.ResourceRequest.Name,
	}, resourceRequest)
	if err != nil {
		return nil, err
	}
	var workloadNode naglfarv1.TestResource
	err = r.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      workloadSpec.ResourceRequest.Node,
	}, &workloadNode)
	return &workloadNode, err
}

func (r *TestWorkloadReconciler) checkTopologiesReady(ctx context.Context, clusterTopologies map[types.NamespacedName]struct{}) (
	topologies []*naglfarv1.TestClusterTopology,
	ready bool,
	err error) {
	for objectKey := range clusterTopologies {
		topology := new(naglfarv1.TestClusterTopology)
		if err := r.Get(ctx, objectKey, topology); err != nil {
			return nil, false, err
		}
		if topology.Status.State != naglfarv1.ClusterTopologyStateReady {
			return nil, false, nil
		}
		topologies = append(topologies, topology)
	}
	return topologies, true, nil
}

func (r *TestWorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naglfarv1.TestWorkload{}).
		Complete(r)
}

func (r *TestWorkloadReconciler) uninstallWorkload(ctx context.Context, workload *naglfarv1.TestWorkload) error {
	for _, item := range workload.Spec.Workloads {
		workloadNode, err := r.getWorkloadRequestNode(ctx, workload.Namespace, item.DockerContainer)
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		if workloadNode == nil {
			continue
		}
		if workloadNode.Status.State.CouldUninstall() {
			workloadNode.Status.State = naglfarv1.ResourceDestroy
			// release claim
			workloadNode.Status.ClaimRef = nil
			if r.Status().Update(ctx, workloadNode); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *TestWorkloadReconciler) buildTopologyEnvs(spec *naglfarv1.TestWorkloadSpec,
	topologies []*naglfarv1.TestClusterTopology,
	list naglfarv1.TestResourceList) (envs []string, err error) {
	var resources = make([]*naglfarv1.TestResource, 0)
	for idx := range list.Items {
		resources = append(resources, &list.Items[idx])
	}
	for _, topoRef := range spec.ClusterTopologiesRefs {
		for _, topo := range topologies {
			if topo.Name == topoRef.Name {
				var prefix = topoRef.Name
				if len(topoRef.AliasName) != 0 {
					prefix = topoRef.AliasName
				}
				es, err := BuildInjectEnvs(topo, resources)
				if err != nil {
					return nil, err
				}
				for _, env := range es {
					envs = append(envs, fmt.Sprintf("%s_%s", prefix, env))
				}
				break
			}
		}
	}
	return
}
