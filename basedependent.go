package framework

import (
	"halkyon.io/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type BaseDependentResource struct {
	config DependentResourceConfig
	owner  v1beta1.HalkyonResource
}

func NewBaseDependentResource(owner v1beta1.HalkyonResource, dependentType schema.GroupVersionKind) *BaseDependentResource {
	return NewConfiguredBaseDependentResource(owner, NewConfig(dependentType, owner.GetNamespace()))
}

func NewConfiguredBaseDependentResource(owner v1beta1.HalkyonResource, config DependentResourceConfig) *BaseDependentResource {
	return &BaseDependentResource{
		config: config,
		owner:  owner,
	}
}

func DefaultFetcher(dep DependentResource, helper *K8SHelper) (runtime.Object, error) {
	config := dep.GetConfig()
	into, err := helper.Scheme.New(config.GroupVersionKind)
	if err != nil {
		return nil, err
	}
	return helper.Fetch(dep.Name(), config.Namespace, into)
}

func DefaultDependentResourceNameFor(owner v1beta1.HalkyonResource) string {
	return owner.GetName()
}

func DefaultIsReady(_ runtime.Object) (ready bool, message string) {
	return true, ""
}

func DefaultNameFrom(dep DependentResource, _ runtime.Object) string {
	return dep.Name()
}

func (b BaseDependentResource) GetConfig() DependentResourceConfig {
	return b.config
}

func (b BaseDependentResource) Owner() v1beta1.HalkyonResource {
	return b.owner
}
