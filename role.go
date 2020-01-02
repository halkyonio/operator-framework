package framework

import (
	"halkyon.io/api/v1beta1"
	authorizv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Role struct {
	*BaseDependentResource
	namer func() string
}

func (res Role) NameFrom(underlying runtime.Object) string {
	return DefaultNameFrom(res, underlying)
}

func (res Role) Fetch(helper *K8SHelper) (runtime.Object, error) {
	return DefaultFetcher(res, helper)
}

func (res Role) IsReady(underlying runtime.Object) (ready bool, message string) {
	return DefaultIsReady(underlying)
}

var _ DependentResource = &Role{}

func (res Role) Update(toUpdate runtime.Object) (bool, error) {
	return false, nil
}

func NewOwnedRole(owner v1beta1.HalkyonResource, namerFn func() string) Role {
	role := Role{BaseDependentResource: NewBaseDependentResource(&authorizv1.Role{}, owner), namer: namerFn}
	role.config.Watched = false
	return role
}

func (res Role) Name() string {
	return res.namer()
}

func (res Role) Build(empty bool) (runtime.Object, error) {
	ser := &authorizv1.Role{}
	if !empty {
		c := res.Owner()
		ser.ObjectMeta = metav1.ObjectMeta{
			Name:      res.Name(),
			Namespace: c.GetNamespace(),
		}
		ser.Rules = []authorizv1.PolicyRule{
			{
				APIGroups:     []string{"security.openshift.io"},
				Resources:     []string{"securitycontextconstraints"},
				ResourceNames: []string{"privileged"},
				Verbs:         []string{"use"},
			},
		}
	}
	return ser, nil
}
