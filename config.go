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
	TypeName            string
}

var defaultConfig = DependentResourceConfig{
	Watched:             true,
	Owned:               true,
	CreatedOrUpdated:    true,
	CheckedForReadiness: false,
	OwnerStatusField:    "",
}

func NewConfig(gvk schema.GroupVersionKind) DependentResourceConfig {
	return DependentResourceConfig{
		Watched:             defaultConfig.Watched,
		Owned:               defaultConfig.Owned,
		CreatedOrUpdated:    defaultConfig.CreatedOrUpdated,
		CheckedForReadiness: defaultConfig.CheckedForReadiness,
		OwnerStatusField:    defaultConfig.OwnerStatusField,
		GroupVersionKind:    gvk,
		TypeName:            gvk.Kind,
	}
}
