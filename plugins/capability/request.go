package capability

import (
	"halkyon.io/api/v1beta1"
	framework "halkyon.io/operator-framework"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type PluginRequest struct {
	Owner  v1beta1.HalkyonResource
	Target schema.GroupVersionKind
	Arg    *unstructured.Unstructured
	Error  error
}

func (p *PluginRequest) setArg(object runtime.Object) {
	u, ok := object.(*unstructured.Unstructured)
	if !ok {
		uns, e := framework.CreateUnstructuredObject(object, object.GetObjectKind().GroupVersionKind())
		if e != nil {
			panic(e)
		}
		u = uns.(*unstructured.Unstructured)
	}
	p.Arg = u
}

func (p *PluginRequest) getArg(object runtime.Object) runtime.Object {
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(p.Arg.Object, object)
	if err != nil {
		panic(err)
	}
	return object
}
