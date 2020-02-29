package capability

import (
	"fmt"
	"github.com/hashicorp/go-hclog"
	halkyon "halkyon.io/api/capability/v1beta1"
	framework "halkyon.io/operator-framework"
	"reflect"
)

// PluginResource gathers behavior that plugin implementors are expected to provide to the plugins architecture
type PluginResource interface {
	// GetSupportedCategory returns the CapabilityCategory that this plugin supports
	GetSupportedCategory() halkyon.CapabilityCategory
	// GetSupportedTypes returns the list of supported CapabilityTypes along with associated versions when they exist.
	// Note that, while a plugin can only support one CapabilityCategory (e.g. "database"), a plugin can provide support for
	// multiple CapabilityTypes (e.g. "postgresql", "mysql", etc.) within the confine of the specified category.
	GetSupportedTypes() []TypeInfo
	// GetDependentResourcesWith returns an ordered list of DependentResources initialized with the specified owner.
	// DependentResources represent secondary resources that the capability might need to work (e.g. Kubernetes Role or Secret)
	// along with the resource (if it exists) implementing the capability itself (e.g. KubeDB's Postgres).
	GetDependentResourcesWith(owner framework.SerializableResource) []framework.DependentResource
	// CheckValidity checks that the specified owner is valid according to the Plugin's requirements and returns the list of
	// validation error messages
	CheckValidity(owner framework.SerializableResource) []string
}

type SimplePluginResourceStem struct {
	ct     []TypeInfo
	cc     halkyon.CapabilityCategory
	name   string
	Logger hclog.Logger
}

func NewSimplePluginResourceStem(cat halkyon.CapabilityCategory, typ TypeInfo) SimplePluginResourceStem {
	return SimplePluginResourceStem{cc: cat, ct: []TypeInfo{typ}, name: GetPluginExecutableName()}
}
func (p SimplePluginResourceStem) GetSupportedCategory() halkyon.CapabilityCategory {
	return p.cc
}

func (p SimplePluginResourceStem) GetSupportedTypes() []TypeInfo {
	return p.ct
}

func (p *SimplePluginResourceStem) SetLogger(logger hclog.Logger) {
	p.Logger = logger
}

func (p SimplePluginResourceStem) GetPrefixedValidationMessage(msg string) string {
	return fmt.Sprintf("%s: %s", p.name, msg)
}

type NeedsLogging interface {
	SetLogger(logger hclog.Logger)
}

type QueryingSimplePluginResourceStem struct {
	SimplePluginResourceStem
	resolver func(logger hclog.Logger) TypeInfo
}

func NewQueryingSimplePluginResourceStem(cat halkyon.CapabilityCategory, typeInfoResolver func(logger hclog.Logger) TypeInfo) QueryingSimplePluginResourceStem {
	return QueryingSimplePluginResourceStem{
		SimplePluginResourceStem: SimplePluginResourceStem{cc: cat},
		resolver:                 typeInfoResolver,
	}
}

func (p *QueryingSimplePluginResourceStem) GetSupportedTypes() []TypeInfo {
	if len(p.ct) == 0 {
		p.ct = []TypeInfo{p.resolver(p.Logger)}
	}
	return p.ct
}

var _ PluginResource = &AggregatePluginResource{}

type AggregatePluginResource struct {
	category        halkyon.CapabilityCategory
	pluginResources map[halkyon.CapabilityType]PluginResource
}

func NewAggregatePluginResource(logger hclog.Logger, resources ...PluginResource) (PluginResource, error) {
	apr := AggregatePluginResource{
		pluginResources: make(map[halkyon.CapabilityType]PluginResource, len(resources)),
	}
	for _, resource := range resources {
		if needsLogging, ok := resource.(NeedsLogging); ok {
			name := reflect.TypeOf(needsLogging).Elem().Name()
			needsLogging.SetLogger(logger.Named(name))
		}
		category := categoryKey(resource.GetSupportedCategory())
		if len(apr.category) == 0 {
			apr.category = category
		}
		if !apr.category.Equals(category) {
			return nil, fmt.Errorf("can only aggregate PluginResources providing the same category, got %v and %v", apr.category, category)
		}
		for _, typeInfo := range resource.GetSupportedTypes() {
			apr.pluginResources[typeKey(typeInfo.Type)] = resource
		}
	}
	return apr, nil
}

func (a AggregatePluginResource) GetSupportedCategory() halkyon.CapabilityCategory {
	return a.category
}

func (a AggregatePluginResource) GetSupportedTypes() []TypeInfo {
	types := make([]TypeInfo, 0, len(a.pluginResources))
	for _, resource := range a.pluginResources {
		types = append(types, resource.GetSupportedTypes()...)
	}
	return types
}

func (a AggregatePluginResource) GetDependentResourcesWith(owner framework.SerializableResource) []framework.DependentResource {
	capType := typeKey(owner.(*halkyon.Capability).Spec.Type)
	return a.pluginResources[capType].GetDependentResourcesWith(owner)
}

func (a AggregatePluginResource) CheckValidity(owner framework.SerializableResource) []string {
	errors := make([]string, 0, len(a.pluginResources))
	for _, resource := range a.pluginResources {
		if msgs := resource.CheckValidity(owner); msgs != nil {
			errors = append(errors, msgs...)
		}
	}
	return errors
}
