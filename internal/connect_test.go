package internal

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/require"
	gotmplv1 "github.com/willabides/gotmpl/internal/gen/proto/go/gotmpl/v1"
	"github.com/willabides/gotmpl/internal/plugins"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestConnectHandlerExecute(t *testing.T) {
	cmd := exec.Command("make", "bin/exampleplugin", "bin/gotmpl-sprig")
	cmd.Dir = "../"
	err := cmd.Run()
	require.NoError(t, err)
	ctx := context.Background()
	funcMap, err := plugins.StartPlugins(ctx, []string{"../bin/exampleplugin", "../bin/gotmpl-sprig"}, &plugins.StartPluginsOptions{
		Timeout: 1 * time.Minute,
	})
	require.NoError(t, err)
	handler := &ConnectHandler{
		Funcs: funcMap,
	}
	data, err := structpb.NewValue(map[string]any{
		"foo": "bar",
	})
	require.NoError(t, err)
	req := connect.NewRequest(&gotmplv1.ExecuteRequest{
		Template: "hello {{ upcase .foo | b64enc }}",
		Data:     data,
	})
	resp, err := handler.Execute(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "hello QkFS", resp.Msg.Result)
}
