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

	"github.com/docker/docker/api/types/mount"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/PingCAP-QE/Naglfar/pkg/tiup"
)

// TestClusterTopologyReconciler reconciles a TestClusterTopology object
type TestClusterTopologyReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testclustertopologies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testclustertopologies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *TestClusterTopologyReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("testclustertopology", req.NamespacedName)

	var ct naglfarv1.TestClusterTopology
	if err := r.Get(ctx, req.NamespacedName, &ct); err != nil {
		log.Error(err, "unable to fetch TestClusterTopology")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	finalizerName := req.Name
	if ct.ObjectMeta.DeletionTimestamp.IsZero() {
		if !stringsContains(ct.ObjectMeta.Finalizers, finalizerName) {
			ct.ObjectMeta.Finalizers = append(ct.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, &ct); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if stringsContains(ct.ObjectMeta.Finalizers, finalizerName) {
			if err := r.deleteTopology(ctx, &ct); err != nil {
				return ctrl.Result{}, err
			}
		}
		ct.Finalizers = stringsRemove(ct.Finalizers, finalizerName)
		if err := r.Update(ctx, &ct); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	switch ct.Status.State {
	case "":
		ct.Status.State = naglfarv1.ClusterTopologyStatePending
		err := r.Status().Update(ctx, &ct)
		return ctrl.Result{}, err
	case naglfarv1.ClusterTopologyStatePending:
		var rr naglfarv1.TestResourceRequest
		// we should install a SUT on the resources what we have requested
		if len(ct.Spec.ResourceRequest) != 0 {
			if err := r.Get(ctx, types.NamespacedName{
				Namespace: req.Namespace,
				Name:      ct.Spec.ResourceRequest,
			}, &rr); err != nil {
				return ctrl.Result{}, err
			}
			if rr.Status.State != naglfarv1.TestResourceRequestReady {
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			}
			switch {
			case ct.Spec.TiDBCluster != nil:
				requeue, err := r.installTiDBCluster(ctx, &ct, &rr)
				if err != nil {
					r.Recorder.Event(&ct, "Warning", "Install", err.Error())
					return ctrl.Result{}, err
				}
				if requeue {
					return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
				}
				r.Recorder.Event(&ct, "Normal", "Install", fmt.Sprintf("cluster %s is installed", ct.Name))
			}
			ct.Status.State = naglfarv1.ClusterTopologyStateReady
			if err := r.Status().Update(ctx, &ct); err != nil {
				log.Error(err, "unable to update TestClusterTopology")
				return ctrl.Result{}, err
			}
		} else {
			// use the cluster created by tidb-operator
			// TODO: wait for the cluster be ready
		}
	case naglfarv1.ClusterTopologyStateReady:
		// DO NOTHING
	}
	return ctrl.Result{}, nil
}

func (r *TestClusterTopologyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naglfarv1.TestClusterTopology{}).
		Complete(r)
}

func (r *TestClusterTopologyReconciler) installTiDBCluster(ctx context.Context, ct *naglfarv1.TestClusterTopology, rr *naglfarv1.TestResourceRequest) (requeue bool, err error) {
	log := r.Log.WithValues("installTiDBCluster", types.NamespacedName{
		Namespace: ct.Namespace,
		Name:      ct.Name,
	})
	var resourceList naglfarv1.TestResourceList
	var resources []*naglfarv1.TestResource
	if err := r.List(ctx, &resourceList, client.InNamespace(rr.Namespace), client.MatchingFields{resourceOwnerKey: rr.Name}); err != nil {
		log.Error(err, "unable to list child resources")
	}
	filterClusterResources := func() []*naglfarv1.TestResource {
		allHosts := ct.Spec.TiDBCluster.AllHosts()
		result := make([]*naglfarv1.TestResource, 0)
		for idx, item := range resourceList.Items {
			if _, ok := allHosts[item.Name]; ok {
				result = append(result, &resourceList.Items[idx])
			}
		}
		return result
	}
	resources = filterClusterResources()
	for _, resource := range resources {
		switch resource.Status.State {
		case naglfarv1.ResourceUninitialized:
			requeue = true
			if resource.Status.Image == "" {
				// TODO fix hardcode
				resource.Status.Image = tiup.ContainerImage
				resource.Status.Privilege = true
				resource.Status.Mounts = []naglfarv1.TestResourceMount{{
					Type:     mount.TypeBind,
					Source:   "/sys/fs/cgroup",
					Target:   "/sys/fs/cgroup",
					ReadOnly: true,
				}}
				err := r.Status().Update(ctx, resource)
				if err != nil {
					return false, err
				}
			}
		case naglfarv1.ResourceReady:
			if resource.Status.Image != tiup.ContainerImage {
				return false, fmt.Errorf("resource node %s uses an incorrect image: %s", resource.Name, resource.Status.Image)
			}
		case naglfarv1.ResourcePending, naglfarv1.ResourceFail, naglfarv1.ResourceFinish, naglfarv1.ResourceDestroy:
			return false, fmt.Errorf("resource node %s is in the `%s` state", resource.Name, naglfarv1.ResourceFinish)
		}
	}
	if requeue {
		return true, nil
	}
	tiupCtl, err := tiup.MakeClusterManager(log, ct.Spec.DeepCopy(), resources)
	if err != nil {
		return false, err
	}
	return false, tiup.IgnoreClusterDuplicated(tiupCtl.InstallCluster(log, ct.Name, ct.Spec.TiDBCluster.Version))
}

func (r *TestClusterTopologyReconciler) deleteTopology(ctx context.Context, ct *naglfarv1.TestClusterTopology) error {
	log := r.Log.WithValues("deleteTopology", types.NamespacedName{
		Namespace: ct.Namespace,
		Name:      ct.Name,
	})
	// if we cluster is installed on resource nodes
	if len(ct.Spec.ResourceRequest) != 0 {
		switch {
		case ct.Spec.TiDBCluster != nil:
			var rr naglfarv1.TestResourceRequest
			if err := r.Get(ctx, types.NamespacedName{
				Namespace: ct.Namespace,
				Name:      ct.Spec.ResourceRequest,
			}, &rr); err != nil {
				return err
			}
			if rr.Status.State != naglfarv1.TestResourceRequestReady {
				return fmt.Errorf("testResourceRequest %s/%s isn't ready", rr.Namespace, rr.Name)
			}
			var resourceList naglfarv1.TestResourceList
			var resources []*naglfarv1.TestResource
			if err := r.List(ctx, &resourceList, client.InNamespace(rr.Namespace), client.MatchingFields{resourceOwnerKey: rr.Name}); err != nil {
				log.Error(err, "unable to list child resources")
			}
			for idx := range resourceList.Items {
				resources = append(resources, &resourceList.Items[idx])
			}
			tiupCtl, err := tiup.MakeClusterManager(r.Log, ct.Spec.DeepCopy(), resources)
			if err != nil {
				return err
			}
			if err := tiupCtl.UninstallCluster(ct.Name); err != nil {
				// we ignore cluster not exist error
				return tiup.IgnoreClusterNotExist(err)
			}
		}
	}
	return nil
}
