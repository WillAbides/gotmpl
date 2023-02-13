package plugins

import (
	"context"
	"fmt"
	"reflect"
	"text/template"
)

type tmplFunc func(args ...any) (any, error)

type funcMapPlugin struct {
	funcNames    []string
	wrappedFuncs map[string]tmplFunc
}

// NewFuncMapPlugin creates a plugin that provides functions from a text/template.FuncMap.
func NewFuncMapPlugin(funcMap template.FuncMap) Plugin {
	plugin := funcMapPlugin{
		wrappedFuncs: make(map[string]tmplFunc, len(funcMap)),
	}
	for name, fn := range funcMap {
		plugin.funcNames = append(plugin.funcNames, name)
		plugin.wrappedFuncs[name] = wrapTemplateFunction(fn)
	}
	return &plugin
}

func (f *funcMapPlugin) ListFunctions(_ context.Context) ([]string, error) {
	return f.funcNames, nil
}

func (f *funcMapPlugin) ExecuteFunction(_ context.Context, name string, args []any) (any, error) {
	fn, ok := f.wrappedFuncs[name]
	if !ok {
		return nil, fmt.Errorf("function %q not found", name)
	}
	return fn(args...)
}

// wrapTemplateFunction wraps a function (fn) in another function with the signature func(args ...any) (any, error).
// fn must return either 1 or 2 values, where the first value is the return value of the function and the second value
// is an error. If the function returns 1 value, the second value is assumed to be nil.
// fn may accept any number of arguments. When the returned function is called the arguments will be converted to
// the types that fn expects. It is an error if the number of arguments passed to the returned function does not
// match the number of arguments that fn accepts or if the arguments cannot be converted to the types that fn expects.
func wrapTemplateFunction(fn any) tmplFunc {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		panic(fmt.Sprintf("expected function, got %s", fnType))
	}
	if fnType.NumOut() != 1 && fnType.NumOut() != 2 {
		panic(fmt.Sprintf("expected function with 1 or 2 return values, got %d", fnType.NumOut()))
	}
	if fnType.NumOut() == 2 && fnType.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
		panic(fmt.Sprintf("expected second return value to be error, got %s", fnType.Out(1)))
	}
	return func(args ...any) (any, error) {
		if len(args) != fnType.NumIn() {
			return nil, fmt.Errorf("expected %d arguments, got %d", fnType.NumIn(), len(args))
		}
		in := make([]reflect.Value, len(args))
		for i := range args {
			arg := reflect.ValueOf(args[i])
			if !arg.Type().ConvertibleTo(fnType.In(i)) {
				return nil, fmt.Errorf("cannot convert argument %d to %s", i, fnType.In(i))
			}
			in[i] = arg.Convert(fnType.In(i))
		}
		out := reflect.ValueOf(fn).Call(in)
		if fnType.NumOut() == 1 {
			return out[0].Interface(), nil
		}
		if out[1].IsNil() {
			return out[0].Interface(), nil
		}
		return nil, out[1].Interface().(error)
	}
}
