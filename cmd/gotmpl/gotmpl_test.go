package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gotmplv1 "github.com/willabides/gotmpl/internal/gen/proto/go/gotmpl/v1"
	pluginv1 "github.com/willabides/gotmpl/internal/gen/proto/go/plugin/v1"
	internalPlugins "github.com/willabides/gotmpl/internal/plugins"
	publicPlugins "github.com/willabides/gotmpl/plugins"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test_serverCmd(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		ctx := context.Background()
		pluginSrv := startPluginServer(t, publicPlugins.NewFuncMapPlugin(map[string]any{"upcase": strings.ToUpper}))
		grpcServerPort := startTestServer(ctx, t, []string{fmt.Sprintf("localhost:%d", pluginSrv)})
		clientConn, err := grpc.Dial(fmt.Sprintf("localhost:%d", grpcServerPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		data, err := structpb.NewValue(map[string]any{"name": "world"})
		require.NoError(t, err)
		client := gotmplv1.NewGotmplServiceClient(clientConn)
		got, err := client.Execute(ctx, &gotmplv1.ExecuteRequest{
			Template: "hello {{ upcase .name }}",
			Data:     data,
		})
		require.NoError(t, err)
		assert.Equal(t, "hello WORLD", got.Result)
	})
}

func Test_execCmd(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		ctx := context.Background()
		pluginSrv := startPluginServer(t, publicPlugins.NewFuncMapPlugin(map[string]any{"upcase": strings.ToUpper}))
		data, err := json.Marshal(map[string]any{"name": "world"})
		require.NoError(t, err)
		var stdout bytes.Buffer
		c := &execCmd{
			Template: "hello {{ upcase .name }}",
			Data:     string(data),
			pluginData: pluginData{
				GrpcPlugin: []string{fmt.Sprintf("localhost:%d", pluginSrv)},
			},
		}
		err = c.Run(ctx, &kong.Context{
			Kong: &kong.Kong{
				Stdout: &stdout,
			},
		})
		require.NoError(t, err)
		require.Equal(t, "hello WORLD", stdout.String())
	})
}

func startTestServer(ctx context.Context, t *testing.T, grpcPlugins []string) int {
	ctx, cancel := context.WithCancel(ctx)
	stdoutReader, stdoutWriter := io.Pipe()
	stdoutScanner := bufio.NewScanner(stdoutReader)
	c := &serverCmd{
		pluginData: pluginData{
			GrpcPlugin: grpcPlugins,
		},
		Address: "localhost:0",
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NoError(t, c.Run(ctx, &kong.Context{
			Kong: &kong.Kong{
				Stdout: stdoutWriter,
			},
		}))
	}()
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})
	for stdoutScanner.Scan() {
		line := stdoutScanner.Text()
		if strings.Contains(line, "listening on") {
			port := 0
			_, err := fmt.Sscanf(line, "listening on 127.0.0.1:%d", &port)
			require.NoError(t, err)
			return port
		}
	}
	return 0
}

func startPluginServer(t *testing.T, p internalPlugins.Plugin) int {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0
	}
	server := grpc.NewServer()
	pluginv1.RegisterPluginServiceServer(server, internalPlugins.NewGrpcServer(p))
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NoError(t, server.Serve(listener))
	}()
	t.Cleanup(func() {
		server.Stop()
		wg.Wait()
	})
	return listener.Addr().(*net.TCPAddr).Port
}
