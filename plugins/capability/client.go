package capability

import (
	"encoding/gob"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	halkyon "halkyon.io/api/capability/v1beta1"
	framework "halkyon.io/operator-framework"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
)

// Plugin is the operator-facing interface that can be interacted with in Halkyon
type Plugin interface {
	// Name returns the name of this Plugin
	Name() string
	// GetCategory retrieves the CapabilityCategory supported by this Plugin
	GetCategory() halkyon.CapabilityCategory
	// GetTypes returns TypeInfo providing information about CapabilityTypes this Plugin supports
	GetTypes() []TypeInfo
	// ReadyFor initializes the DependentResources needed by the given Capability and readies the Plugin for requests by the host.
	// Note that the order in which the DependentResources are returned is significant and the operator will process them in the
	// specified order. This is needed because some capabilities might require some dependent resources to be present before
	// processing others.
	ReadyFor(owner *halkyon.Capability) []framework.DependentResource
	// Kill kills the RPC client and server associated with this Plugin when the host process terminates
	Kill()
}

type TypeInfo struct {
	Type     halkyon.CapabilityType
	Versions []string
}

type PluginClient struct {
	client      *rpc.Client
	name        string
	owner       *halkyon.Capability
	gpClient    *plugin.Client
	capCategory *halkyon.CapabilityCategory
	capTypes    *[]TypeInfo
	log         logr.Logger
}

var _ Plugin = &PluginClient{}
var _ killableClient = &PluginClient{}

var emptyGVK = schema.GroupVersionKind{}

type killableClient interface {
	Plugin
	recordGoPluginClient(client *plugin.Client)
}

func (p *PluginClient) Name() string {
	return p.name
}

func (p *PluginClient) recordGoPluginClient(client *plugin.Client) {
	p.gpClient = client
}

func (p *PluginClient) GetCategory() halkyon.CapabilityCategory {
	if p.capCategory == nil {
		var cat halkyon.CapabilityCategory
		p.call("GetCategory", emptyGVK, &cat)
		p.capCategory = &cat
	}
	return *p.capCategory
}

func (p *PluginClient) GetTypes() []TypeInfo {
	if p.capTypes == nil {
		res := []TypeInfo{}
		p.call("GetTypes", emptyGVK, &res)
		p.capTypes = &res
	}
	return *p.capTypes
}

func (p *PluginClient) Kill() {
	p.gpClient.Kill()
}

func (p *PluginClient) ReadyFor(owner *halkyon.Capability) []framework.DependentResource {
	client := &PluginClient{
		client: p.client,
		name:   p.name,
		log:    p.log,
		owner:  owner,
	}
	resourcesTypes := []schema.GroupVersionKind{}
	client.call("GetDependentResourceTypes", emptyGVK, &resourcesTypes)
	depRes := make([]framework.DependentResource, 0, len(resourcesTypes))
	for _, rt := range resourcesTypes {
		depRes = append(depRes, &PluginDependentResource{client: client, gvk: rt, owner: owner})
	}
	return depRes
}

// NewPlugin creates the infrastructure required for the host (the operator) to be able to call the plugin binary which path is
// given, setting up a logger that can be used to output information in the operator logs. The new Plugin is queried and its
// supported category/type pairs are registered so that when a Capability requiring one of these pairs is created, the operator
// can delegate to the appropriate plugin. The RPC server and client are also started using the Handshake configuration.
func NewPlugin(path string, log logr.Logger) (Plugin, error) {
	name := filepath.Base(path)

	// We're a host. Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins:         map[string]plugin.Plugin{name: &GoPluginPlugin{name: name}},
		Cmd:             exec.Command(path),
		Logger: hclog.New(&hclog.LoggerOptions{
			Output: hclog.DefaultOutput,
			Level:  hclog.Trace,
			Name:   name,
		}),
	})

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense(name)
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}
	p := raw.(*PluginClient)
	p.log = log
	p.recordGoPluginClient(client)

	register(p)

	return p, nil
}

func (p *PluginClient) call(method string, targetDependentType schema.GroupVersionKind, result interface{}, underlying ...runtime.Object) {
	request := p.createRequest(method, targetDependentType, underlying...)
	p.callWithRequest(method, request, result)
}

func (p *PluginClient) callWithRequest(method string, request PluginRequest, result interface{}) {
	err := p.client.Call("Plugin."+method, request, result)
	if err != nil {
		p.log.Error(err, fmt.Sprintf("error calling %s on %s plugin", method, p.name))
	}
}

func (p *PluginClient) createRequest(method string, targetDependentType schema.GroupVersionKind, underlying ...runtime.Object) PluginRequest {
	if len(underlying) > 1 {
		p.log.Error(fmt.Errorf("error calling %s on %s plugin", method, p.name), fmt.Sprintf("call only accepts one extra argument, was given %v", underlying))
	}
	request := PluginRequest{}
	if p.owner != nil {
		request.Owner = p.owner
	}
	if !targetDependentType.Empty() {
		request.Target = targetDependentType
	}
	if len(underlying) == 1 {
		request.setArg(underlying[0])
	}
	return request
}

func init() {
	gob.Register(&halkyon.Capability{})
}
