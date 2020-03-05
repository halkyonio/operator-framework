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
	Owner() SerializableResource
	Fetch() (runtime.Object, error)
	Build(empty bool) (runtime.Object, error)
	Update(toUpdate runtime.Object) (bool, runtime.Object, error)
	GetCondition(underlying runtime.Object, err error) *v1beta1.DependentCondition
	GetConfig() DependentResourceConfig
}

func CreateOrUpdate(r DependentResource) error {
	// if the resource specifies that it shouldn't be created, exit fast
	config := r.GetConfig()
	if !config.Created && !config.Updated {
		return nil
	}

	kind := config.TypeName
	object, err := r.Fetch()
	logger := LoggerFor(r.Owner())
	if err != nil {
		if config.Created && errors.IsNotFound(err) {
			// create the object
			obj, errBuildObject := r.Build(false)
			if errBuildObject != nil {
				return errBuildObject
			}

			// set controller reference if the resource should be owned
			if config.Owned {
				// in most instances, resourceDefinedOwner == owner but some resources might want to return a different one
				resourceDefinedOwner := r.Owner()
				if e := controllerutil.SetControllerReference(resourceDefinedOwner, obj.(v1.Object), Helper.Scheme); e != nil {
					logger.Error(err, "Failed to set owner", "owner", resourceDefinedOwner, "resource", r.Name())
					return e
				}
			}

			alreadyExists := false
			if err = Helper.Client.Create(context.TODO(), obj); err != nil {
				// ignore error if it's to state that obj already exists
				alreadyExists = errors.IsAlreadyExists(err)
				if !alreadyExists {
					logger.Error(err, "Failed to create new ", "kind", kind)
					return err
				}
			}
			if !alreadyExists {
				logger.Info("Created successfully", "kind", kind, "name", obj.(v1.Object).GetName())
			}
			return nil
		}
		logger.Error(err, "Failed to get", "kind", kind)
		return err
	} else {
		if config.Updated {
			// if the resource defined an updater, use it to try to update the resource
			updated, toUpdate, err := r.Update(object)
			if err != nil {
				return err
			}
			if updated {
				if err = Helper.Client.Update(context.TODO(), toUpdate); err != nil {
					logger.Error(err, "Failed to update", "kind", kind)
				}
				logger.Info("Updated successfully", "kind", kind, "name", object.(v1.Object).GetName())
			}
			return err
		}
		return nil
	}
}
