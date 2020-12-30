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
	"github.com/PingCAP-QE/Naglfar/pkg/tiup"
	"github.com/PingCAP-QE/Naglfar/pkg/util"
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
		if !util.StringsContains(ct.ObjectMeta.Finalizers, finalizerName) {
			ct.ObjectMeta.Finalizers = append(ct.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, &ct); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if util.StringsContains(ct.ObjectMeta.Finalizers, finalizerName) {
			if err := r.deleteTopology(ctx, &ct); err != nil {
				return ctrl.Result{}, err
			}
		}
		ct.Finalizers = util.StringsRemove(ct.Finalizers, finalizerName)
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
				r.Recorder.Eventf(&ct, "Warning", "Install", err.Error())
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
		// first create
		if ct.Status.PreTiDBCluster == nil {
			ct.Status.PreTiDBCluster = ct.Spec.TiDBCluster.DeepCopy()
			if err := r.Status().Update(ctx, &ct); err != nil {
				log.Error(err, "unable to update TestClusterTopology")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

		// cluster config no change
		if !tiup.IsClusterConfigModified(ct.Status.PreTiDBCluster, ct.Spec.TiDBCluster) {
			return ctrl.Result{}, nil
		}

		log.Info("Cluster is updating", "clusterName", ct.Name)
		ct.Status.State = naglfarv1.ClusterTopologyStateUpdating
		if err := r.Status().Update(ctx, &ct); err != nil {
			log.Error(err, "unable to update TestClusterTopology")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	case naglfarv1.ClusterTopologyStateUpdating:
		var rr naglfarv1.TestResourceRequest
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: req.Namespace,
			Name:      ct.Spec.ResourceRequest,
		}, &rr); err != nil {
			return ctrl.Result{}, err
		}

		isOK, result, err := r.updateTiDBCluster(ctx, &ct, &rr)
		if !isOK {
			return result, err
		}

		ct.Status.PreTiDBCluster = ct.Spec.TiDBCluster.DeepCopy()
		ct.Status.State = naglfarv1.ClusterTopologyStateReady
		if err := r.Status().Update(ctx, &ct); err != nil {
			log.Error(err, "unable to update TestClusterTopology")
			return ctrl.Result{}, err
		}
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

	resources = filterClusterResources(ct, resourceList)
	requeue, err = r.initResource(ctx, resources, ct)

	if requeue {
		return true, err
	}
	tiupCtl, err := tiup.MakeClusterManager(log, ct.Spec.DeepCopy(), resources, hostname2ClusterIP(resourceList))
	if err != nil {
		return false, err
	}
	return false, tiupCtl.InstallCluster(log, ct.Name, ct.Spec.TiDBCluster.Version)
}

func (r *TestClusterTopologyReconciler) updateServerConfigs(ctx context.Context, ct *naglfarv1.TestClusterTopology, rr *naglfarv1.TestResourceRequest) (requeue bool, err error) {
	log := r.Log.WithValues("updateServerConfigs", types.NamespacedName{
		Namespace: ct.Namespace,
		Name:      ct.Name,
	})

	var resourceList naglfarv1.TestResourceList
	var resources []*naglfarv1.TestResource
	if err := r.List(ctx, &resourceList, client.InNamespace(rr.Namespace), client.MatchingFields{resourceOwnerKey: rr.Name}); err != nil {
		log.Error(err, "unable to list child resources")
	}
	resources = filterClusterResources(ct, resourceList)

	tiupCtl, err := tiup.MakeClusterManager(log, ct.Spec.DeepCopy(), resources)
	if err != nil {
		return false, err
	}
	return false, tiupCtl.UpdateCluster(log, ct.Name, ct)
}

func (r *TestClusterTopologyReconciler) scaleInTiDBCluster(ctx context.Context, ct *naglfarv1.TestClusterTopology, rr *naglfarv1.TestResourceRequest) (requeue bool, err error) {
	log := r.Log.WithValues("scaleInTiDBCluster", types.NamespacedName{
		Namespace: ct.Namespace,
		Name:      ct.Name,
	})

	var resourceList naglfarv1.TestResourceList
	var resources []*naglfarv1.TestResource
	if err := r.List(ctx, &resourceList, client.InNamespace(rr.Namespace), client.MatchingFields{resourceOwnerKey: rr.Name}); err != nil {
		log.Error(err, "unable to list child resources")
	}
	resources = filterClusterResources(ct, resourceList)

	tiupCtl, err := tiup.MakeClusterManager(log, ct.Spec.DeepCopy(), resources)

	if err != nil {
		return false, err
	}
	return false, tiupCtl.ScaleInCluster(log, ct.Name, ct, hostname2ClusterIP(resourceList))
}

func (r *TestClusterTopologyReconciler) scaleOutTiDBCluster(ctx context.Context, ct *naglfarv1.TestClusterTopology, rr *naglfarv1.TestResourceRequest) (requeue bool, err error) {
	log := r.Log.WithValues("scaleOutTiDBCluster", types.NamespacedName{
		Namespace: ct.Namespace,
		Name:      ct.Name,
	})
	var resourceList naglfarv1.TestResourceList
	var resources []*naglfarv1.TestResource
	if err := r.List(ctx, &resourceList, client.InNamespace(rr.Namespace), client.MatchingFields{resourceOwnerKey: rr.Name}); err != nil {
		log.Error(err, "unable to list child resources")
	}
	resources = filterClusterResources(ct, resourceList)
	requeue, err = r.initResource(ctx, resources, ct)
	if requeue {
		return true, err
	}
	tiupCtl, err := tiup.MakeClusterManager(log, ct.Spec.DeepCopy(), resources, hostname2ClusterIP(resourceList))
	if err != nil {
		return false, err
	}
	return false, tiupCtl.ScaleOutCluster(log, ct.Name, ct, hostname2ClusterIP(resourceList))
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
				return client.IgnoreNotFound(err)
			}
			if rr.Status.State != naglfarv1.TestResourceRequestReady {
				return nil
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

func BuildInjectEnvs(t *naglfarv1.TestClusterTopology, resources []*naglfarv1.TestResource) (envs []string, err error) {
	switch {
	case t.Spec.TiDBCluster != nil:
		return buildTiDBClusterInjectEnvs(t, resources)
	default:
		return
	}
}

type strSet []string

func (s strSet) add(elem string) strSet {
	if util.StringsContains(s, elem) {
		return s
	}
	return append(s, elem)
}

func indexResourceExposedPorts(ctf *naglfarv1.TestClusterTopologySpec, trs []*naglfarv1.TestResource) (indexes map[string]strSet, err error) {
	spec, _, err := tiup.BuildSpecification(ctf, trs, true)
	if err != nil {
		return nil, err
	}
	indexes = make(map[string]strSet)
	for _, item := range trs {
		indexes[item.Name] = make(strSet, 0)
	}
	for _, item := range spec.TiDBServers {
		indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.Port))
		indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.StatusPort))
	}
	for _, item := range spec.PDServers {
		indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.ClientPort))
	}
	for _, item := range spec.TiKVServers {
		indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.StatusPort))
	}
	for _, item := range spec.Grafana {
		indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.Port))
	}
	for _, item := range spec.Monitors {
		indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.Port))
	}
	return
}

