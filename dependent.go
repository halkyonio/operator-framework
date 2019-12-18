package framework

import (
	"context"
	"halkyon.io/api/v1beta1"
	"halkyon.io/operator-framework/util"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type DependentResource interface {
	Name() string
	Fetch(helper *K8SHelper) (runtime.Object, error)
	Build() (runtime.Object, error)
	Update(toUpdate runtime.Object) (bool, error)
	Owner() v1beta1.HalkyonResource
	GetTypeName() string
	ShouldWatch() bool
	CanBeCreatedOrUpdated() bool
	CreateOrUpdate(helper *K8SHelper) error
	ShouldBeOwned() bool
	IsReady(underlying runtime.Object) (ready bool, message string)
	OwnerStatusField() string
	ShouldBeCheckedForReadiness() bool
	NameFrom(underlying runtime.Object) string
	GetGroupVersionKind() schema.GroupVersionKind
}

type DependentResourceStatus struct {
	DependentName    string
	Ready            bool
	Message          string
	OwnerStatusField string
}

func NewFailedDependentResourceStatus(dependentName string, errorOrMessage interface{}) DependentResourceStatus {
	msg := ""
	switch errorOrMessage.(type) {
	case string:
		msg = errorOrMessage.(string)
	case error:
		msg = errorOrMessage.(error).Error()
	}
	return DependentResourceStatus{DependentName: dependentName, Ready: false, Message: msg}
}

func NewReadyDependentResourceStatus(dependentName string, fieldName string) DependentResourceStatus {
	return DependentResourceStatus{DependentName: dependentName, OwnerStatusField: fieldName, Ready: true}
}

type DependentResourceHelper struct {
	_owner    Resource
	_delegate DependentResource
	apiType   runtime.Object
	gvk       *schema.GroupVersionKind
}

func (res DependentResourceHelper) GetTypeName() string {
	return util.GetObjectName(res.apiType)
}

func (res *DependentResourceHelper) GetGroupVersionKind() schema.GroupVersionKind {
	if res.gvk == nil {
		gvk, err := apiutil.GVKForObject(res.apiType, res._owner.Helper().Scheme)
		if err != nil {
			panic(err)
		}
		res.gvk = &gvk
	}
	return *res.gvk
}

func (res DependentResourceHelper) IsReady(underlying runtime.Object) (ready bool, message string) {
	return true, ""
}

func (res DependentResourceHelper) ShouldBeCheckedForReadiness() bool {
	return false
}

func (res DependentResourceHelper) OwnerStatusField() string {
	return ""
}

func (res DependentResourceHelper) NameFrom(underlying runtime.Object) string {
	return res.Name()
}

func (res DependentResourceHelper) ShouldWatch() bool {
	return true
}

func (res DependentResourceHelper) ShouldBeOwned() bool {
	return true
}

func (res DependentResourceHelper) CanBeCreatedOrUpdated() bool {
	return true
}

func NewDependentResource(resourceType runtime.Object, owner Resource) *DependentResourceHelper {
	return &DependentResourceHelper{_owner: owner, apiType: resourceType}
}

func (res *DependentResourceHelper) SetDelegate(delegate DependentResource) {
	res._delegate = delegate
}

func (res DependentResourceHelper) Name() string {
	return DefaultDependentResourceNameFor(res.Owner())
}

func (res DependentResourceHelper) Fetch(helper *K8SHelper) (runtime.Object, error) {
	delegate := res._delegate
	into, err := helper.Scheme.New(delegate.GetGroupVersionKind())
	if err != nil {
		return nil, err
	}
	return helper.Fetch(delegate.Name(), delegate.Owner().GetNamespace(), into)
}

func (res DependentResourceHelper) Owner() v1beta1.HalkyonResource {
	return res._owner
}

func (res DependentResourceHelper) OwnerAsResource() Resource {
	return res._owner
}

func DefaultDependentResourceNameFor(owner v1beta1.HalkyonResource) string {
	return owner.GetName()
}

func CreateOrUpdate(r DependentResource, helper *K8SHelper) error {
	// if the resource specifies that it shouldn't be created, exit fast
	if !r.CanBeCreatedOrUpdated() {
		return nil
	}

	kind := r.GetTypeName()
	object, err := r.Fetch(helper)
	if err != nil {
		if errors.IsNotFound(err) {
			// create the object
			obj, errBuildObject := r.Build()
			if errBuildObject != nil {
				return errBuildObject
			}

			// set controller reference if the resource should be owned
			if r.ShouldBeOwned() {
				// in most instances, resourceDefinedOwner == owner but some resources might want to return a different one
				resourceDefinedOwner := r.Owner()
				if e := controllerutil.SetControllerReference(resourceDefinedOwner, obj.(v1.Object), helper.Scheme); e != nil {
					helper.ReqLogger.Error(err, "Failed to set owner", "owner", resourceDefinedOwner, "resource", r.Name())
					return e
				}
			}

			alreadyExists := false
			if err = helper.Client.Create(context.TODO(), obj); err != nil {
				// ignore error if it's to state that obj already exists
				alreadyExists = errors.IsAlreadyExists(err)
				if !alreadyExists {
					helper.ReqLogger.Error(err, "Failed to create new ", "kind", kind)
					return err
				}
			}
			if !alreadyExists {
				helper.ReqLogger.Info("Created successfully", "kind", kind, "name", obj.(v1.Object).GetName())
			}
			return nil
		}
		helper.ReqLogger.Error(err, "Failed to get", "kind", kind)
		return err
	} else {
		// if the resource defined an updater, use it to try to update the resource
		updated, err := r.Update(object)
		if err != nil {
			return err
		}
		if updated {
			if err = helper.Client.Update(context.TODO(), object); err != nil {
				helper.ReqLogger.Error(err, "Failed to update", "kind", kind)
			}
		}
		return err
	}
}

func (res DependentResourceHelper) CreateOrUpdate(helper *K8SHelper) error {
	return CreateOrUpdate(res._delegate, helper)
}
