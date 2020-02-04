package framework

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"
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
	for _, dependent := range dependents {
		config := dependent.GetConfig()
		if config.Watched {
			if err := callback(dependent.Owner(), config.GroupVersionKind); err != nil {
				return toInit, err
			}
		}
	}
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

func (b *BaseResource) FetchUpdatedDependent(predicate Predicate) (runtime.Object, error) {
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
		fetch, err := dependent.Fetch()
		if err != nil {
			return nil, err
		}
		return fetch, nil
	default:
		return nil, fmt.Errorf("found %d dependent resources matching %s", matching, predicateDesc)
	}
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

func (b *BaseResource) ComputeStatus(current Resource) (statuses []DependentResourceStatus, notReadyWantsUpdate bool) {
	statuses = b.areDependentResourcesReady()
	msgs := make([]string, 0, len(statuses))
	for _, status := range statuses {
		if !status.Ready {
			msgs = append(msgs, fmt.Sprintf("%s => %s", status.DependentName, status.Message))
		}
	}
	if len(msgs) > 0 {
		msg := fmt.Sprintf("Waiting for the following resources: %s", strings.Join(msgs, " / "))
		LoggerFor(current.GetAsHalkyonResource()).Info(msg)
		// set the status but ignore the result since dependents are not ready, we do need to update and requeue in any case
		_ = current.SetInitialStatus(msg)
		b.SetNeedsRequeue(true)
		return statuses, true
	}

	return statuses, false
}

func (b *BaseResource) areDependentResourcesReady() (statuses []DependentResourceStatus) {
	statuses = make([]DependentResourceStatus, 0, len(b.dependents))
	for _, dependent := range b.dependents {
		config := dependent.GetConfig()
		if config.CheckedForReadiness {
			name := config.TypeName()
			fetched, err := dependent.Fetch()
			if err != nil {
				statuses = append(statuses, NewFailedDependentResourceStatus(name, err))
			} else {
				ready, message := dependent.IsReady(fetched)
				if !ready {
					statuses = append(statuses, NewFailedDependentResourceStatus(name, message))
				} else {
					statuses = append(statuses, NewReadyDependentResourceStatus(dependent.NameFrom(fetched), config.OwnerStatusField))
				}
			}
		}
	}
	return statuses
}
