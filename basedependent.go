package framework

import (
	"fmt"
	"halkyon.io/api/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type BaseDependentResource struct {
	config DependentResourceConfig
	owner  SerializableResource
}

func NewBaseDependentResource(owner SerializableResource, dependentType schema.GroupVersionKind) *BaseDependentResource {
	return NewConfiguredBaseDependentResource(owner, NewConfig(dependentType))
}

func NewConfiguredBaseDependentResource(owner SerializableResource, config DependentResourceConfig) *BaseDependentResource {
	return &BaseDependentResource{
		config: config,
		owner:  owner,
	}
}

func DefaultFetcher(dep DependentResource) (runtime.Object, error) {
	config := dep.GetConfig()
	into, err := Helper.Scheme.New(config.GroupVersionKind)
	if err != nil {
		return nil, err
	}
	return Helper.Fetch(dep.Name(), dep.Owner().GetNamespace(), into)
}

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

func DefaultGetConditionFor(dep DependentResource, err error) *v1beta1.DependentCondition {
	return DefaultCustomizedGetConditionFor(dep, err, nil, nil)
}

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

func (b BaseDependentResource) GetConfig() DependentResourceConfig {
	return b.config
}

func (b BaseDependentResource) Owner() SerializableResource {
	return b.owner
}
