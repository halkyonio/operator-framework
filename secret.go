package framework

import (
	"halkyon.io/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"strings"
)

var secretGVK = v1.SchemeGroupVersion.WithKind("Secret")
var _ DependentResource = &Secret{}

type NeedsSecret interface {
	GetDataMap() map[string][]byte
	GetSecretName() string
	Owner() v1beta1.HalkyonResource
}

type Secret struct {
	*BaseDependentResource
	Delegate NeedsSecret
}

func (res Secret) NameFrom(underlying runtime.Object) string {
	return DefaultNameFrom(res, underlying)
}

func (res Secret) Fetch() (runtime.Object, error) {
	return DefaultFetcher(res)
}

func (res Secret) GetCondition(underlying runtime.Object, err error) *v1beta1.DependentCondition {
	return DefaultGetConditionFor(res, err)
}

func (res Secret) Update(toUpdate runtime.Object) (bool, runtime.Object, error) {
	secret := toUpdate.(*v1.Secret)
	dataMap := res.Delegate.GetDataMap()
	if !reflect.DeepEqual(dataMap, secret.Data) {
		secret.Data = dataMap
		return true, secret, nil
	}

	return false, secret, nil
}

func NewSecret(owner NeedsSecret, config ...DependentResourceConfig) Secret {
	var c DependentResourceConfig
	if len(config) != 1 {
		c = NewDefaultSecretConfig()
	} else {
		c = config[0]
	}

	return Secret{BaseDependentResource: NewConfiguredBaseDependentResource(owner.Owner(), c), Delegate: owner}
}

func NewDefaultSecretConfig() DependentResourceConfig {
	config := NewConfig(secretGVK)
	config.Watched = true
	config.Updated = true
	return config
}

//buildSecret returns the secret resource
func (res Secret) Build(empty bool) (runtime.Object, error) {
	secret := &v1.Secret{}
	if !empty {
		secret.ObjectMeta = metav1.ObjectMeta{
			Name:      res.Name(),
			Namespace: res.owner.GetNamespace(),
		}
		secret.Data = res.Delegate.GetDataMap()
	}

	return secret, nil
}

func (res Secret) Name() string {
	return res.Delegate.GetSecretName()
}

func DefaultSecretNameFor(secretOwner NeedsSecret) string {
	return strings.ToLower(secretOwner.Owner().GetName()) + "-config"
}
