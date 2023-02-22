package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/alecthomas/kong"
	"github.com/willabides/gotmpl/internal"
	"github.com/willabides/gotmpl/internal/gen/proto/go/gotmpl/v1/gotmplv1connect"
	"github.com/willabides/gotmpl/internal/plugins"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type pluginData struct {
	CommandPlugin []string `kong:"short=c"`
	GrpcPlugin    []string `kong:"short=g"`
}

type execCmd struct {
	pluginData `kong:",embed"`
	Missingkey string
	Package    string
	Template   string `kong:"arg,required"`
	Data       string `kong:"arg,required"`
}

func (c *execCmd) Run(ctx context.Context, k *kong.Context) error {
	funcs, err := plugins.InitPlugins(ctx, c.CommandPlugin, c.GrpcPlugin, nil)
	if err != nil {
		return err
	}
	var data any
	err = json.Unmarshal([]byte(c.Data), &data)
	if err != nil {
		return err
	}
	return internal.Execute(k.Stdout, c.Template, data, &internal.ExecuteOptions{
		Funcs:      funcs,
		Missingkey: c.Missingkey,
		Package:    c.Package,
	})
}

type serverCmd struct {
	pluginData `kong:",embed"`
	Address    string `kong:"help='address to listen on',default='localhost:8080'"`
}

func (c *serverCmd) Run(ctx context.Context, k *kong.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	funcs, err := plugins.InitPlugins(ctx, c.CommandPlugin, c.GrpcPlugin, nil)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle(gotmplv1connect.NewGotmplServiceHandler(&internal.ConnectHandler{
		Funcs: funcs,
	}))
	srv := &http.Server{
		Addr:        c.Address,
		Handler:     h2c.NewHandler(mux, &http2.Server{}),
		ReadTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		e := srv.Shutdown(ctx)
		if e != nil {
			fmt.Fprintf(k.Stderr, "error shutting down server: %s", e)
		}
	}()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		sig := <-sigs
		fmt.Fprintf(k.Stderr, "received signal %q, shutting down\n", sig)
		cancel()
	}()
	listener, err := net.Listen("tcp", c.Address)
	if err != nil {
		return err
	}
	fmt.Fprintf(k.Stdout, "listening on %s\n", listener.Addr())
	err = srv.Serve(listener)
	if err == http.ErrServerClosed {
		err = nil
	}
	return err
}

type Cmd struct {
	Server serverCmd `kong:"cmd,help='start a server'"`
	Exec   execCmd   `kong:"cmd,help='execute a template'"`
}

func main() {
	ctx := context.Background()
	k := kong.Parse(&Cmd{}, kong.Bind(ctx))
	k.BindTo(ctx, (*context.Context)(nil))
	k.FatalIfErrorf(k.Run())
}
