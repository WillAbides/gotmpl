package main

import (
	"github.com/Masterminds/sprig/v3"
	"github.com/willabides/gotmpl/plugins"
)

func main() {
	plugin := plugins.NewFuncMapPlugin(sprig.TxtFuncMap())
	plugins.ServePlugin(plugin)
}
