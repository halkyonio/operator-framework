package framework

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Predicate interface {
	Matches(resource DependentResource) bool
}

type TypePredicate struct {
	gvk  schema.GroupVersionKind
	desc string
}

func (tp TypePredicate) Matches(resource DependentResource) bool {
	return resource.GetConfig().GroupVersionKind == tp.gvk
}

func (tp TypePredicate) String() string {
	return tp.desc
}

func TypePredicateFor(gvk schema.GroupVersionKind) Predicate {
	return TypePredicate{
		gvk:  gvk,
		desc: fmt.Sprintf("GetConfig().GroupVersionKind == %v", gvk),
	}
}
