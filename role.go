package framework

import (
	"halkyon.io/api/v1beta1"
	authorizv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Records the GroupVersionKind for Roles
var RoleGVK = authorizv1.SchemeGroupVersion.WithKind("Role")

// NeedsRole encapsulates the behavior that must be provided by Resources requiring a role
type NeedsRole interface {
	// GetRoleName specifies which name the role should have
	GetRoleName() string
	// Owner returns the Resource owning the Role this NeedsRole instance is associated with
	Owner() SerializableResource
}

// Role is a DependentResource representing a Kubernetes Role, designed to be reused in different contexts since it is a common
// use case that Resources need Roles to be properly used.
type Role struct {
	// Provides the base behavior
	*BaseDependentResource
	// NeedsRole owner to which some behavior is delegated
	Delegate NeedsRole
}

var _ DependentResource = &Role{}

// NewOwnedRole creates a new Role instance using the specified NeedsRole instance as owner and delegate
func NewOwnedRole(owner NeedsRole) Role {
	role := Role{BaseDependentResource: NewBaseDependentResource(owner.Owner(), RoleGVK), Delegate: owner}
	role.config.Watched = false
	return role
}

func (res Role) Fetch() (runtime.Object, error) {
	return DefaultFetcher(res)
}

func (res Role) GetCondition(_ runtime.Object, err error) *v1beta1.DependentCondition {
	return DefaultGetConditionFor(res, err)
}

func (res Role) Update(_ runtime.Object) (bool, runtime.Object, error) {
	return false, nil, nil
}

func (res Role) Name() string {
	return res.Delegate.GetRoleName()
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
