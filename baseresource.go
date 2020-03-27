package framework

import (
	"fmt"
	"halkyon.io/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

// BaseResource provides some base behavior that can be reused when implementing the Resource interface
type BaseResource struct {
	v1beta1.StatusAware
	dependents []DependentResource
	requeue    bool
}

func (b *BaseResource) SetNeedsRequeue(requeue bool) {
	b.requeue = requeue
}

func (b *BaseResource) NeedsRequeue() bool {
	return b.requeue
}

// NewBaseResource creates a new BaseResource delegating its status to the specified StatusAware instance
func NewBaseResource(statusAware v1beta1.StatusAware) *BaseResource {
	return &BaseResource{dependents: make([]DependentResource, 0, 15), StatusAware: statusAware}
}

// CreateOrUpdateDependents calls CreateOrUpdate on the dependents of the associated BaseResource
func (b *BaseResource) CreateOrUpdateDependents() error {
	for _, dep := range b.dependents {
		if e := CreateOrUpdate(dep); e != nil {
			// wrap error so that downstream client can process the original error based on needs
			return fmt.Errorf("failed to create or update '%s' %s: %w", dep.Name(), dep.GetConfig().TypeName, e)
		}
	}
	return nil
}

// GetDependent retrieves the DependentResource associated with the specified predicate or returns an error if no such
// DependentResource exists or, conversely, if several DependentResources match the given predicate.
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

// FetchUpdatedDependent fetches the latest cluster state of the Object associated with the DependentResource identified by the
// specified predicate. As it calls GetDependent, it returns the same error conditions.
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

// ComputeStatus computes the aggregated status of this BaseResource based on the status of each DependentResource that declares
// that it needs to be checked for readiness.
func (b *BaseResource) ComputeStatus() (needsUpdate bool) {
	// todo: compute whether we need to update the resource
	status := b.GetStatus()
	for _, dependent := range b.dependents {
		config := dependent.GetConfig()
		if config.CheckedForReadiness {
			fetched, err := dependent.Fetch()
			condition := dependent.GetCondition(fetched, err)
			needsUpdate = needsUpdate || status.SetCondition(condition)
		}
	}
	if needsUpdate {
		b.SetStatus(status)
	}
	return
}

// DefaultErrorHandler updates the specified status based on the given error if needed, returning whether the status was updated
// along with the updated status to be used by calling code.
func DefaultErrorHandler(status v1beta1.Status, err error) (updated bool, updatedStatus v1beta1.Status) {
	errMsg := err.Error()
	if "Failed" != status.Reason || errMsg != status.Message {
		status.Reason = "Failed"
		status.Message = errMsg
		return true, status
	}
	return false, status
}
