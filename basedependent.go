package framework

import (
	"fmt"
	"halkyon.io/api/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// BaseDependentResource provides a default implementation for several DependentResource methods. In particular, it handles the
// DependentResource configuration and owner.
type BaseDependentResource struct {
	config DependentResourceConfig
	owner  SerializableResource
}

// NewBaseDependentResource creates a new BaseDependentResource of the specified dependent type and with the specified owner.
// The DependentResource is initialized with a default configuration as provided by NewConfig.
func NewBaseDependentResource(owner SerializableResource, dependentType schema.GroupVersionKind) *BaseDependentResource {
	return NewConfiguredBaseDependentResource(owner, NewConfig(dependentType))
}

// NewConfiguredBaseDependentResource creates a new BaseDependentResource with the specified configuration and with the
// specified owner.
func NewConfiguredBaseDependentResource(owner SerializableResource, config DependentResourceConfig) *BaseDependentResource {
	return &BaseDependentResource{
		config: config,
		owner:  owner,
	}
}

// DefaultFetcher provides a default mechanism to fetch latest Object state underlying the specified DependentResource from the
// cluster.
func DefaultFetcher(dep DependentResource) (runtime.Object, error) {
	config := dep.GetConfig()
	into, err := Helper.Scheme.New(config.GroupVersionKind)
	if err != nil {
		return nil, err
	}
	return Helper.Fetch(dep.Name(), dep.Owner().GetNamespace(), into)
}

// DefaultDependentResourceNameFor returns a default name for a DependentResource for a given owner.
func DefaultDependentResourceNameFor(owner SerializableResource) string {
	return owner.GetName()
}

// ErrorDependentCondition analyzes the error to attempt to determine the most appropriate DependentCondition to return
func ErrorDependentCondition(dep DependentResource, err error) *v1beta1.DependentCondition {
	if err != nil {
		config := dep.GetConfig()
		d := &v1beta1.DependentCondition{
			Type:          v1beta1.DependentFailed,
			DependentType: config.GroupVersionKind,
			DependentName: dep.Name(),
			Reason:        string(v1beta1.DependentFailed),
			Message:       err.Error(),
		}
		if errors.IsNotFound(err) {
			d.Type = v1beta1.DependentPending
			d.Reason = string(v1beta1.DependentPending)
			d.Message = fmt.Sprintf("%s '%s' was not found: %s", config.TypeName, d.DependentName, err.Error())
		}
		return d
	}
	return nil
}

// DefaultGetConditionFor provides generic DependentCondition creation for the specified DependentResource and given the
// (possibly nil) specified error. Simply calls DefaultCustomizedGetConditionFor function with a nil customize function.
func DefaultGetConditionFor(dep DependentResource, err error) *v1beta1.DependentCondition {
	return DefaultCustomizedGetConditionFor(dep, err, nil, nil)
}

// DefaultCustomizedGetConditionFor provides generic DependentCondition creation for the specified DependentResource and given
// the (possibly nil) specified error. The generated condition is set up so that it is using type DependentReady if a nil error
// is provided, using ErrorDependentCondition if the specified error is not nil. This generated condition can then be further
// customized with the provided customize function based on the state of the Object underlying the specified DependentResource.
// We encourage implementers to use this function to create DependentConditions for their DependentResources.
func DefaultCustomizedGetConditionFor(dep DependentResource, err error, underlying runtime.Object, customize func(underlying runtime.Object, cond *v1beta1.DependentCondition)) *v1beta1.DependentCondition {
	if c := ErrorDependentCondition(dep, err); c != nil {
		return c
	}
	d := &v1beta1.DependentCondition{
		DependentName: dep.Name(),
		DependentType: dep.GetConfig().GroupVersionKind,
		Type:          v1beta1.DependentReady,
		Reason:        string(v1beta1.DependentReady),
	}
	if customize != nil {
		customize(underlying, d)
	}
	return d
}

// GetConfig retrieves the DependentResourceConfig associated with this BaseDependentResource
func (b BaseDependentResource) GetConfig() DependentResourceConfig {
	return b.config
}

// Owner retrieves the SerializableResource owning this BaseDependentResource, i.e. of which Resource has this DependentResource
// as a dependent.
func (b BaseDependentResource) Owner() SerializableResource {
	return b.owner
}