func buildTiDBClusterInjectEnvs(t *naglfarv1.TestClusterTopology, resources []*naglfarv1.TestResource) (envs []string, err error) {
	spec, _, err := tiup.BuildSpecification(&t.Spec, resources, false)
	if err != nil {
		return
	}
	for idx, item := range spec.TiDBServers {
		envs = append(envs, fmt.Sprintf("tidb%d=%s:%d", idx, item.Host, item.Port))
	}
	for idx, item := range spec.PDServers {
		envs = append(envs, fmt.Sprintf("pd%d=%s:%d", idx, item.Host, item.ClientPort))
	}
	for idx, item := range spec.Monitors {
		envs = append(envs, fmt.Sprintf("prometheus%d=%s:%d", idx, item.Host, item.Port))
	}
	return
}

func hostname2ClusterIP(resourceList naglfarv1.TestResourceList) map[string]string {
	result := make(map[string]string)
	for idx := range resourceList.Items {
		item := &resourceList.Items[idx]
		result[item.Name] = item.Status.ClusterIP
	}
	return result
}

func (r *TestClusterTopologyReconciler) updateTiDBCluster(ctx context.Context, ct *naglfarv1.TestClusterTopology, rr *naglfarv1.TestResourceRequest) (isOk bool, result ctrl.Result, err error) {

	if tiup.IsServerConfigModified(ct.Status.PreTiDBCluster.ServerConfigs, ct.Spec.TiDBCluster.ServerConfigs) {
		// update serverConfig
		requeue, err := r.updateServerConfigs(ctx, ct, rr)
		if err != nil {
			r.Recorder.Event(ct, "Warning", "Update serverConfigs", err.Error())
			return false, ctrl.Result{}, err
		}
		if requeue {
			return false, ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	if tiup.IsScaleIn(ct.Status.PreTiDBCluster, ct.Spec.TiDBCluster) {
		requeue, err := r.scaleInTiDBCluster(ctx, ct, rr)
		if err != nil {
			r.Recorder.Event(ct, "Warning", "Update scale-in", err.Error())
			return false, ctrl.Result{}, err
		}
		if requeue {
			return false, ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	if tiup.IsScaleOut(ct.Status.PreTiDBCluster, ct.Spec.TiDBCluster) {
		requeue, err := r.scaleOutTiDBCluster(ctx, ct, rr)
		if err != nil {
			r.Recorder.Event(ct, "Warning", "Update scale-out", err.Error())
			return false, ctrl.Result{}, err
		}
		if requeue {
			return false, ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}
	return true, ctrl.Result{}, nil
}

func filterClusterResources(ct *naglfarv1.TestClusterTopology, resourceList naglfarv1.TestResourceList) []*naglfarv1.TestResource {
	allHosts := ct.Spec.TiDBCluster.AllHosts()
	result := make([]*naglfarv1.TestResource, 0)
	for idx := range resourceList.Items {
		item := &resourceList.Items[idx]
		if _, ok := allHosts[item.Name]; ok {
			result = append(result, item)
		}
	}
	return result
}

func (r *TestClusterTopologyReconciler) initResource(ctx context.Context, resources []*naglfarv1.TestResource, ct *naglfarv1.TestClusterTopology) (bool, error) {
	var requeue bool
	exposedPortIndexer, err := indexResourceExposedPorts(ct.Spec.DeepCopy(), resources)
	if err != nil {
		return requeue, fmt.Errorf("index exposed ports failed: %v", err)
	}
	for _, resource := range resources {
		switch resource.Status.State {
		case naglfarv1.ResourceUninitialized:
			requeue = true
			if resource.Status.Image == "" {
				resource.Status.Image = tiup.ContainerImage
				resource.Status.CapAdd = []string{"SYS_ADMIN"}
				resource.Status.Binds = append(resource.Status.Binds, "/sys/fs/cgroup:/sys/fs/cgroup:ro")
				resource.Status.ExposedPorts = exposedPortIndexer[resource.Name]
				resource.Status.ExposedPorts = append(resource.Status.ExposedPorts, naglfarv1.SSHPort)
				err := r.Status().Update(ctx, resource)
				if err != nil {
					return false, err
				}
			}
		case naglfarv1.ResourceReady:
			if resource.Status.Image != tiup.ContainerImage {
				return false, fmt.Errorf("resource node %s uses an incorrect image: %s", resource.Name, resource.Status.Image)
			}
		case naglfarv1.ResourcePending, naglfarv1.ResourceFinish, naglfarv1.ResourceDestroy:
			return false, fmt.Errorf("resource node %s is in the `%s` state", resource.Name, naglfarv1.ResourceFinish)
		}
	}
	return requeue, nil
}
