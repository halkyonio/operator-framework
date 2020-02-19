package util

import (
	"fmt"
	"halkyon.io/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func GetImageReference(imageName string, version ...string) string {
	runtimeVersion := "latest"
	if len(version) == 1 && len(version[0]) > 0 {
		runtimeVersion = version[0]
	}
	return fmt.Sprintf("%s:%s", imageName, runtimeVersion)
}

func Index(vs []string, t string) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}
	return -1
}

func NewTrue() *bool {
	b := true
	return &b
}

func NewFalse() *bool {
	b := false
	return &b
}

func GetObjectName(object runtime.Object) string {
	t := reflect.TypeOf(object)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

func GetGVKFor(object runtime.Object, scheme *runtime.Scheme) schema.GroupVersionKind {
	gvk, err := apiutil.GVKForObject(object, scheme)
	if err != nil {
		panic(err)
	}
	return gvk
}

// Convert Array of parameters to a Map
func ParametersAsMap(parameters []v1beta1.NameValuePair) map[string]string {
	result := make(map[string]string)
	for _, parameter := range parameters {
		result[parameter.Name] = parameter.Value
	}
	return result
}
