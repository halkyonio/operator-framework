package framework

import (
	"fmt"
	"halkyon.io/api/v1beta1"
	"halkyon.io/operator-framework/util"
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

func HasChangedFromStatusUpdate(status interface{}, statuses []DependentResourceStatus, msg string) (changed bool, updatedMsg string) {
	updatedMsg = msg
	for _, s := range statuses {
		changed = changed || util.MustSetNamedStringField(status, s.OwnerStatusField, s.DependentName)
		if changed {
			updatedMsg = fmt.Sprintf("%s: '%s' changed to '%s'", msg, s.OwnerStatusField, s.DependentName)
		}
	}
	return changed, updatedMsg
}
