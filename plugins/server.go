package plugins

import (
	hcplugin "github.com/hashicorp/go-plugin"
	"github.com/willabides/gotmpl/internal/plugins"
)

// Plugin is the interface that plugins must implement.
type Plugin interface{ plugins.Plugin }

// PluginHandshake is the handshake config for plugins.
var PluginHandshake = plugins.PluginHandshake

// ServePlugin serves a plugin. This is meant to be called from a plugin's main function.
func ServePlugin(p Plugin) {
	hcplugin.Serve(&hcplugin.ServeConfig{
		HandshakeConfig: PluginHandshake,
		GRPCServer:      hcplugin.DefaultGRPCServer,
		Plugins: map[string]hcplugin.Plugin{
			"gotmpl": &plugins.PluginServer{
				GotmplPlugin: p,
			},
		},
	})
}
