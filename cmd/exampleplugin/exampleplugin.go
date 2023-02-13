package main

import (
	"strings"
	"text/template"

	"github.com/willabides/gotmpl/plugins"
)

func main() {
	plugin := plugins.NewFuncMapPlugin(template.FuncMap{
		"upcase":   strings.ToUpper,
		"downcase": strings.ToLower,
	})
	plugins.ServePlugin(plugin)
}
