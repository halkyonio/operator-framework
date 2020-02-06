package framework

import (
	"fmt"
	"halkyon.io/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type BaseResource struct {
	dependents          []DependentResource
	requeue             bool
	primaryResourceType runtime.Object
}

func (b *BaseResource) PrimaryResourceType() runtime.Object {
	return b.primaryResourceType
}

func (b *BaseResource) SetNeedsRequeue(requeue bool) {
	b.requeue = requeue
}

func (b *BaseResource) NeedsRequeue() bool {
	return b.requeue
}

func NewHasDependents(primary runtime.Object) *BaseResource {
	return &BaseResource{dependents: make([]DependentResource, 0, 15), primaryResourceType: primary}
}

func (b *BaseResource) CreateOrUpdateDependents() error {
	for _, dep := range b.dependents {
		if e := CreateOrUpdate(dep); e != nil {
			return fmt.Errorf("failed to create or update '%s' %s: %s", dep.Name(), dep.GetConfig().TypeName(), e.Error())
		}
	}
	return nil
}

func FetchAndInitNewResource(name string, namespace string, toInit Resource, callback WatchCallback) (Resource, error) {
	toInit.SetName(name)
	toInit.SetNamespace(namespace)
	_, err := Helper.Fetch(name, namespace, toInit.GetAsHalkyonResource())
	if err != nil {
		return toInit, err
	}
	dependents, err := toInit.InitDependentResources()
	if err != nil {
		return toInit, err
	}

	// init status if needed
	status := toInit.GetStatus()
	if len(status.Conditions) == 0 {
		status.Conditions = make([]v1beta1.DependentCondition, 0, len(dependents))
	}
	for _, dependent := range dependents {
		// add watch if needed
		config := dependent.GetConfig()
		if config.Watched {
			if err := callback(dependent.Owner(), config.GroupVersionKind); err != nil {
				return toInit, err
			}
		}
		// init associated status condition if needed
		_ = status.GetConditionFor(dependent.Name(), config.GroupVersionKind)
	}
	toInit.SetStatus(status)
	return toInit, err
}

type Predicate interface {
	Matches(resource DependentResource) bool
}

type TypePredicate struct {
	gvk  schema.GroupVersionKind
	desc string
}

func (tp TypePredicate) Matches(resource DependentResource) bool {
	return resource.GetConfig().GroupVersionKind == tp.gvk
}

func (tp TypePredicate) String() string {
	return tp.desc
}

func TypePredicateFor(gvk schema.GroupVersionKind) Predicate {
	return TypePredicate{
		gvk:  gvk,
		desc: fmt.Sprintf("GetConfig().GroupVersionKind == %v", gvk),
	}
}

func (b *BaseResource) GetDependent(predicate Predicate) (DependentResource, error) {
	var dependent DependentResource
	matching := 0
	for _, d := range b.dependents {
		if predicate.Matches(d) {
			dependent = d
			matching++
		}
	}
	predicateDesc := "predicate"
	if stringer, ok := predicate.(fmt.Stringer); ok {
		predicateDesc = stringer.String()
	}

	switch matching {
	case 0:
		return nil, fmt.Errorf("couldn't find any dependent resource matching %s", predicateDesc)
	case 1:
		return dependent, nil
	default:
		return nil, fmt.Errorf("found %d dependent resources matching %s", matching, predicateDesc)
	}
}

func (b *BaseResource) FetchUpdatedDependent(predicate Predicate) (runtime.Object, error) {
	dependent, err := b.GetDependent(predicate)
	if err != nil {
		return nil, err
	}
	return dependent.Fetch()
}

// AddDependentResource adds dependent resources to this base resource, keeping the order in which they are added, it is
// therefore possible to create dependent resources in a specific order since they are created in the same order as specified here
func (b *BaseResource) AddDependentResource(resources ...DependentResource) []DependentResource {
	for _, dependent := range resources {
		if dependent.Owner() == nil {
			panic(fmt.Errorf("dependent resource %s must have an owner", dependent.Name()))
		}
		b.dependents = append(b.dependents, dependent)
	}
	return b.dependents
}

func (b *BaseResource) ComputeStatus(current Resource) (needsUpdate bool) {
	// todo: compute whether we need to update the resource
	status := current.GetStatus()
	for _, dependent := range b.dependents {
		config := dependent.GetConfig()
		if config.CheckedForReadiness {
			fetched, err := dependent.Fetch()
			condition := status.GetConditionFor(dependent.Name(), config.GroupVersionKind)
			if err != nil {
				needsUpdate = needsUpdate || status.SetCondition(condition, v1beta1.DependentFailed, err.Error())
			} else {
				ready, message := dependent.IsReady(fetched)
				conditionType := v1beta1.DependentPending
				if ready {
					conditionType = v1beta1.DependentReady
				}
				needsUpdate = needsUpdate || status.SetCondition(condition, conditionType, message)
			}
		}
	}
	current.SetStatus(status)
	return
}
