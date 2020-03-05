package framework

import (
	"context"
	"halkyon.io/api/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// DependentResource represents any resource a Resource requires to be realized on the cluster.
type DependentResource interface {
	// Name returns the name used to identify this DependentResource on the cluster, given the parent Resource's namespace
	Name() string
	// Owner returns the SerializableResource owning this DependentResource. For all intent and purposes, this owner is a
	// Resource, reduced to its strictly needed information so that it can be serialized and sent over the network to plugins.
	Owner() SerializableResource
	// Fetch retrieves the object associated with this DependentResource from the cluster
	Fetch() (runtime.Object, error)
	// Build generates the runtime.Object needed to store the representation of this DependentResource on the cluster. For
	// example, a DependentResource representing a Kubernetes secret would return a Secret object as defined by the Kubernetes
	// API.
	Build(empty bool) (runtime.Object, error)
	// Update applies any needed changes to the specified runtime.Object and returns an updated version which calling code needs
	// to use since the return object might be different from the input one. The first return value is a bool indicating whether
	// or not the input object was changed in the process so that the framework can know whether to store the updated value.
	Update(toUpdate runtime.Object) (bool, runtime.Object, error)
	// GetCondition returns a DependentCondition object describing the condition of this DependentResource based either on the
	// state of the specified underlying runtime.Object (i.e. the Kubernetes resource) associated with this DependentResource or
	// the given error which might have occurred while processing this DependentResource.
	GetCondition(underlying runtime.Object, err error) *v1beta1.DependentCondition
	// GetConfig retrieves the configuration associated with this DependentResource, configuration describing how the framework
	// needs to handle this DependentResource when it comes to watching it for changes, updating it, etc.
	GetConfig() DependentResourceConfig
}

// CreateOrUpdate provides a generic implementation of the logic to create or update a DependentResource. A DependentResource is
// created if its associated configuration allows it and if a NotFound error is thrown when attempting to fetch it: its Build
// method is called and the resulting object is sent to the cluster to be created. Otherwise, if the resource is indeed fetched,
// it will be updated according to its Update method (if its configuration allows for it) and save to the cluster.
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
