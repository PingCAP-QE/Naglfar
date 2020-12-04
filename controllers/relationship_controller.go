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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/PingCAP-QE/Naglfar/pkg/ref"
)

// RelationshipReconciler reconciles a relationship object
type RelationshipReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=relationships,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=relationships/status,verbs=get;update;patch

func (r *RelationshipReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {
	ctx := context.Background()
	log := r.Log.WithValues("relationship", req.NamespacedName)

	// your logic here
	relation := new(naglfarv1.Relationship)

	if err = r.Get(ctx, req.NamespacedName, relation); err != nil {
		log.Error(err, "unable to fetch Relationship")

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		err = client.IgnoreNotFound(err)
		return
	}

	if relation.Status.MachineToResources == nil ||
		relation.Status.ResourceToMachine == nil ||
		relation.Status.AcceptedRequests == nil ||
		relation.Status.MachineLock == nil {
		if relation.Status.MachineToResources == nil {
			log.Info("initialize MachineToResources")
			relation.Status.MachineToResources = make(map[string]naglfarv1.ResourceRefList)
		}
		if relation.Status.ResourceToMachine == nil {
			log.Info("initialize ResourceToMachine")
			relation.Status.ResourceToMachine = make(map[string]naglfarv1.MachineRef)
		}
		if relation.Status.AcceptedRequests == nil {
			log.Info("initialize AcceptedRequests")
			relation.Status.AcceptedRequests = make(naglfarv1.AcceptResources, 0)
		}
		if relation.Status.MachineLock == nil {
			log.Info("initialize machineLock")
			relation.Status.MachineLock = make(map[string]ref.Ref)
		}
		err = r.Status().Update(ctx, relation)
	}

	return ctrl.Result{}, nil
}

func (r *RelationshipReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naglfarv1.Relationship{}).
		Complete(r)
}
