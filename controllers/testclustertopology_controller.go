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
	"strconv"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/PingCAP-QE/Naglfar/pkg/flink"
	"github.com/PingCAP-QE/Naglfar/pkg/haproxy"
	"github.com/PingCAP-QE/Naglfar/pkg/ref"
	"github.com/PingCAP-QE/Naglfar/pkg/tiup"
	"github.com/PingCAP-QE/Naglfar/pkg/tiup/cluster"
	dm "github.com/PingCAP-QE/Naglfar/pkg/tiup/dm"
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
		if ct.Spec.ResourceRequest != "" {
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
				r.Recorder.Event(&ct, "Normal", "Install", fmt.Sprintf("tidb cluster %s is installed", ct.Name))
			case ct.Spec.FlinkCluster != nil:
				requeue, err := r.installFlinkCluster(ctx, &ct, &rr)
				if err != nil {
					r.Recorder.Event(&ct, "Warning", "Install", err.Error())
					return ctrl.Result{}, err
				}
				if requeue {
					return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
				}
				r.Recorder.Event(&ct, "Normal", "Install", fmt.Sprintf("flink cluster %s is installed", ct.Name))
			case ct.Spec.DMCluster != nil:
				requeue, err := r.installDMCluster(ctx, &ct, &rr)
				if err != nil {
					r.Recorder.Event(&ct, "Warning", "Install", err.Error())
					return ctrl.Result{}, err
				}
				if requeue {
					return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
				}
				r.Recorder.Event(&ct, "Normal", "Install", fmt.Sprintf("dm cluster %s is installed", ct.Name))
			}
			ct.Status.State = naglfarv1.ClusterTopologyStateReady
			if err := r.Status().Update(ctx, &ct); err != nil {
				log.Error(err, "unable to update TestClusterTopology")
				return ctrl.Result{}, err
			}
		}
		//else {
		// use the cluster created by tidb-operator
		// TODO: wait for the cluster be ready
		//}
	case naglfarv1.ClusterTopologyStateReady:
		// first create
		switch {
		case ct.Spec.TiDBCluster != nil:
			if ct.Status.PreTiDBCluster == nil {
				ct.Status.PreTiDBCluster = ct.Spec.TiDBCluster.DeepCopy()
				if err := r.Status().Update(ctx, &ct); err != nil {
					log.Error(err, "unable to update TestClusterTopology")
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, nil
			}
			// cluster config no change
			if !cluster.IsClusterConfigModified(ct.Status.PreTiDBCluster, ct.Spec.TiDBCluster) {
				if ct.Status.TiDBClusterInfo.IsScaling {
					var rr naglfarv1.TestResourceRequest
					if err := r.Get(ctx, types.NamespacedName{
						Namespace: req.Namespace,
						Name:      ct.Spec.ResourceRequest,
					}, &rr); err != nil {
						return ctrl.Result{}, err
					}

					// check whether all the scale-in tikvs have migrated the region to other tikvs and are in the tombstone state. If finished, prune these useless tikvs
					isFinish, err := r.checkTiKVFinishedScaleIn(ctx, &ct, &rr)
					if err != nil {
						return ctrl.Result{}, err
					}
					if err := r.Status().Update(ctx, &ct); err != nil {
						log.Error(err, "unable to update TestClusterTopology")
						return ctrl.Result{}, err
					}

					if isFinish {
						isOK, result, err := r.pruneTiDBCluster(ctx, &ct, &rr)
						if err != nil {
							r.Recorder.Event(&ct, "Warning", "Prune TiKVs", err.Error())
							return ctrl.Result{}, err
						}
						if isOK {
							ct.Status.TiDBClusterInfo.IsScaling = false
							if err := r.Status().Update(ctx, &ct); err != nil {
								log.Error(err, "unable to update TestClusterTopology")
								return ctrl.Result{}, err
							}
							return ctrl.Result{}, nil
						}
						return result, nil
					} else {
						return ctrl.Result{RequeueAfter: time.Second * 10}, nil
					}
				}
				return ctrl.Result{}, nil
			}
			log.Info("Cluster is updating", "clusterName", ct.Name)
			ct.Status.State = naglfarv1.ClusterTopologyStateUpdating
			if err := r.Status().Update(ctx, &ct); err != nil {
				log.Error(err, "unable to update TestClusterTopology")
				return ctrl.Result{}, err
			}
		case ct.Spec.FlinkCluster != nil:
		case ct.Spec.DMCluster != nil:

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

		if cluster.IsScaleIn(ct.Status.PreTiDBCluster, ct.Spec.TiDBCluster) {
			ct.Status.TiDBClusterInfo.IsScaling = true
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
	requeue, err = r.initResource(ctx, ct, resources)
	if requeue {
		return true, err
	}
	tiupCtl, err := cluster.MakeClusterManager(log, ct.Spec.DeepCopy(), resources, hostname2ClusterIP(resourceList))
	if err != nil {
		return false, err
	}
	return false, tiupCtl.InstallCluster(log, ct.Name, ct.Spec.TiDBCluster.Version)
}

func (r *TestClusterTopologyReconciler) upgradeTiDBCluster(ctx context.Context, ct *naglfarv1.TestClusterTopology, rr *naglfarv1.TestResourceRequest) (requeue bool, err error) {
	log := r.Log.WithValues("upgradeTiDBCluster", types.NamespacedName{
		Namespace: ct.Namespace,
		Name:      ct.Name,
	})

	var resourceList naglfarv1.TestResourceList
	var resources []*naglfarv1.TestResource
	if err := r.List(ctx, &resourceList, client.InNamespace(rr.Namespace), client.MatchingFields{resourceOwnerKey: rr.Name}); err != nil {
		log.Error(err, "unable to list child resources")
		return false, err
	}
	resources = filterClusterResources(ct, resourceList)

	tiupCtl, err := cluster.MakeClusterManager(log, ct.Spec.DeepCopy(), resources)

	if err != nil {
		return false, err
	}
	return false, tiupCtl.UpgradeCluster(log, ct.Name, ct, hostname2ClusterIP(resourceList))
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

	tiupCtl, err := cluster.MakeClusterManager(log, ct.Spec.DeepCopy(), resources)
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

	tiupCtl, err := cluster.MakeClusterManager(log, ct.Spec.DeepCopy(), resources)

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
	requeue, err = r.initResource(ctx, ct, resources)
	if requeue {
		return true, err
	}
	tiupCtl, err := cluster.MakeClusterManager(log, ct.Spec.DeepCopy(), resources, hostname2ClusterIP(resourceList))
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
	if ct.Spec.ResourceRequest != "" {
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
		switch {
		case ct.Spec.TiDBCluster != nil:
			if ct.Spec.TiDBCluster.HAProxy != nil {
				for i := 0; i < len(resources); i++ {
					if resources[i].Name == ct.Spec.TiDBCluster.HAProxy.Host {
						machine, err := r.getResourceMachine(ctx, resources[i])
						if err != nil {
							return err
						}
						err = haproxy.DeleteConfigFromMachine(machine.Spec.Host, haproxy.GenerateFilePrefix(ct)+haproxy.FileName)
						if err != nil {
							return err
						}
					}
				}
			}
			tiupCtl, err := cluster.MakeClusterManager(r.Log, ct.Spec.DeepCopy(), resources)
			if err != nil {
				return err
			}
			if err := tiupCtl.UninstallCluster(ct.Name); err != nil {
				// we ignore cluster not exist error
				return tiup.IgnoreClusterNotExist(err)
			}
		case ct.Spec.FlinkCluster != nil:
			log.Info("flink cluster is uninstalled")
		case ct.Spec.DMCluster != nil:
			tiupCtl, err := dm.MakeClusterManager(r.Log, ct.Spec.DeepCopy(), resources)
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
	indexes = make(map[string]strSet)
	switch {
	case ctf.TiDBCluster != nil:
		spec, _, err := cluster.BuildSpecification(ctf, trs, true)
		if err != nil {
			return nil, err
		}
		for _, item := range trs {
			indexes[item.Name] = make(strSet, 0)
		}
		for _, item := range spec.TiDBServers {
			indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.Port))
		}
		for _, item := range spec.PDServers {
			indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.ClientPort))
		}
		for _, item := range spec.Grafanas {
			indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.Port))
		}
		for _, item := range spec.Monitors {
			indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.Port))
		}
		for _, item := range spec.CDCServers {
			indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.Port))
		}
		for _, item := range spec.TiFlashServers {
			indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.TCPPort)).
				add(fmt.Sprintf("%d/tcp", item.HTTPPort)).
				add(fmt.Sprintf("%d/tcp", item.StatusPort)).
				add(fmt.Sprintf("%d/tcp", item.FlashServicePort)).
				add(fmt.Sprintf("%d/tcp", item.FlashProxyPort)).
				add(fmt.Sprintf("%d/tcp", item.FlashProxyStatusPort))
		}
		for idx := range indexes {
			indexes[idx] = indexes[idx].add(naglfarv1.SSHPort)
		}
	case ctf.FlinkCluster != nil:
		spec, _, err := flink.BuildSpecification(ctf, trs, true)
		if err != nil {
			return nil, err
		}
		indexes[spec.JobManager.Host] = indexes[spec.JobManager.Host].add(fmt.Sprintf("%d/tcp", spec.JobManager.WebPort))
	case ctf.DMCluster != nil:
		spec, _, err := dm.BuildSpecification(ctf, trs, true)
		if err != nil {
			return nil, err
		}
		for _, item := range trs {
			indexes[item.Name] = make(strSet, 0)
		}
		for _, item := range spec.Masters {
			indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.Port))
		}
		for _, item := range spec.Workers {
			indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.Port))
		}
		for _, item := range spec.Grafanas {
			indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.Port))
		}
		for _, item := range spec.Alertmanagers {
			indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.WebPort))
		}
		for _, item := range spec.Monitors {
			indexes[item.Host] = indexes[item.Host].add(fmt.Sprintf("%d/tcp", item.Port))
		}
		for idx := range indexes {
			indexes[idx] = indexes[idx].add(naglfarv1.SSHPort)
		}
	}
	return
}

