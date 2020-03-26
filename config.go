package framework

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DependentResourceConfig represents the configuration associated with a DependentResource. The framework takes action based on
// this configuration, for example, on whether the associated DependentResource is checked for readiness when assessing the
// status of its associated Resource or whether it needs to be watched, created or updated… The defaultConfig var records the
// default values for these who might be omitted.
type DependentResourceConfig struct {
	// Watched determines whether the operator should be notified when the associated DependentResource's state changes.
	// Defaults to true.
	Watched bool
	// Owned determines whether the Resource associated with the associated DependentResource owns this DependentResource,
	// meaning that the lifecycle of the DependentResource is tied to that of its Resource (e.g. the DependentResource is
	// deleted when the parent Resource is deleted). Defaults to true.
	Owned bool
	// Created determines whether the associated DependentResource should be created if it doesn't already exist. Generally,
	// this should be true, however, in some cases such as when a DependentResource is actually another Resource, i.e.
	// something that can (and maybe needs to) be created by a user, this should be set to false indicating that the operator
	// should wait for the associated DependentResource to be created, independently. Defaults to true.
	Created bool
	// Updated determines whether the associated DependentResource defines custom behavior to be applied when the resource
	// already exists on the cluster. Defaults to false.
	Updated bool
	// CheckedForReadiness determines whether the associated DependentResource should participate in the overall status of the
	// parent Resource, in particular when it comes to checking whether the Resource is considered ready to be used. Defaults
	// to false.
	CheckedForReadiness bool
	// GroupVersionKind records the GroupVersionKind of the associated DependentResource so that it can be used with
	// Unstructured for example.
	GroupVersionKind schema.GroupVersionKind
	// TypeName records the DependentResource's type to be displayed in messages / logs, this defaults to its associated Kind
	// but, in some instances, e.g. for Capabilities part of Component's contract, it might be needed to be overridden to be
	// more precise / specific.
	TypeName string
}

// defaultConfig records the default configuration values for these values that might be omitted.
var defaultConfig = DependentResourceConfig{
	Watched:             true,
	Owned:               true,
	Created:             true,
	Updated:             false,
	CheckedForReadiness: false,
}

// NewConfig creates a new default DependentResourceConfig for a DependentResource with the specified GroupVersionKind. All
// values are set by default as recorded in defaultConfig, TypeName is simply the GVK's Kind in this case.
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
