package framework

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type DependentResourceConfig struct {
	Watched             bool
	Owned               bool
	Created             bool
	Updated             bool
	CheckedForReadiness bool
	GroupVersionKind    schema.GroupVersionKind
	TypeName            string
}

var defaultConfig = DependentResourceConfig{
	Watched:             true,
	Owned:               true,
	Created:             true,
	Updated:             false,
	CheckedForReadiness: false,
}

func NewConfig(gvk schema.GroupVersionKind) DependentResourceConfig {
	return DependentResourceConfig{
		Watched:             defaultConfig.Watched,
		Owned:               defaultConfig.Owned,
		Created:             defaultConfig.Created,
		Updated:             defaultConfig.Updated,
		CheckedForReadiness: defaultConfig.CheckedForReadiness,
		GroupVersionKind:    gvk,
		TypeName:            gvk.Kind,
	}
}
