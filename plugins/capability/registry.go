package capability

import (
	"fmt"
	"halkyon.io/api/capability-info/clientset/versioned"
	"halkyon.io/api/capability-info/v1beta1"
	halkyon "halkyon.io/api/capability/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"strings"
)

type typeRegistry map[halkyon.CapabilityType]Plugin
type pluginsRegistry map[halkyon.CapabilityCategory]typeRegistry

var plugins pluginsRegistry
var capInfoClient = versioned.NewForConfigOrDie(controllerruntime.GetConfigOrDie()).HalkyonV1beta1().CapabilityInfos()

func GetPluginFor(category halkyon.CapabilityCategory, capabilityType halkyon.CapabilityType) (Plugin, error) {
	if types, ok := plugins[categoryKey(category)]; ok {
		if p, ok := types[typeKey(capabilityType)]; ok {
			return p, nil
		}
	}
	return nil, fmt.Errorf("couldn't find a plugin to handle capability with category '%s' and type '%s'", category, capabilityType)
}

func categoryKey(category halkyon.CapabilityCategory) halkyon.CapabilityCategory {
	return halkyon.CapabilityCategory(strings.ToLower(category.String()))
}

func typeKey(capType halkyon.CapabilityType) halkyon.CapabilityType {
	return halkyon.CapabilityType(strings.ToLower(capType.String()))
}

func register(p *PluginClient) {
	category := p.GetCategory()
	categoryKey := categoryKey(category)
	if len(plugins) == 0 {
		plugins = make(pluginsRegistry, 7)
	}
	types, ok := plugins[categoryKey]
	if !ok {
		types = make(typeRegistry, 7)
		plugins[categoryKey] = types
	}
	typeInfos := p.GetTypes()
	for _, typeInfo := range typeInfos {
		t := typeInfo.Type
		typeKey := typeKey(t)
		plug, ok := types[typeKey]
		if ok {
			p.log.Error(fmt.Errorf("a plugin named '%s' is already registered for '%s'/'%s' category/type pair", plug.Name(), category, t),
				fmt.Sprintf("'%s' plugin will not be registered to provide capability '%s'/'%s'", p.Name(), category, t))
			continue
		}

		// create or update associated CapabilityInfo
		capabilityName := fmt.Sprintf("%v-%v", categoryKey, typeKey)
		capInfo := &v1beta1.CapabilityInfo{
			ObjectMeta: v1.ObjectMeta{Name: capabilityName},
			Spec: v1beta1.CapabilityInfoSpec{
				Versions: v1beta1.VersionsAsString(typeInfo.Versions...),
				Category: category.String(),
				Type:     t.String(),
			},
		}

		// check if the capability info already exist
		ci, err := capInfoClient.Get(capabilityName, v1.GetOptions{})
		if err == nil {
			// if it exists, update it with potentially new information
			capInfo.ResourceVersion = ci.ResourceVersion
			_, err = capInfoClient.Update(capInfo)
		} else {
			// if not create it
			if errors.IsNotFound(err) {
				_, err = capInfoClient.Create(capInfo)
			}
		}

		// if an error occurred at any time, log it and ignore the
		if err != nil {
			p.log.Error(err, fmt.Sprintf("couldn't create or update capabilityinfo named '%s', associated capability will be ignored", capabilityName))
			continue
		}

		// if everything went well, register plugin
		types[typeKey] = p
		p.log.Info(fmt.Sprintf("Registered plugin named '%s' for category '%s' / type '%s' pair", p.name, category, t))
	}
}
