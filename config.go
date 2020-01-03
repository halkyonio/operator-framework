package framework

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
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
