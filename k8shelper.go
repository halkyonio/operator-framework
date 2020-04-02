package framework

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"halkyon.io/operator-framework/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var loggers = make(map[string]logr.Logger, 7)

// K8SHelper provides access to, and ways to interact with, the Kubernetes environment we're running on
type K8SHelper struct {
	Client client.Client
	Config *rest.Config
	Scheme *runtime.Scheme
}

// Helper provides easy access to the K8SHelper that has been set up when the operator called InitHelper
var Helper K8SHelper

// Fetch fetches the resource identified by the specified name and namespace into the given Object, returning the fetched Object
// or an error with a useful error message (only NotFound errors are passed through as-is) if something went wrong
func (rh K8SHelper) Fetch(name, namespace string, into runtime.Object) (runtime.Object, error) {
	if err := rh.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, into); err != nil {
		if errors.IsNotFound(err) {
			return into, err
		}
		return into, fmt.Errorf("couldn't fetch '%s' %s from namespace '%s': %s", name, util.GetObjectName(into), namespace, err.Error())
	}
	return into, nil
}

// LoggerFor retrieves a logger appropriate for the specified SerializableResource
func LoggerFor(resourceType SerializableResource) logr.Logger {
	name := controllerNameFor(resourceType)
	return loggers[name]
}

// Initializes the helper with the context provided by the specified Manager instance. This needs to be called early on by the
// operator when it is setup and before any Resource-related operations occur.
func InitHelper(mgr manager.Manager) {
	config := mgr.GetConfig()
	Helper = K8SHelper{
		Client: mgr.GetClient(),
		Config: config,
		Scheme: mgr.GetScheme(),
	}
	checkIfOpenShift(config)
}

func registerLogger(nameForLogger string) {
	logger, ok := loggers[nameForLogger]
	if !ok {
		logger = log.Log.WithName(nameForLogger)
		loggers[nameForLogger] = logger
	}
}
