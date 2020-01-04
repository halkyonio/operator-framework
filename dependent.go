package framework

import (
	"context"
	"halkyon.io/api/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type DependentResource interface {
	Name() string
	Owner() v1beta1.HalkyonResource
	NameFrom(underlying runtime.Object) string
	Fetch(helper *K8SHelper) (runtime.Object, error)
	Build(empty bool) (runtime.Object, error)
	Update(toUpdate runtime.Object) (bool, error)
	IsReady(underlying runtime.Object) (ready bool, message string)
	GetConfig() DependentResourceConfig
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

func CreateOrUpdate(r DependentResource, helper *K8SHelper) error {
	// if the resource specifies that it shouldn't be created, exit fast
	if !r.GetConfig().CreatedOrUpdated {
		return nil
	}

	kind := r.GetConfig().TypeName()
	object, err := r.Fetch(helper)
	if err != nil {
		if errors.IsNotFound(err) {
			// create the object
			obj, errBuildObject := r.Build(false)
			if errBuildObject != nil {
				return errBuildObject
			}

			// set controller reference if the resource should be owned
			if r.GetConfig().Owned {
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
