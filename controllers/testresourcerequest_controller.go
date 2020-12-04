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
	"math/rand"
	"sort"
	"time"

	"github.com/go-logr/logr"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/PingCAP-QE/Naglfar/pkg/ref"
)

const requestFinalizer = "testresourcerequest.naglfar.pingcap.com"

var (
	resourceOwnerKey = ".metadata.controller"
	apiGVStr         = naglfarv1.GroupVersion.String()
)

// TestResourceRequestReconciler reconciles a TestResourceRequest object
type TestResourceRequestReconciler struct {
	client.Client
	Ctx     context.Context
	Log     logr.Logger
	Eventer record.EventRecorder
	Scheme  *runtime.Scheme
}

// ItemGroup groups item according to machine name
type ItemGroup struct {
	machine string
	items   []*naglfarv1.TestResource
}

// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresourcerequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresourcerequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=naglfar.pingcap.com,resources=testresources/status,verbs=get

// TODO: fail
func (r *TestResourceRequestReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {
	r.Ctx = context.Background()
	log := r.Log.WithValues("testresourcerequest", req.NamespacedName)

	var resourceRequest naglfarv1.TestResourceRequest
	if err := r.Get(r.Ctx, req.NamespacedName, &resourceRequest); err != nil {
		log.Error(err, "unable to fetch TestResourceRequest")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if resourceRequest.ObjectMeta.DeletionTimestamp.IsZero() && !stringsContains(resourceRequest.ObjectMeta.Finalizers, requestFinalizer) {
		resourceRequest.ObjectMeta.Finalizers = append(resourceRequest.ObjectMeta.Finalizers, requestFinalizer)
		err = r.Update(r.Ctx, &resourceRequest)
		return
	}

	if !resourceRequest.ObjectMeta.DeletionTimestamp.IsZero() && stringsContains(resourceRequest.ObjectMeta.Finalizers, requestFinalizer) {
		return r.reconcileDeletion(&resourceRequest)
	}

	switch resourceRequest.Status.State {
	case "":
		resourceRequest.Status.State = naglfarv1.TestResourceRequestPending
		err = r.Status().Update(r.Ctx, &resourceRequest)
		if err == nil {
			result.Requeue = true
		}
		return
	case naglfarv1.TestResourceRequestPending:
		return r.reconcilePending(&resourceRequest)
	case naglfarv1.TestResourceRequestReady:
		return r.reconcileReady(&resourceRequest)
	default:
		return ctrl.Result{}, nil
	}
}

func (r *TestResourceRequestReconciler) reconcilePending(request *naglfarv1.TestResourceRequest) (ctrl.Result, error) {
	var (
		resourceMap   = make(map[string]*naglfarv1.TestResource)
		requiredCount = 0
	)
	resources, err := r.listResources(r.Ctx, request)
	if err != nil {
		return ctrl.Result{}, err
	}
	for idx, item := range resources.Items {
		resourceOwner := metav1.GetControllerOf(&item)
		// filter out resources that not owned by this request
		if resourceOwner == nil || resourceOwner.UID != request.UID {
			r.Eventer.Eventf(request, "Warning", "Request", "there exists old resource named: %s", item.Name)
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		resourceMap[item.Name] = &resources.Items[idx]
		// Because the deletion is async, here need to decide whether the resource is owned by this request
		if item.Status.State.IsRequired() {
			requiredCount++
		}
	}
	// build uncreated resources
	if len(resourceMap) != len(request.Spec.Items) {
		for idx, item := range request.Spec.Items {
			if _, e := resourceMap[item.Name]; !e {
				tr := request.ConstructTestResource(idx)
				if err := ctrl.SetControllerReference(request, tr, r.Scheme); err != nil {
					r.Log.Error(err, "unable to construct a TestResource from template")
					// don't bother requeuing until we get a change to the spec
					return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
				}
				if err := r.Create(r.Ctx, tr); err != nil && !apierrors.IsAlreadyExists(err) {
					r.Log.Error(err, "unable to create a TestResource for TestResourceRequest", "testResource", tr)
					return ctrl.Result{}, err
				}
			}
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// if all resources has been required, set the resource request's state to be ready
	if requiredCount == len(request.Spec.Items) {
		r.Log.Info("all resources are in ready state")
		request.Status.State = naglfarv1.TestResourceRequestReady
		err = r.Status().Update(r.Ctx, request)
		return ctrl.Result{}, nil
	}

	rel, err := r.getRelationship()
	if err != nil {
		return ctrl.Result{}, err
	}
	selfRef := ref.CreateRef(&request.ObjectMeta)
	// check whether there already exists a record in relationship's accepted request list
	// if not, try to submit a request to relationship reconcile
	if !rel.Status.AcceptedRequests.IfExist(selfRef) {
		return r.tryRequest(rel, request, resourceMap)
	}
	return ctrl.Result{RequeueAfter: time.Second}, nil
}

func (r *TestResourceRequestReconciler) reconcileReady(request *naglfarv1.TestResourceRequest) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// 1. remove resources
// 2. remove self from accept requests list
// 3. unlock machines
// 4. remove finalizer
func (r *TestResourceRequestReconciler) reconcileDeletion(resourceRequest *naglfarv1.TestResourceRequest) (ctrl.Result, error) {
	resources, err := r.listResources(r.Ctx, resourceRequest)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err = r.removeAllResources(r.Ctx, resources); err != nil {
		return ctrl.Result{}, err
	}
	if len(resources.Items) != 0 {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	rel, err := r.getRelationship()
	if err != nil {
		return ctrl.Result{}, err
	}
	// remove accept requests list
	rel.Status.AcceptedRequests.Remove(ref.CreateRef(&resourceRequest.ObjectMeta))

	selfRef := ref.CreateRef(&resourceRequest.ObjectMeta)
	toCleanMachineKeys := make([]string, 0)
	for machine, requestRef := range rel.Status.MachineLocks {
		if requestRef == selfRef {
			toCleanMachineKeys = append(toCleanMachineKeys, machine)
		}
	}
	for _, machineKey := range toCleanMachineKeys {
		delete(rel.Status.MachineLocks, machineKey)
	}
	err = r.Status().Update(r.Ctx, rel)
	if err != nil {
		return ctrl.Result{}, err
	}
	resourceRequest.ObjectMeta.Finalizers = stringsRemove(resourceRequest.ObjectMeta.Finalizers, requestFinalizer)
	err = r.Update(r.Ctx, resourceRequest)
	return ctrl.Result{}, err
}

func (r *TestResourceRequestReconciler) tryRequest(
	rel *naglfarv1.Relationship,
	request *naglfarv1.TestResourceRequest,
	resourceMap map[string]*naglfarv1.TestResource) (ctrl.Result, error) {
	// record exclusive machines
	toLockMachines := make(map[string]*naglfarv1.Machine)
	machines, err := r.getMachines()
	if err != nil {
		return ctrl.Result{}, err
	}
	machineAliasNameMap := make(map[string]*naglfarv1.MachineRequest)
	requeueResult := ctrl.Result{RequeueAfter: 10 * time.Second}

	// wait all exclusive machines idle
	for idx, machineRequest := range request.Spec.Machines {
		if machineRequest.TestMachineResource != "" {
			machine, ok := machines[machineRequest.TestMachineResource]
			if !ok {
				return ctrl.Result{}, fmt.Errorf("no ready machine %s", machineRequest.TestMachineResource)
			}
			// if the request machine is already locked by another request, requeue
			if lockRequest, exist := rel.Status.MachineLocks[machineRequest.TestMachineResource]; exist {
				r.Eventer.Eventf(request, "Warning", "Request",
					"the machine %s is locked by another request %s",
					machineRequest.TestMachineResource,
					lockRequest.Key())
				return requeueResult, nil
			}
			toLockMachines[machineRequest.TestMachineResource] = machine
			machineAliasNameMap[machineRequest.Name] = request.Spec.Machines[idx]
		}
	}
	// find suit nodes for request items
	nonLockMachines := r.selectNonLockMachines(machines, rel)
	itemGroups := r.groupCoLocatedItems(request.Spec.Items, resourceMap)
	sortedMachineList := r.sortMachines(nonLockMachines, toLockMachines, rel)
	for _, itemGroup := range itemGroups {
		candidateMachineList := sortedMachineList
		if itemGroup.machine != "" {
			machineRequest := machineAliasNameMap[itemGroup.machine]
			if machineRequest.TestMachineResource != "" {
				candidateMachineList = []*naglfarv1.Machine{nonLockMachines[machineRequest.TestMachineResource]}
			}
			if machineRequest.Exclusive {
				candidateMachineList = r.filterMachines(candidateMachineList, func(m *naglfarv1.Machine) bool {
					return len(rel.Status.MachineToResources[ref.CreateRef(&m.ObjectMeta).Key()]) == 0
				})
			}
		}
		if !r.tryBindResource(rel, candidateMachineList, itemGroup) {
			r.Eventer.Eventf(request, "Warning", "Request", "fail to find suitable machines")
			return requeueResult, nil
		}
	}
	for _, item := range toLockMachines {
		rel.Status.MachineLocks[item.Name] = ref.CreateRef(&request.ObjectMeta)
	}
	err = r.Status().Update(r.Ctx, rel)
	return ctrl.Result{}, err
}

func (r *TestResourceRequestReconciler) tryBindResource(rel *naglfarv1.Relationship, candidateMachines []*naglfarv1.Machine, group ItemGroup) bool {
FindMachines:
	for _, machine := range candidateMachines {
		resources := rel.Status.MachineToResources[ref.CreateRef(&machine.ObjectMeta).Key()]
		bindings := make(map[string]*naglfarv1.ResourceBinding)
		for _, item := range group.items {
			binding, overflow := r.resourceOverflow(machine.Rest(resources), &item.Spec)
			if overflow {
				continue FindMachines
			}
			resources = append(resources, naglfarv1.ResourceRef{
				Ref:     ref.CreateRef(&item.ObjectMeta),
				Binding: *binding,
			})
			bindings[item.Name] = binding
		}
		rel.Status.MachineToResources[ref.CreateRef(&machine.ObjectMeta).Key()] = resources
		for _, item := range group.items {
			rel.Status.ResourceToMachine[ref.CreateRef(&item.ObjectMeta).Key()] = naglfarv1.MachineRef{
				Ref:     ref.CreateRef(&machine.ObjectMeta),
				Binding: *bindings[item.Name],
			}
		}
		return true
	}
	return false
}

func (r *TestResourceRequestReconciler) resourceOverflow(rest *naglfarv1.AvailableResource, newResource *naglfarv1.TestResourceSpec) (*naglfarv1.ResourceBinding, bool) {
	if rest == nil {
		return nil, true
	}

	binding := &naglfarv1.ResourceBinding{
		CPUSet: pickCpuSet(rest.IdleCPUSet, newResource.Cores),
		Memory: newResource.Memory,
		Disks:  make(map[string]naglfarv1.DiskBinding),
	}

	for name, disk := range newResource.Disks {
		for device, diskResource := range rest.Disks {
			if disk.Kind == diskResource.Kind &&
				disk.Size.Unwrap() <= diskResource.Size.Unwrap() {

				delete(rest.Disks, name)
				binding.Disks[name] = naglfarv1.DiskBinding{
					Kind:       disk.Kind,
					Size:       diskResource.Size,
					Device:     device,
					OriginPath: diskResource.MountPath,
					MountPath:  disk.MountPath,
				}

				break
			}
		}
	}

	overflow := len(binding.Disks) < len(newResource.Disks) ||
		rest.Memory.Unwrap() < newResource.Memory.Unwrap() ||
		len(binding.CPUSet) < int(newResource.Cores)

	if overflow {
		return nil, overflow
	}

	return binding, false
}

// groupCoLocatedItems groups request items, and put explicit specified machines to front
func (r *TestResourceRequestReconciler) groupCoLocatedItems(
	items []*naglfarv1.ResourceRequestItem,
	resourceMap map[string]*naglfarv1.TestResource) []ItemGroup {

	namedMachineItem := make(map[string][]*naglfarv1.TestResource)
	noNamedMachineItemGroup := make([]ItemGroup, 0)
	result := make([]ItemGroup, 0)

	for _, item := range items {
		if item.Spec.TestMachineResource != "" {
			namedMachineItem[item.Spec.TestMachineResource] = append(namedMachineItem[item.Spec.TestMachineResource],
				resourceMap[item.Name])
		} else {
			noNamedMachineItemGroup = append(noNamedMachineItemGroup, ItemGroup{
				items: []*naglfarv1.TestResource{resourceMap[item.Name]},
			})
		}
	}
	for machine := range namedMachineItem {
		result = append(result, ItemGroup{
			machine: machine,
			items:   namedMachineItem[machine],
		})
	}
	return append(result, noNamedMachineItemGroup...)
}

func (r *TestResourceRequestReconciler) sortMachines(machines map[string]*naglfarv1.Machine,
	toLockMachine map[string]*naglfarv1.Machine,
	rel *naglfarv1.Relationship) []*naglfarv1.Machine {
	result := make([]*naglfarv1.Machine, 0)
	for idx := range machines {
		result = append(result, machines[idx])
	}
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	sort.Slice(result, func(i, j int) bool {
		m1 := result[i]
		m2 := result[j]
		_, m1ok := toLockMachine[m1.Name]
		_, m2ok := toLockMachine[m2.Name]
		if m1ok && !m2ok {
			return true
		}
		if !m1ok && m2ok {
			return false
		}
		return len(rel.Status.MachineToResources[ref.CreateRef(&m1.ObjectMeta).Key()]) > len(rel.Status.MachineToResources[ref.CreateRef(&m2.ObjectMeta).Key()])
	})
	return result
}

func (r *TestResourceRequestReconciler) selectNonLockMachines(machines map[string]*naglfarv1.Machine, relations *naglfarv1.Relationship) map[string]*naglfarv1.Machine {
	result := make(map[string]*naglfarv1.Machine)
	for key := range machines {
		if _, exist := relations.Status.MachineLocks[key]; !exist {
			result[key] = machines[key]
		}
	}
	return result
}

func (r *TestResourceRequestReconciler) filterMachines(machines []*naglfarv1.Machine, pred func(m *naglfarv1.Machine) bool) []*naglfarv1.Machine {
	result := make([]*naglfarv1.Machine, 0)
	for idx, machine := range machines {
		if pred(machine) {
			result = append(result, machines[idx])
		}
	}
	return result
}

func (r *TestResourceRequestReconciler) getMachines() (map[string]*naglfarv1.Machine, error) {
	var machineList naglfarv1.MachineList
	if err := r.List(r.Ctx, &machineList); err != nil {
		return nil, err
	}
	book := make(map[string]*naglfarv1.Machine)
	for idx, machine := range machineList.Items {
		book[machine.Name] = &machineList.Items[idx]
	}
	return book, nil
}

func (r *TestResourceRequestReconciler) getRelationship() (*naglfarv1.Relationship, error) {
	var relation naglfarv1.Relationship
	err := r.Get(r.Ctx, relationshipName, &relation)
	if apierrors.IsNotFound(err) {
		r.Log.Error(err, fmt.Sprintf("relationship(%s) not found", relationshipName))
		err = nil
	}
	return &relation, err
}

func (r *TestResourceRequestReconciler) listResources(ctx context.Context, request *naglfarv1.TestResourceRequest) (naglfarv1.TestResourceList, error) {
	var resources naglfarv1.TestResourceList
	err := r.List(ctx, &resources, client.InNamespace(request.Namespace), client.MatchingFields{resourceOwnerKey: request.Name})
	return resources, err
}

func (r *TestResourceRequestReconciler) removeAllResources(ctx context.Context, resourceList naglfarv1.TestResourceList) error {
	for _, resource := range resourceList.Items {
		err := r.Delete(ctx, &resource)
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

func init() {
	rand.Seed(time.Now().Unix())
}
