package capability

import (
	"context"
	"halkyon.io/api/v1beta1"
	framework "halkyon.io/operator-framework"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

type PluginDependentResource struct {
	client *PluginClient
	config *framework.DependentResourceConfig
	gvk    schema.GroupVersionKind
	owner  framework.SerializableResource
	name   *string
}

var _ framework.DependentResource = &PluginDependentResource{}

func (p *PluginDependentResource) Name() string {
	if p.name == nil {
		name := ""
		p.client.call("Name", p.gvk, &name)
		p.name = &name
	}
	return *p.name
}

func (p PluginDependentResource) Owner() framework.SerializableResource {
	return p.owner
}

func (p PluginDependentResource) Fetch() (runtime.Object, error) {
	into := framework.CreateEmptyUnstructured(p.GetConfig().GroupVersionKind)
	if err := framework.Helper.Client.Get(context.TODO(), types.NamespacedName{Name: p.Name(), Namespace: p.owner.GetNamespace()}, into); err != nil {
		return nil, err
	}
	return into, nil
}

func (p PluginDependentResource) Build(_ bool) (runtime.Object, error) {
	b := &BuildResponse{}
	p.client.call("Build", p.gvk, b)
	return b.Built, nil
}

func (p PluginDependentResource) Update(toUpdate runtime.Object) (bool, runtime.Object, error) {
	res := UpdateResponse{}
	p.client.call("Update", p.gvk, &res, toUpdate)
	return res.NeedsUpdate, res.Updated, res.Error
}

func (p *PluginDependentResource) GetCondition(underlying runtime.Object, err error) (res *v1beta1.DependentCondition) {
	// we cannot serialize the error, so have to rely on default error handling
	if c := framework.ErrorDependentCondition(p, err); c != nil {
		return c
	}
	res = &v1beta1.DependentCondition{}
	p.client.call("GetCondition", p.gvk, res, underlying)
	return
}

func (p *PluginDependentResource) GetConfig() framework.DependentResourceConfig {
	if p.config == nil {
		config := &framework.DependentResourceConfig{}
		p.client.call("GetConfig", p.gvk, config)
		p.config = config
	}
	return *p.config
}
