package framework

import (
	"halkyon.io/api/v1beta1"
	authorizv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var RoleBindingGVK = authorizv1.SchemeGroupVersion.WithKind("RoleBinding")

type NeedsRoleBinding interface {
	GetRoleBindingName() string
	GetAssociatedRoleName() string
	GetServiceAccountName() string
	Owner() v1beta1.HalkyonResource
}

type RoleBinding struct {
	*BaseDependentResource
	Delegate NeedsRoleBinding
}

func (res RoleBinding) NameFrom(underlying runtime.Object) string {
	return DefaultNameFrom(res, underlying)
}

func (res RoleBinding) Fetch() (runtime.Object, error) {
	return DefaultFetcher(res)
}

func (res RoleBinding) GetCondition(_ runtime.Object, err error) *v1beta1.DependentCondition {
	return DefaultGetConditionFor(res, err)
}

var _ DependentResource = &RoleBinding{}

func (res RoleBinding) Update(toUpdate runtime.Object) (bool, error) {
	// add appropriate subject for owner
	rb := toUpdate.(*authorizv1.RoleBinding)
	owner := res.Owner()

	// check if the binding contains the current owner as subject
	namespace := owner.GetNamespace()
	name := res.Delegate.GetServiceAccountName()
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

func NewOwnedRoleBinding(owner NeedsRoleBinding) RoleBinding {
	binding := RoleBinding{
		BaseDependentResource: NewBaseDependentResource(owner.Owner(), RoleBindingGVK),
		Delegate:              owner,
	}
	binding.config.Watched = false
	return binding
}

func (res RoleBinding) Name() string {
	return res.Delegate.GetRoleBindingName()
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
			Name: res.Delegate.GetAssociatedRoleName(),
		}
		ser.Subjects = []authorizv1.Subject{
			{Kind: "ServiceAccount", Name: res.Delegate.GetServiceAccountName(), Namespace: namespace},
		}
	}
	return ser, nil
}
