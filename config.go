package framework

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type DependentResourceConfig struct {
	Watched             bool
	Owned               bool
	CreatedOrUpdated    bool
	CheckedForReadiness bool
	OwnerStatusField    string
	GroupVersionKind    schema.GroupVersionKind
	Namespace           string
}

var defaultConfig = DependentResourceConfig{
	Watched:             true,
	Owned:               true,
	CreatedOrUpdated:    true,
	CheckedForReadiness: false,
	OwnerStatusField:    "",
}

func NewConfigFrom(objectType runtime.Object, owner Resource) DependentResourceConfig {
	gvk, err := apiutil.GVKForObject(objectType, owner.Helper().Scheme)
	if err != nil {
		panic(err)
	}
	return NewConfig(gvk, owner.GetNamespace())
}

func NewConfig(gvk schema.GroupVersionKind, ns string) DependentResourceConfig {
	return DependentResourceConfig{
		Watched:             defaultConfig.Watched,
		Owned:               defaultConfig.Owned,
		CreatedOrUpdated:    defaultConfig.CreatedOrUpdated,
		CheckedForReadiness: defaultConfig.CheckedForReadiness,
		OwnerStatusField:    defaultConfig.OwnerStatusField,
		GroupVersionKind:    gvk,
		Namespace:           ns,
	}
}

func (c DependentResourceConfig) TypeName() string {
	return c.GroupVersionKind.Kind
}
