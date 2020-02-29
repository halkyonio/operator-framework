package framework

import (
	"fmt"
	"halkyon.io/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
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
			// wrap error so that downstream client can process the original error based on needs
			return fmt.Errorf("failed to create or update '%s' %s: %w", dep.Name(), dep.GetConfig().TypeName, e)
		}
	}
	return nil
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
			condition := dependent.GetCondition(fetched, err)
			needsUpdate = needsUpdate || status.SetCondition(condition)
		}
	}
	if needsUpdate {
		current.SetStatus(status)
	}
	return
}

func DefaultErrorHandler(status v1beta1.Status, err error) (updated bool, updatedStatus v1beta1.Status) {
	errMsg := err.Error()
	if "Failed" != status.Reason || errMsg != status.Message {
		status.Reason = "Failed"
		status.Message = errMsg
		return true, status
	}
	return false, status
}
