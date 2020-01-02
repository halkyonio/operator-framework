package framework

import (
	"halkyon.io/api/v1beta1"
	authorizv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type RoleBinding struct {
	*BaseDependentResource
	namer               func() string
	associatedRoleNamer func() string
	serviceAccountNamer func() string
}

func (res RoleBinding) NameFrom(underlying runtime.Object) string {
	return DefaultNameFrom(res, underlying)
}

func (res RoleBinding) Fetch(helper *K8SHelper) (runtime.Object, error) {
	return DefaultFetcher(res, helper)
}

func (res RoleBinding) IsReady(underlying runtime.Object) (ready bool, message string) {
	return DefaultIsReady(underlying)
}

var _ DependentResource = &RoleBinding{}

func (res RoleBinding) Update(toUpdate runtime.Object) (bool, error) {
	// add appropriate subject for owner
	rb := toUpdate.(*authorizv1.RoleBinding)
	owner := res.Owner()

	// check if the binding contains the current owner as subject
	namespace := owner.GetNamespace()
	name := res.serviceAccountNamer()
	found := false
	for _, subject := range rb.Subjects {
		if subject.Name == name && subject.Namespace == namespace {
			found = true
			break
		}
	}

	if !found {
		rb.Subjects = append(rb.Subjects, authorizv1.Subject{
			Kind:      "ServiceAccount",
			Namespace: namespace,
			Name:      name,
		})
	}

	return !found, nil
}

func (res RoleBinding) NewInstanceWith(owner v1beta1.HalkyonResource) DependentResource {
	return NewOwnedRoleBinding(owner, res.namer, res.associatedRoleNamer, res.serviceAccountNamer)
}

func NewOwnedRoleBinding(owner v1beta1.HalkyonResource, namer, associatedRoleNamer, serviceAccountNamer func() string) RoleBinding {
	binding := RoleBinding{
		BaseDependentResource: NewBaseDependentResource(&authorizv1.RoleBinding{}, owner),
		namer:                 namer,
		associatedRoleNamer:   associatedRoleNamer,
		serviceAccountNamer:   serviceAccountNamer,
	}
	binding.config.Watched = false
	return binding
}

func (res RoleBinding) Name() string {
	return res.namer()
}

func (res RoleBinding) Build(empty bool) (runtime.Object, error) {
	ser := &authorizv1.RoleBinding{}
	if !empty {
		c := res.Owner()
		namespace := c.GetNamespace()
		ser.ObjectMeta = metav1.ObjectMeta{
			Name:      res.Name(),
			Namespace: namespace,
		}
		ser.RoleRef = authorizv1.RoleRef{
			Kind: "Role",
			Name: res.associatedRoleNamer(),
		}
		ser.Subjects = []authorizv1.Subject{
			{Kind: "ServiceAccount", Name: res.serviceAccountNamer(), Namespace: namespace},
		}
	}
	return ser, nil
}
