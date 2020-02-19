package framework

import (
	"halkyon.io/api/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Resource interface {
	v1.Object
	runtime.Object
	v1beta1.StatusAware
	NeedsRequeue() bool
	ShouldDelete() bool
	ComputeStatus() (needsUpdate bool)
	CheckValidity() error
	Init() bool
	GetAsHalkyonResource() v1beta1.HalkyonResource
	PrimaryResourceType() runtime.Object
	Delete() error
	CreateOrUpdate() error
	NewEmpty() Resource
	InitDependentResources() ([]DependentResource, error)
}
