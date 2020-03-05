package framework

import (
	"halkyon.io/api/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Resource is the core interface allowing users to define the behavior of primary resources. A Resource is primarily
// responsible for managing the set of its associated DependentResources and taking the appropriate actions based on their status
type Resource interface {
	v1.Object
	runtime.Object
	v1beta1.StatusAware
	// NeedsRequeue determines whether this Resource needs to be requeued in the reconcile loop
	NeedsRequeue() bool
	// ComputeStatus computes the status of this Resource based on the cluster state. Default implementation uses the
	// aggregated status of this Resource's dependents' condition. Return value indicates whether the status of the Resource has
	// changed as the result of the computation and therefore the needs to be updated on the cluster.
	ComputeStatus() (needsUpdate bool)
	// CheckValidity checks whether this Resource is valid according to its semantics. Note that some/all of this functionality
	// might be implemented as a validation webhook instead.
	CheckValidity() error
	// ProvideDefaultValues initializes any potentially missing optional values to appropriate defaults
	ProvideDefaultValues() bool
	// GetUnderlyingAPIResource returns the object implementing the custom resource this Resource represents as a
	// SerializableResource
	GetUnderlyingAPIResource() SerializableResource
	// Delete performs any operation that might be needed when a reconcile request occurs for a Resource that does not exist on
	// the cluster anymore
	Delete() error
	// CreateOrUpdate creates or updates all dependent resources associated with this Resource depending on the state of the
	//cluster
	CreateOrUpdate() error
	// NewEmpty returns a new empty instance of this Resource so that it can be populated during the reconcile loop. Note that
	// NewEmpty must return a Resource with an initialized GroupVersionKind so that calls to the GroupVersionKind method is
	// guaranteed to return a non-empty GroupVersionKind
	NewEmpty() Resource
	// InitDependentResources returns the array of DependentResources that are associated with this Resource.
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
