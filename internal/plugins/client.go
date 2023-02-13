package plugins

import (
	"context"
	"fmt"
	"os/exec"
	"text/template"
	"time"

	hcplugin "github.com/hashicorp/go-plugin"
)

type StartPluginsOptions struct {
	Timeout time.Duration
}

// StartPlugins starts plugins from the given commands and returns a single template.FuncMap that contains all the
// functions from all the plugins. The plugins will be killed when ctx is canceled.
func StartPlugins(ctx context.Context, commands []string, options *StartPluginsOptions) (template.FuncMap, error) {
	if len(commands) == 0 {
		return nil, nil
	}
	timeout := time.Duration(0)
	if options != nil {
		timeout = options.Timeout
	}
	killFuncs := make([]func(), len(commands))
	killEmAll := func() {
		for i := range killFuncs {
			killFuncs[i]()
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
		funcs, err := provider.ListFunctions(ctx)
		if err != nil {
			killEmAll()
			return nil, err
		}
		for _, funcName := range funcs {
			funcMap[funcName] = pluginFunc(ctx, provider, funcName, timeout)
		}
	}
	go func() {
		<-ctx.Done()
		killEmAll()
	}()
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

func grpcFuncProviderFromCommand(command string) (_ Plugin, kill func(), _ error) {
	pluginClient := hcplugin.NewClient(&hcplugin.ClientConfig{
		// nolint:gosec // pluginCmd is a user-provided string
		Cmd:             exec.Command("sh", "-c", command),
		HandshakeConfig: PluginHandshake,
		Plugins: map[string]hcplugin.Plugin{
			"gotmpl": &PluginServer{},
		},
		AllowedProtocols: []hcplugin.Protocol{
			hcplugin.ProtocolGRPC,
		},
	})
	kill = pluginClient.Kill
	rpcClient, err := pluginClient.Client()
	if err != nil {
		kill()
		return nil, kill, err
	}
	raw, err := rpcClient.Dispense("gotmpl")
	if err != nil {
		kill()
		return nil, kill, err
	}
	provider, ok := raw.(Plugin)
	if !ok {
		kill()
		return nil, kill, fmt.Errorf("unexpected type %T", raw)
	}
	return provider, kill, nil
}
