package framework

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"halkyon.io/api/v1beta1"
	"halkyon.io/operator-framework/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
)

func NewGenericReconciler(resource Resource) *GenericReconciler {
	return &GenericReconciler{resource: resource}
}

type GenericReconciler struct {
	resource Resource
}

func (b *GenericReconciler) logger() logr.Logger {
	return LoggerFor(b.resource.GetAsHalkyonResource())
}

func (b *GenericReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	b.logger().WithValues("namespace", request.Namespace)

	// Fetch the primary resource
	resource, err := FetchAndInitNewResource(request.Name, request.Namespace, b.resource.NewEmpty(), getCallbackFor(b.resource))
	typeName := util.GetObjectName(b.resource.PrimaryResourceType())
	if err != nil {
		if errors.IsNotFound(err) {
			// Return and don't create
			if resource.ShouldDelete() {
				b.logger().Info("'" + request.Name + "' " + typeName + " is marked for deletion. Running clean-up.")
				err := resource.Delete()
				return reconcile.Result{Requeue: resource.NeedsRequeue()}, err
			}
			return reconcile.Result{}, nil
		}
		// Error reading the object - create the request.
		b.logger().Error(err, "failed to initialize '"+request.Name+"' "+typeName)
		if resource != nil {
			b.updateStatusIfNeeded(resource, err)
			return reconcile.Result{Requeue: false}, nil
		}
		return reconcile.Result{}, err
	}

	initialStatus := resource.GetStatusAsString()
	if resource.GetGeneration() == 1 && len(initialStatus) == 0 {
		resource.SetInitialStatus("Initializing")
	}

	if resource.Init() {
		if e := Helper.Client.Update(context.Background(), resource.GetAsHalkyonResource()); e != nil {
			b.logger().Error(e, fmt.Sprintf("failed to update '%s' %s", resource.GetName(), typeName))
		}
		return reconcile.Result{}, nil
	}

	if err := resource.CheckValidity(); err != nil {
		b.updateStatusIfNeeded(resource, err)
		return reconcile.Result{}, nil
	}

	b.logger().Info("-> "+typeName, "name", resource.GetName(), "status", initialStatus)

	err = resource.CreateOrUpdate()

	// always check status for updates
	b.updateStatusIfNeeded(resource, err)

	requeue := resource.NeedsRequeue()

	// only log exit if status changed to avoid being too verbose
	newStatus := resource.GetStatusAsString()
	if newStatus != initialStatus {
		msg := "<- " + typeName
		if requeue {
			msg += " (requeued)"
		}
		b.logger().Info(msg, "name", resource.GetName(), "status", newStatus)
	}
	return reconcile.Result{Requeue: requeue}, nil
}

func (b *GenericReconciler) updateStatusIfNeeded(instance Resource, err error) {
	// update the resource if the status has changed
	updateStatus := false
	if err == nil {
		updateStatus = instance.ComputeStatus()
	} else {
		updateStatus = instance.SetErrorStatus(err)
		b.logger().Error(err, fmt.Sprintf("'%s' %s has an error", instance.GetName(), util.GetObjectName(instance.GetAsHalkyonResource())))
	}
	if updateStatus {
		object := instance.GetAsHalkyonResource()
		if e := Helper.Client.Status().Update(context.Background(), object); e != nil {
			b.logger().Error(e, fmt.Sprintf("failed to update status for '%s' %s", instance.GetName(), util.GetObjectName(object)))
		}
	}
}

func RegisterNewReconciler(resource Resource, mgr manager.Manager) error {
	resourceType := resource.PrimaryResourceType()

	// initialize the GVK for this resource type
	gvk, err := apiutil.GVKForObject(resourceType, mgr.GetScheme())
	if err != nil {
		return err
	}
	resourceType.GetObjectKind().SetGroupVersionKind(gvk)

	// Create a new controller
	controllerName := controllerNameFor(resourceType)
	reconciler := NewGenericReconciler(resource)
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: reconciler})
	if err != nil {
		return err
	}

	// Register logger
	registerLogger(controllerName)

	// Watch for changes to primary resource
	if err = c.Watch(&source.Kind{Type: resourceType}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// Create callback for dependent resources to add themselves as watched resources
	callbacks[controllerName] = createCallbackFor(c)

	return nil
}

type WatchCallback func(owner v1beta1.HalkyonResource, dependentGVK schema.GroupVersionKind) error

var callbacks = make(map[string]WatchCallback, 7)

// record which gvks we're already watching to not register another watch again
var watched = make(map[schema.GroupVersionKind]bool, 21)

func createCallbackFor(c controller.Controller) WatchCallback {
	return func(resource v1beta1.HalkyonResource, dependentGVK schema.GroupVersionKind) error {
		// if we're not already watching this secondary resource
		if !watched[dependentGVK] {
			// watch it
			owner := &handler.EnqueueRequestForOwner{
				IsController: true,
				OwnerType:    CreateEmptyUnstructured(resource.GetGroupVersionKind()),
			}
			if err := c.Watch(createSourceForGVK(dependentGVK), owner); err != nil {
				return err
			}
			watched[dependentGVK] = true
		}
		return nil
	}
}

func getCallbackFor(resource Resource) WatchCallback {
	return callbacks[controllerNameFor(resource.PrimaryResourceType())]
}

func controllerNameFor(resource runtime.Object) string {
	return strings.ToLower(util.GetObjectName(resource)) + "-controller"
}

func CreateUnstructuredObject(from runtime.Object, gvk schema.GroupVersionKind) (runtime.Object, error) {
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(from)
	if err != nil {
		return nil, err
	}
	obj := &unstructured.Unstructured{Object: u}
	obj.SetGroupVersionKind(gvk)
	return obj, nil
}

// createSourceForGVK creates a *source.Kind for the given gvk.
func createSourceForGVK(gvk schema.GroupVersionKind) *source.Kind {
	return &source.Kind{Type: CreateEmptyUnstructured(gvk)}
}

func CreateEmptyUnstructured(gvk schema.GroupVersionKind) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)
	return u
}
