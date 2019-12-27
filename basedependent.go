package framework

import (
	"halkyon.io/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

type BaseDependentResource struct {
	config DependentResourceConfig
	owner  v1beta1.HalkyonResource
}

func NewBaseDependentResource(objectType runtime.Object, owner Resource) *BaseDependentResource {
	config := NewConfigFrom(objectType, owner)
	return &BaseDependentResource{
		config: config,
		owner:  owner.GetAsHalkyonResource(),
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
