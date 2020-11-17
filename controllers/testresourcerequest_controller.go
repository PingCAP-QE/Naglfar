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
	"strings"
	"time"

	"github.com/go-logr/logr"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
)

const requestFinalizer = "testresourcerequest.naglfar.pingcap.com"

var (
	resourceOwnerKey = ".metadata.controller"
	apiGVStr         = naglfarv1.GroupVersion.String()
)

// TestResourceRequestReconciler reconciles a TestResourceRequest object
type TestResourceRequestReconciler struct {
	client.Client
	Log     logr.Logger
	Eventer record.EventRecorder
	Scheme  *runtime.Scheme
}

// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresourcerequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresourcerequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresources/status,verbs=get

// TODO: fail
func (r *TestResourceRequestReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {
	ctx := context.Background()
	log := r.Log.WithValues("testresourcerequest", req.NamespacedName)

	var resourceRequest naglfarv1.TestResourceRequest
	if err := r.Get(ctx, req.NamespacedName, &resourceRequest); err != nil {
		log.Error(err, "unable to fetch TestResourceRequest")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if resourceRequest.ObjectMeta.DeletionTimestamp.IsZero() && !stringsContains(resourceRequest.ObjectMeta.Finalizers, requestFinalizer) {
		resourceRequest.ObjectMeta.Finalizers = append(resourceRequest.ObjectMeta.Finalizers, requestFinalizer)
		err = r.Update(ctx, &resourceRequest)
		return
	}

	if !resourceRequest.ObjectMeta.DeletionTimestamp.IsZero() && stringsContains(resourceRequest.ObjectMeta.Finalizers, requestFinalizer) {
		if err = r.removeAllResources(ctx, &resourceRequest); err != nil {
			return
		}

		resourceRequest.ObjectMeta.Finalizers = stringsRemove(resourceRequest.ObjectMeta.Finalizers, requestFinalizer)
		err = r.Update(ctx, &resourceRequest)
		return
	}

	if resourceRequest.Status.State == "" {
		resourceRequest.Status.State = naglfarv1.TestResourceRequestPending
		err = r.Status().Update(ctx, &resourceRequest)
		if err == nil {
			result.Requeue = true
		}
		return
	}

	if resourceRequest.Status.State == naglfarv1.TestResourceRequestReady {
		return
	}

	var (
		resourceMap     = make(map[string]*naglfarv1.TestResource)
		failedResources = make([]string, 0)
		requiredCount   = 0
	)

	resources, err := r.listResources(ctx, &resourceRequest)
	if err != nil {
		log.Error(err, "unable to list child resources")
		return
	}

	for idx, item := range resources.Items {
		resourceMap[item.Name] = &resources.Items[idx]

		if item.Status.State == naglfarv1.ResourceFail {
			failedResources = append(failedResources, item.Name)
		}

		if item.Status.State.IsRequired() {
			requiredCount++
		}
	}

	if len(failedResources) > 0 {
		r.Eventer.Eventf(&resourceRequest, "Warning", "ResourceFail", "resources request fails: %s", strings.Join(failedResources, ","))
		err = r.removeAllResources(ctx, &resourceRequest)
		return
	}

	// if all resources has been required, set the resource request's state to be ready
	if requiredCount == len(resourceRequest.Spec.Items) {
		log.Info("all resources are in ready state")
		resourceRequest.Status.State = naglfarv1.TestResourceRequestReady
		err = r.Status().Update(ctx, &resourceRequest)
		return
	}

	for idx, item := range resourceRequest.Spec.Items {
		if _, e := resourceMap[item.Name]; !e {
			tr := resourceRequest.ConstructTestResource(idx)
			if err := ctrl.SetControllerReference(&resourceRequest, tr, r.Scheme); err != nil {
				log.Error(err, "unable to construct a TestResource from template")
				// don't bother requeuing until we get a change to the spec
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
			}

			if err := r.Create(ctx, tr); err != nil && !apierrors.IsAlreadyExists(err) {
				log.Error(err, "unable to create a TestResource for TestResourceRequest", "testResource", tr)
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{RequeueAfter: time.Second}, nil
}

func (r *TestResourceRequestReconciler) listResources(ctx context.Context, request *naglfarv1.TestResourceRequest) (naglfarv1.TestResourceList, error) {
	var resources naglfarv1.TestResourceList
	err := r.List(ctx, &resources, client.InNamespace(request.Namespace), client.MatchingFields{resourceOwnerKey: request.Name})
	return resources, err
}

func (r *TestResourceRequestReconciler) removeAllResources(ctx context.Context, request *naglfarv1.TestResourceRequest) error {
	resources, err := r.listResources(ctx, request)
	if err != nil {
		return err
	}

	for _, resource := range resources.Items {
		err = r.Delete(ctx, &resource)
		if client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	return nil
}

func (r *TestResourceRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&naglfarv1.TestResource{}, resourceOwnerKey, func(rawObject runtime.Object) []string {
		resource := rawObject.(*naglfarv1.TestResource)
		owner := metav1.GetControllerOf(resource)
		if owner == nil {
			return nil
		}
		// make sure it's a TestResourceRequest
		if owner.APIVersion != apiGVStr || owner.Kind != "TestResourceRequest" {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&naglfarv1.TestResourceRequest{}).
		Owns(&naglfarv1.TestResource{}).
		Complete(r)
}
