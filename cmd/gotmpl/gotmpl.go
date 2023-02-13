package main

import (
	"context"
	"fmt"
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

type serverCmd struct {
	Plugin  []string `kong:"short=p,type=path"`
	Address string   `kong:"help='address to listen on',default='localhost:8080'"`
}

func (c *serverCmd) Run(ctx context.Context) (errOut error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	funcs, err := plugins.StartPlugins(ctx, c.Plugin, nil)
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
		if errOut == nil {
			errOut = e
		}
	}()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		sig := <-sigs
		fmt.Printf("received signal %q, shutting down\n", sig)
		cancel()
	}()
	err = srv.ListenAndServe()
	if err == http.ErrServerClosed {
		err = nil
	}
	return err
}

type execCmd struct {
	Plugin     []string `kong:"short=p,type=path"`
	Missingkey string
	Package    string
	Template   string `kong:"arg,required"`
	Data       string `kong:"arg"`
}

func (c *execCmd) Run(ctx context.Context) error {
	funcs, err := plugins.StartPlugins(ctx, c.Plugin, nil)
	if err != nil {
		return err
	}
	return internal.Execute(os.Stdout, c.Template, c.Data, &internal.ExecuteOptions{
		Funcs:      funcs,
		Missingkey: c.Missingkey,
		Package:    c.Package,
	})
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
