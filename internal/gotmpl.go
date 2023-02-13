package internal

import (
	htmlTemplate "html/template"
	"io"
	"text/template"
)

//go:generate ../script/buf generate

const (
	TextPackage string = "text"
	HtmlPackage string = "html"
)

const (
	MissingkeyInvalid string = "invalid"
	MissingkeyZero    string = "zero"
	MissingkeyError   string = "error"
)

type ExecuteOptions struct {
	// Package is the template package to use. Defaults to TextPackage.
	Package string

	// Missingkey is the action to take when a template references a key that is not present in the data.
	// Defaults to MissingkeyInvalid.
	Missingkey string

	Funcs template.FuncMap
}

func Execute(w io.Writer, tmpl string, data any, opts *ExecuteOptions) error {
	var options ExecuteOptions
	if opts != nil {
		options = *opts
	}
	var funcs template.FuncMap
	if opts != nil {
		funcs = opts.Funcs
	}
	missingKeyOption := "missingkey=invalid"
	if options.Missingkey != "" {
		missingKeyOption = "missingkey=" + options.Missingkey
	}
	if options.Package == HtmlPackage {
		return executeHtmlTemplate(w, tmpl, data, funcs, missingKeyOption)
	}
	return executeTextTemplate(w, tmpl, data, funcs, missingKeyOption)
}

func executeTextTemplate(w io.Writer, tmpl string, data any, funcs template.FuncMap, missingKeyOption string) error {
	t, err := template.New("").Option(missingKeyOption).Funcs(funcs).Parse(tmpl)
	if err != nil {
		return err
	}
	return t.Execute(w, data)
}

func executeHtmlTemplate(w io.Writer, tmpl string, data any, funcs template.FuncMap, option string) error {
	t, err := htmlTemplate.New("").Option(option).Funcs(funcs).Parse(tmpl)
	if err != nil {
		return err
	}
	return t.Execute(w, data)
}
