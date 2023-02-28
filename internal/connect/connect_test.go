package connect

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/require"
	gotmplv1 "github.com/willabides/gotmpl/internal/gen/proto/go/gotmpl/v1"
	"github.com/willabides/gotmpl/internal/plugins"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestConnectHandlerExecute(t *testing.T) {
	cmd := exec.Command("make", "bin/exampleplugin")
	cmd.Dir = filepath.FromSlash("../../")
	err := cmd.Run()
	require.NoError(t, err)
	ctx := context.Background()
	funcMap, err := plugins.InitPlugins(
		ctx,
		[]string{filepath.FromSlash("../../bin/exampleplugin")},
		nil,
		&plugins.InitPluginsOptions{ExecTimeout: 1 * time.Minute},
	)
	require.NoError(t, err)
	handler := &ConnectHandler{
		Funcs: funcMap,
	}
	data, err := structpb.NewValue(map[string]any{
		"foo": "bar",
	})
	require.NoError(t, err)
	req := connect.NewRequest(&gotmplv1.ExecuteRequest{
		Template: "hello {{ upcase .foo }}",
		Data:     data,
	})
	resp, err := handler.Execute(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "hello BAR", resp.Msg.Result)
}
