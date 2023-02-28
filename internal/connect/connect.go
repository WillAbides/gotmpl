package connect

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/bufbuild/connect-go"
	"github.com/willabides/gotmpl/internal"
	gotmplv1 "github.com/willabides/gotmpl/internal/gen/proto/go/gotmpl/v1"
)

type ConnectHandler struct {
	Funcs template.FuncMap
}

func (s *ConnectHandler) Execute(
	_ context.Context,
	req *connect.Request[gotmplv1.ExecuteRequest],
) (*connect.Response[gotmplv1.ExecuteResponse], error) {
	request := req.Msg
	if request.Template == "" {
		return nil, fmt.Errorf("template is required")
	}
	opts := internal.ExecuteOptions{
		Funcs: s.Funcs,
	}
	if request.Package != nil {
		switch *request.Package {
		case internal.TextPackage, internal.HtmlPackage:
			opts.Package = *request.Package
		default:
			return nil, fmt.Errorf("invalid package: %s", *request.Package)
		}
	}
	if request.Missingkey != nil {
		switch *request.Missingkey {
		case internal.MissingkeyInvalid, internal.MissingkeyZero, internal.MissingkeyError:
			opts.Missingkey = *request.Missingkey
		default:
			return nil, fmt.Errorf("invalid missingkey: %s", *request.Missingkey)
		}
	}
	var buf bytes.Buffer
	err := internal.Execute(&buf, request.Template, request.Data.AsInterface(), &opts)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&gotmplv1.ExecuteResponse{
		Result: buf.String(),
	}), nil
}