func buildTiDBClusterInjectEnvs(t *naglfarv1.TestClusterTopology, resources []*naglfarv1.TestResource) (envs []string, err error) {
	spec, _, err := cluster.BuildSpecification(&t.Spec, resources, false)
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
	// set haproxy env
	if t.Spec.TiDBCluster.HAProxy != nil {
		for _, item := range resources {
			if item.Name == t.Spec.TiDBCluster.HAProxy.Host {
				envs = append(envs, fmt.Sprintf("tidb=%s:%d", item.Status.ClusterIP, t.Spec.TiDBCluster.HAProxy.Port))
			}
		}
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

func (r *TestClusterTopologyReconciler) checkTiKVFinishedScaleIn(ctx context.Context, ct *naglfarv1.TestClusterTopology, rr *naglfarv1.TestResourceRequest) (isFinish bool, err error) {
	log := r.Log.WithValues("checkTiKVFinishedScaleIn", types.NamespacedName{
		Namespace: ct.Namespace,
		Name:      ct.Name,
	})

	var resourceList naglfarv1.TestResourceList
	var resources []*naglfarv1.TestResource
	if err := r.List(ctx, &resourceList, client.InNamespace(rr.Namespace), client.MatchingFields{resourceOwnerKey: rr.Name}); err != nil {
		log.Error(err, "unable to list child resources")
	}
	resources = filterClusterResources(ct, resourceList)

	tiupCtl, err := cluster.MakeClusterManager(log, ct.Spec.DeepCopy(), resources)
	if err != nil {
		return false, err
	}

	pendingOfflineList, err := tiupCtl.GetNodeStatusList(log, ct.Name, "Pending")
	if err != nil {
		return false, err
	}
	tmp, err := tiupCtl.GetNodeStatusList(log, ct.Name, "Offline")
	if err != nil {
		return false, err
	}
	var offlineList []string
	for i := 0; i < len(tmp); i++ {
		isExist := false
		for j := 0; j < len(pendingOfflineList); j++ {
			if tmp[i] == pendingOfflineList[j] {
				isExist = true
			}
		}
		if !isExist {
			offlineList = append(offlineList, tmp[i])
		}
	}
	ct.Status.TiDBClusterInfo.PendingOfflineList = pendingOfflineList
	ct.Status.TiDBClusterInfo.OfflineList = offlineList
	if len(pendingOfflineList) != 0 || len(offlineList) != 0 {
		log.Info("cluster is trying prune", "PendingOfflineList", pendingOfflineList, "OfflineList", offlineList)
		return false, nil
	}

	return true, nil
}

func (r *TestClusterTopologyReconciler) pruneTiDBCluster(ctx context.Context, ct *naglfarv1.TestClusterTopology, rr *naglfarv1.TestResourceRequest) (isOk bool, result ctrl.Result, err error) {
	log := r.Log.WithValues("pruneTiDBCluster", types.NamespacedName{
		Namespace: ct.Namespace,
		Name:      ct.Name,
	})

	var resourceList naglfarv1.TestResourceList
	var resources []*naglfarv1.TestResource
	if err := r.List(ctx, &resourceList, client.InNamespace(rr.Namespace), client.MatchingFields{resourceOwnerKey: rr.Name}); err != nil {
		log.Error(err, "unable to list child resources")
	}
	resources = filterClusterResources(ct, resourceList)

	tiupCtl, err := cluster.MakeClusterManager(log, ct.Spec.DeepCopy(), resources)
	if err != nil {
		return false, ctrl.Result{}, err
	}

	err = tiupCtl.PruneCluster(log, ct.Name, ct)
	if err != nil {
		return false, ctrl.Result{}, err
	}
	log.Info("prune tidb cluster successfully")
	return true, ctrl.Result{}, nil
}

func (r *TestClusterTopologyReconciler) updateTiDBCluster(ctx context.Context, ct *naglfarv1.TestClusterTopology, rr *naglfarv1.TestResourceRequest) (isOk bool, result ctrl.Result, err error) {

	if cluster.IsUpgraded(ct.Status.PreTiDBCluster, ct.Spec.TiDBCluster) {
		requeue, err := r.upgradeTiDBCluster(ctx, ct, rr)
		if err != nil {
			r.Recorder.Event(ct, "Warning", "Upgrade", err.Error())
			return false, ctrl.Result{}, err
		}
		if requeue {
			return false, ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}
	if cluster.IsServerConfigModified(ct.Status.PreTiDBCluster.ServerConfigs, ct.Spec.TiDBCluster.ServerConfigs) {
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

	if cluster.IsScaleIn(ct.Status.PreTiDBCluster, ct.Spec.TiDBCluster) {
		requeue, err := r.scaleInTiDBCluster(ctx, ct, rr)
		if err != nil {
			r.Recorder.Event(ct, "Warning", "Update scale-in", err.Error())
			return false, ctrl.Result{}, err
		}
		if requeue {
			return false, ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	if cluster.IsScaleOut(ct.Status.PreTiDBCluster, ct.Spec.TiDBCluster) {
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
	var allHosts map[string]struct{}
	switch {
	case ct.Spec.TiDBCluster != nil:
		allHosts = ct.Spec.TiDBCluster.AllHosts()
	case ct.Spec.FlinkCluster != nil:
		allHosts = ct.Spec.FlinkCluster.AllHosts()
	case ct.Spec.DMCluster != nil:
		allHosts = ct.Spec.DMCluster.AllHosts()
	}
	result := make([]*naglfarv1.TestResource, 0)
	for idx := range resourceList.Items {
		item := &resourceList.Items[idx]
		if _, ok := allHosts[item.Name]; ok {
			result = append(result, item)
		}
	}
	return result
}

func (r *TestClusterTopologyReconciler) initResource(ctx context.Context, ct *naglfarv1.TestClusterTopology, resources []*naglfarv1.TestResource) (bool, error) {
	var requeue bool
	var err error
	switch {
	case ct.Spec.TiDBCluster != nil:
		requeue, err = r.initTiDBResources(ctx, ct, resources)
	case ct.Spec.DMCluster != nil:
		requeue, err = r.initDMResources(ctx, ct, resources)
	case ct.Spec.FlinkCluster != nil:
		requeue, err = r.initFlinkResources(ctx, ct, resources)
	}
	return requeue, err
}

func (r *TestClusterTopologyReconciler) initDMResources(ctx context.Context, ct *naglfarv1.TestClusterTopology, resources []*naglfarv1.TestResource) (bool, error) {
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

func (r *TestClusterTopologyReconciler) initTiDBResources(ctx context.Context, ct *naglfarv1.TestClusterTopology, resources []*naglfarv1.TestResource) (bool, error) {
	var requeue bool
	var haProxyResources []*naglfarv1.TestResource
	var tiupResources []*naglfarv1.TestResource
	if ct.Spec.TiDBCluster.HAProxy != nil {
		for i := 0; i < len(resources); i++ {
			if resources[i].Name == ct.Spec.TiDBCluster.HAProxy.Host {
				haProxyResources = append(haProxyResources, resources[i])
			} else {
				tiupResources = append(tiupResources, resources[i])
			}
		}
	} else {
		tiupResources = resources
	}

	exposedPortIndexer, err := indexResourceExposedPorts(ct.Spec.DeepCopy(), resources)
	if err != nil {
		return requeue, fmt.Errorf("index exposed ports failed: %v", err)
	}
	for _, resource := range tiupResources {
		switch resource.Status.State {
		case naglfarv1.ResourceUninitialized:
			requeue = true
			if resource.Status.Image == "" {
				resource.Status.Image = tiup.ContainerImage
				resource.Status.CapAdd = []string{"SYS_ADMIN"}
				resource.Status.Binds = append(resource.Status.Binds, "/sys/fs/cgroup:/sys/fs/cgroup:ro")
				resource.Status.ExposedPorts = exposedPortIndexer[resource.Name]
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

	if ct.Spec.TiDBCluster.HAProxy != nil {
		var machine *naglfarv1.Machine
		machine, err := r.getResourceMachine(ctx, haProxyResources[0])
		if err != nil {
			return true, nil
		}
		clusterIPMaps := make(map[string]string)
		for i := 0; i < len(tiupResources); i++ {
			if tiupResources[i].Status.State != naglfarv1.ResourceReady {
				return true, nil
			} else {
				clusterIPMaps[tiupResources[i].Name] = tiupResources[i].Status.ClusterIP
			}
		}
		requeue, config := haproxy.GenerateHAProxyConfig(ct.Spec.TiDBCluster, clusterIPMaps)
		if requeue {
			return true, nil
		}
		err = haproxy.WriteConfigToMachine(machine.Spec.Host, haproxy.GenerateFilePrefix(ct)+haproxy.FileName, config)
		if err != nil {
			return true, err
		}
		for _, resource := range haProxyResources {
			if resource.Name != ct.Spec.TiDBCluster.HAProxy.Host {
				continue
			}
			switch resource.Status.State {
			case naglfarv1.ResourceUninitialized:
				requeue = true
				if resource.Status.Image == "" {
					resource.Status.Image = haproxy.BaseImageName + ct.Spec.TiDBCluster.HAProxy.Version
					resource.Status.ExposedPorts = []string{strconv.Itoa(ct.Spec.TiDBCluster.HAProxy.Port) + "/tcp"}
					var mounts []naglfarv1.TestResourceMount
					fileName := haproxy.GenerateFilePrefix(ct) + haproxy.FileName
					sourceMount := haproxy.SourceDir + fileName
					targetMount := haproxy.TargetDir + fileName
					resource.Status.Binds = []string{sourceMount + ":" + targetMount}
					resource.Status.Mounts = mounts
					resource.Status.Command = []string{"haproxy", "-f", "/usr/local/etc/haproxy/" + fileName}
					err := r.Status().Update(ctx, resource)
					if err != nil {
						return false, err
					}
				}
			case naglfarv1.ResourceReady:
				if resource.Status.Image != haproxy.BaseImageName {
					return false, fmt.Errorf("resource node %s uses an incorrect image: %s", resource.Name, resource.Status.Image)
				}
			case naglfarv1.ResourcePending, naglfarv1.ResourceFinish, naglfarv1.ResourceDestroy:
				return false, fmt.Errorf("resource node %s is in the `%s` state", resource.Name, naglfarv1.ResourceFinish)
			}
		}
	}

	return requeue, nil
}

func (r *TestClusterTopologyReconciler) initFlinkResources(ctx context.Context, ct *naglfarv1.TestClusterTopology, resources []*naglfarv1.TestResource) (bool, error) {
	var requeue bool
	exposedPortIndexer, err := indexResourceExposedPorts(ct.Spec.DeepCopy(), resources)
	if err != nil {
		return requeue, fmt.Errorf("index exposed ports failed: %v", err)
	}
	image := flink.BaseImageName + ct.Spec.FlinkCluster.Version
	for _, resource := range resources {
		switch resource.Status.State {
		case naglfarv1.ResourceUninitialized:
			requeue = true
			if resource.Status.Image == "" {
				resource.Status.Image = image
				if resource.Name == ct.Spec.FlinkCluster.JobManager.Host {
					resource.Status.Command = []string{"jobmanager"}
					resource.Status.HostName = "jobmanager"
					config := ct.Spec.FlinkCluster.JobManager.Config
					if ct.Spec.FlinkCluster.JobManager.WebPort != 0 {
						config += "\njobmanager.web.port: " + strconv.Itoa(ct.Spec.FlinkCluster.JobManager.WebPort)
					}
					config = "jobmanager.rpc.address: jobmanager\n" + config
					resource.Status.Envs = []string{"FLINK_PROPERTIES=" + config}
				} else {
					resource.Status.Command = []string{"taskmanager"}
					for i := 0; i < len(ct.Spec.FlinkCluster.TaskManager); i++ {
						if resource.Name == ct.Spec.FlinkCluster.TaskManager[i].Host {
							config := ct.Spec.FlinkCluster.TaskManager[i].Config
							config = "jobmanager.rpc.address: jobmanager\n" + config
							resource.Status.Envs = []string{"FLINK_PROPERTIES=" + config}
							break
						}
					}
				}
				resource.Status.ExposedPorts = exposedPortIndexer[resource.Name]
				err := r.Status().Update(ctx, resource)
				if err != nil {
					return false, err
				}
			}
		case naglfarv1.ResourceReady:
			if resource.Status.Image != image {
				return false, fmt.Errorf("resource node %s uses an incorrect image: %s", resource.Name, resource.Status.Image)
			}
		case naglfarv1.ResourcePending, naglfarv1.ResourceFinish, naglfarv1.ResourceDestroy:
			return false, fmt.Errorf("resource node %s is in the `%s` state", resource.Name, naglfarv1.ResourceFinish)
		}
	}
	return requeue, nil
}

func (r *TestClusterTopologyReconciler) installFlinkCluster(ctx context.Context, ct *naglfarv1.TestClusterTopology, rr *naglfarv1.TestResourceRequest) (requeue bool, err error) {
	log := r.Log.WithValues("installFlinkCluster", types.NamespacedName{
		Namespace: ct.Namespace,
		Name:      ct.Name,
	})
	var resourceList naglfarv1.TestResourceList
	var resources []*naglfarv1.TestResource
	if err := r.List(ctx, &resourceList, client.InNamespace(rr.Namespace), client.MatchingFields{resourceOwnerKey: rr.Name}); err != nil {
		log.Error(err, "unable to list child resources")
	}
	resources = filterClusterResources(ct, resourceList)

	requeue, err = r.initResource(ctx, ct, resources)
	if requeue {
		return true, err
	}
	return false, nil
}

func (r *TestClusterTopologyReconciler) installDMCluster(ctx context.Context, ct *naglfarv1.TestClusterTopology, rr *naglfarv1.TestResourceRequest) (requeue bool, err error) {
	log := r.Log.WithValues("installDMCluster", types.NamespacedName{
		Namespace: ct.Namespace,
		Name:      ct.Name,
	})
	var resourceList naglfarv1.TestResourceList
	var resources []*naglfarv1.TestResource
	if err := r.List(ctx, &resourceList, client.InNamespace(rr.Namespace), client.MatchingFields{resourceOwnerKey: rr.Name}); err != nil {
		log.Error(err, "unable to list child resources")
	}

	resources = filterClusterResources(ct, resourceList)
	requeue, err = r.initResource(ctx, ct, resources)
	if requeue {
		return true, err
	}
	tiupCtl, err := dm.MakeClusterManager(log, ct.Spec.DeepCopy(), resources, hostname2ClusterIP(resourceList))
	if err != nil {
		return false, err
	}
	return false, tiupCtl.InstallCluster(log, ct.Name, ct.Spec.DMCluster.Version)
}

func (r *TestClusterTopologyReconciler) getResourceMachine(ctx context.Context, resource *naglfarv1.TestResource) (*naglfarv1.Machine, error) {
	key := &resource.ObjectMeta
	var relation naglfarv1.Relationship
	err := r.Get(ctx, relationshipName, &relation)
	if apierrors.IsNotFound(err) {
		r.Log.Error(err, fmt.Sprintf("relationship(%s) not found", relationshipName))
		err = nil
	}
	machineRef := relation.Status.ResourceToMachine[ref.CreateRef(key).Key()]
	var machine naglfarv1.Machine
	err = r.Get(ctx, machineRef.Namespaced(), &machine)
	if apierrors.IsNotFound(err) {
		r.Log.Error(err, fmt.Sprintf("relationship(%s) not found", relationshipName))
		return nil, err
	}
	return &machine, nil
}
