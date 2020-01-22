package capability

import (
	"github.com/hashicorp/go-plugin"
	"net/rpc"
)

var _ plugin.Plugin = &GoPluginPlugin{}

type GoPluginPlugin struct {
	name     string
	Delegate PluginResource
}

func (p *GoPluginPlugin) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &PluginServerImpl{capability: p.Delegate}, nil
}

func (p *GoPluginPlugin) Client(b *plugin.MuxBroker, client *rpc.Client) (interface{}, error) {
	return &PluginClient{name: p.name, client: client}, nil
}

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "HALKYON_CAPABILITY_PLUGIN",
	MagicCookieValue: "io.halkyon.capability.plugin",
}
