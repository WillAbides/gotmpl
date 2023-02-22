package plugins

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"text/template"
	"time"

	hcplugin "github.com/hashicorp/go-plugin"
)

type InitPluginsOptions struct {
	// ExecTimeout is the timeout calling ExecuteFunction on a plugin. Defaults to no timeout.
	ExecTimeout time.Duration
	// FunctionsTimeout is the timeout calling ListFunctions on a plugin. Defaults to no timeout.
	FunctionsTimeout time.Duration
}

// InitPlugins initializes both command and grpc plugins and returns a single template.FuncMap that contains functions
// from all plugins with grpc plugins taking precedence over command plugins with the same name. When multiple plugins
// have the same name within the same type, the last one will be used.
// Command plugins will be killed when ctx is canceled.
func InitPlugins(ctx context.Context, commands, grpcAddresses []string, options *InitPluginsOptions) (template.FuncMap, error) {
	if options == nil {
		options = &InitPluginsOptions{}
	}
	funcs, err := initCommandPlugins(ctx, commands, *options)
	if err != nil {
		return nil, err
	}
	if funcs == nil {
		funcs = template.FuncMap{}
	}
	grpcFuncs, err := initGrpcPlugins(ctx, grpcAddresses, *options)
	if err != nil {
		return nil, err
	}
	for k, v := range grpcFuncs {
		funcs[k] = v
	}
	return funcs, nil
}

func initCommandPlugins(ctx context.Context, commands []string, options InitPluginsOptions) (template.FuncMap, error) {
	if len(commands) == 0 {
		return nil, nil
	}
	killFuncs := make([]func(), len(commands))
	killEmAll := func() {
		for i := range killFuncs {
			if killFuncs[i] != nil {
				killFuncs[i]()
			}
		}
	}
	funcMap := template.FuncMap{}
	for i := range commands {
		var err error
		var provider Plugin
		provider, killFuncs[i], err = grpcFuncProviderFromCommand(commands[i])
		if err != nil {
			killEmAll()
			return nil, err
		}
		funcs, err := listFunctionsWithTimeout(ctx, provider, options.FunctionsTimeout)
		if err != nil {
			killEmAll()
			return nil, err
		}
		for _, funcName := range funcs {
			funcMap[funcName] = pluginFunc(ctx, provider, funcName, options.ExecTimeout)
		}
	}
	go func() {
		<-ctx.Done()
		killEmAll()
	}()
	return funcMap, nil
}

func initGrpcPlugins(ctx context.Context, addresses []string, options InitPluginsOptions) (template.FuncMap, error) {
	if len(addresses) == 0 {
		return nil, nil
	}
	funcMap := template.FuncMap{}
	for i := range addresses {
		provider, err := grpcFuncProviderFromAddress(addresses[i])
		if err != nil {
			return nil, err
		}
		funcs, err := listFunctionsWithTimeout(ctx, provider, options.FunctionsTimeout)
		if err != nil {
			return nil, err
		}
		for _, funcName := range funcs {
			funcMap[funcName] = pluginFunc(ctx, provider, funcName, options.ExecTimeout)
		}
	}
	return funcMap, nil
}

func pluginFunc(ctx context.Context, plugin Plugin, name string, timeout time.Duration) func(args ...any) (any, error) {
	return func(args ...any) (any, error) {
		if timeout > 0 {
			var cancel func()
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		return plugin.ExecuteFunction(ctx, name, args)
	}
}

func grpcFuncProviderFromCommand(command string) (Plugin, func(), error) {
	cmd := exec.Command("sh", "-c", command)
	return initPlugin(cmd, nil)
}

// grpcFuncProviderFromAddress is like grpcFuncProviderFromCommand but connects to a plugin server at the given address.
func grpcFuncProviderFromAddress(addr string) (Plugin, error) {
	address, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	pluginClient := hcplugin.NewClient(&hcplugin.ClientConfig{
		HandshakeConfig: PluginHandshake,
		Plugins: map[string]hcplugin.Plugin{
			"gotmpl": &PluginServer{},
		},
		AllowedProtocols: []hcplugin.Protocol{
			hcplugin.ProtocolGRPC,
		},
		Reattach: &hcplugin.ReattachConfig{
			Protocol: hcplugin.ProtocolGRPC,
			Addr:     address,
		},
	})
	rpcClient, err := pluginClient.Client()
	if err != nil {
		return nil, err
	}
	raw, err := rpcClient.Dispense("gotmpl")
	if err != nil {
		return nil, err
	}
	provider, ok := raw.(Plugin)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T", raw)
	}
	return provider, nil
}

func initPlugin(cmd *exec.Cmd, reattach *hcplugin.ReattachConfig) (Plugin, func(), error) {
	pluginClient := hcplugin.NewClient(&hcplugin.ClientConfig{
		Cmd:             cmd,
		Reattach:        reattach,
		HandshakeConfig: PluginHandshake,
		Plugins: map[string]hcplugin.Plugin{
			"gotmpl": &PluginServer{},
		},
		AllowedProtocols: []hcplugin.Protocol{
			hcplugin.ProtocolGRPC,
		},
	})
	rpcClient, err := pluginClient.Client()
	if err != nil {
		pluginClient.Kill()
		return nil, nil, err
	}
	raw, err := rpcClient.Dispense("gotmpl")
	if err != nil {
		pluginClient.Kill()
		return nil, nil, err
	}
	provider, ok := raw.(Plugin)
	if !ok {
		pluginClient.Kill()
		return nil, nil, fmt.Errorf("unexpected type %T", raw)
	}
	return provider, pluginClient.Kill, nil
}

func listFunctionsWithTimeout(ctx context.Context, plugin Plugin, timeout time.Duration) ([]string, error) {
	if timeout <= 0 {
		return plugin.ListFunctions(ctx)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return plugin.ListFunctions(ctx)
}
