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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
)

var (
	resourceOwnerKey = ".metadata.controller"
	apiGVStr         = naglfarv1.GroupVersion.String()
)

// TestResourceRequestReconciler reconciles a TestResourceRequest object
type TestResourceRequestReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresourcerequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresourcerequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresources/status,verbs=get

func (r *TestResourceRequestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("testresourcerequest", req.NamespacedName)
	testResourceNameFormat := "%s-%s"

	var resourceRequest naglfarv1.TestResourceRequest
	if err := r.Get(ctx, req.NamespacedName, &resourceRequest); err != nil {
		log.Error(err, "unable to fetch TestResourceRequest")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	var resources naglfarv1.TestResourceList
	var resourceMap map[string]*naglfarv1.TestResource
	if err := r.List(ctx, &resources, client.InNamespace(req.Namespace), client.MatchingFields{resourceOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list child resources")
	}
	for idx, item := range resources.Items {
		resourceMap[item.Name] = &resources.Items[idx]
	}
	readyCount := 0
	if readyCount == len(resourceRequest.Spec.Items) {
		log.Info("all resources are in ready state")
		resourceRequest.Status.State = naglfarv1.TestResourceRequestReady
		if err := r.Status().Update(ctx, &resourceRequest); err != nil {
			log.Error(err, "unable to update TestResourceRequest status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	constructTestResource := func(resourceRequest *naglfarv1.TestResourceRequest, idx int) (*naglfarv1.TestResource, error) {
		name := fmt.Sprintf("%s-%s", resourceRequest.Name, resourceRequest.Spec.Items[idx].Name)
		tr := &naglfarv1.TestResource{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Name:        name,
				Namespace:   resourceRequest.Namespace,
			},
			Spec: *resourceRequest.Spec.Items[idx].Spec.DeepCopy(),
		}

		if err := ctrl.SetControllerReference(resourceRequest, tr, r.Scheme); err != nil {
			return nil, err
		}
		return tr, nil
	}
	for idx, item := range resourceRequest.Spec.Items {
		if _, e := resourceMap[fmt.Sprintf(testResourceNameFormat, resourceRequest.Name, item.Name)]; !e {
			tr, err := constructTestResource(&resourceRequest, idx)
			if err != nil {
				log.Error(err, "unable to construct a TestResource from template")
				// don't bother requeuing until we get a change to the spec
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
			}
			if err := r.Create(ctx, tr); err != nil {
				log.Error(err, "unable to create a TestResource for TestResourceRequest", "testResource", tr)
				return ctrl.Result{}, err
			}
			log.V(1).Info("create a TestResource", "testResource", tr)
		}
	}
	return ctrl.Result{}, nil
}

func (r *TestResourceRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//if err := mgr.GetFieldIndexer().IndexField(&naglfarv1.TestResource{}, resourceOwnerKey, func(rawObject runtime.Object) []string {
	//	resource := rawObject.(*naglfarv1.TestResource)
	//	owner := metav1.GetControllerOf(resource)
	//	if owner == nil {
	//		return nil
	//	}
	//	// make sure it's a TestResourceRequest
	//	if owner.APIVersion != apiGVStr || owner.Kind != "TestResourceRequest" {
	//		return nil
	//	}
	//	return []string{owner.Name}
	//}); err != nil {
	//	return err
	//}
	return ctrl.NewControllerManagedBy(mgr).
		For(&naglfarv1.TestResourceRequest{}).
		//Owns(&naglfarv1.TestResource{}).
		Complete(r)
}
