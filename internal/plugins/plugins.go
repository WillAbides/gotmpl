package plugins

import (
	"context"

	hcplugin "github.com/hashicorp/go-plugin"
	pluginv1 "github.com/willabides/gotmpl/internal/gen/proto/go/plugin/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

// Plugin is the interface that plugins must implement.
type Plugin interface {
	ListFunctions(ctx context.Context) ([]string, error)
	ExecuteFunction(ctx context.Context, name string, args []any) (any, error)
}

// PluginHandshake is the handshake config for plugins.
var PluginHandshake = hcplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "GOTMPL_PLUGIN",
	MagicCookieValue: "GOTMPL_PLUGIN",
}

type PluginServer struct {
	hcplugin.Plugin
	GotmplPlugin Plugin
}

func (p *PluginServer) GRPCServer(_ *hcplugin.GRPCBroker, s *grpc.Server) error {
	pluginv1.RegisterPluginServiceServer(s, &grpcServer{
		GotmplPlugin: p.GotmplPlugin,
	})
	return nil
}

func (p *PluginServer) GRPCClient(_ context.Context, _ *hcplugin.GRPCBroker, c *grpc.ClientConn) (any, error) {
	return &grpcClient{
		client: pluginv1.NewPluginServiceClient(c),
	}, nil
}

type grpcServer struct {
	GotmplPlugin Plugin
	pluginv1.UnimplementedPluginServiceServer
}

func NewGrpcServer(p Plugin) pluginv1.PluginServiceServer {
	return &grpcServer{
		GotmplPlugin: p,
	}
}

func (p *grpcServer) ListFunctions(ctx context.Context, req *pluginv1.ListFunctionsRequest) (*pluginv1.ListFunctionsResponse, error) {
	funcs, err := p.GotmplPlugin.ListFunctions(ctx)
	if err != nil {
		return nil, err
	}
	return &pluginv1.ListFunctionsResponse{
		Functions: funcs,
	}, nil
}

func (p *grpcServer) ExecuteFunction(ctx context.Context, req *pluginv1.ExecuteFunctionRequest) (*pluginv1.ExecuteFunctionResponse, error) {
	args := make([]any, len(req.Args))
	for i := range req.Args {
		args[i] = req.Args[i].AsInterface()
	}
	result, err := p.GotmplPlugin.ExecuteFunction(ctx, req.Function, args)
	if err != nil {
		return nil, err
	}
	var response pluginv1.ExecuteFunctionResponse
	response.Result, err = structpb.NewValue(result)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

type grpcClient struct {
	client pluginv1.PluginServiceClient
}

func (p grpcClient) ListFunctions(ctx context.Context) ([]string, error) {
	resp, err := p.client.ListFunctions(ctx, &pluginv1.ListFunctionsRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Functions, nil
}

func (p grpcClient) ExecuteFunction(ctx context.Context, name string, args []any) (any, error) {
	req := &pluginv1.ExecuteFunctionRequest{
		Function: name,
		Args:     make([]*structpb.Value, len(args)),
	}
	var err error
	for i := range args {
		req.Args[i], err = structpb.NewValue(args[i])
		if err != nil {
			return nil, err
		}
	}
	resp, err := p.client.ExecuteFunction(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Result.AsInterface(), nil
}
