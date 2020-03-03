package framework

import (
	"halkyon.io/api/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Resource interface {
	v1.Object
	runtime.Object
	v1beta1.StatusAware
	NeedsRequeue() bool
	ComputeStatus() (needsUpdate bool)
	CheckValidity() error
	Init() bool
	GetUnderlyingAPIResource() SerializableResource
	PrimaryResourceType() runtime.Object
	Delete() error
	CreateOrUpdate() error
	NewEmpty() Resource
	InitDependentResources() ([]DependentResource, error)
}

// SerializableResource is the interface that resources that need to be transmitted to plugins need to implement. In particular,
// since such resources go through the Unstructured mechanism, we need to be able to know their GVK at all times.
type SerializableResource interface {
	v1.Object
	runtime.Object
	// GetGroupVersionKind returns the GroupVersionKind associated with this resource. Contrary to what could be obtained from
	// runtime.Object, this method is guaranteed to always return the GVK associated with this resource.
	GetGroupVersionKind() schema.GroupVersionKind
}
