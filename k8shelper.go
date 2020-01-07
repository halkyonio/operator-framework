package framework

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"halkyon.io/api/v1beta1"
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

type K8SHelper struct {
	Client client.Client
	Config *rest.Config
	Scheme *runtime.Scheme
}

var Helper K8SHelper

func (rh K8SHelper) Fetch(name, namespace string, into runtime.Object) (runtime.Object, error) {
	if err := rh.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, into); err != nil {
		if errors.IsNotFound(err) {
			return into, err
		}
		return into, fmt.Errorf("couldn't fetch '%s' %s from namespace '%s': %s", name, util.GetObjectName(into), namespace, err.Error())
	}
	return into, nil
}

func LoggerFor(resourceType v1beta1.HalkyonResource) logr.Logger {
	name := controllerNameFor(resourceType)
	return loggers[name]
}

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
