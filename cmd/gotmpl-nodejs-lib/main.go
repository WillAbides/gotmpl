//go:build js && wasm

package main

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"sync"
	"syscall/js"

	"github.com/willabides/gotmpl/internal"
	"github.com/willabides/gotmpl/internal/plugins"
)

func errMap(errMsg string) map[string]any {
	return map[string]any{
		"error": errMsg,
	}
}

func wasmExec(_ js.Value, args []js.Value) any {
	tmpl := args[0].String()
	data := jsValueInterface(args[1])
	missingkey := args[2].String()
	tmplPackage := args[3].String()
	cmdPlugins := jsValueStringSlice(args[4])
	grpcPlugins := jsValueStringSlice(args[5])
	jsFuncs := buildJsFuncs(args[6])

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	funcs, err := plugins.InitPlugins(ctx, cmdPlugins, grpcPlugins, nil)
	if err != nil {
		return errMap(fmt.Errorf("init plugins: %w", err).Error())
	}
	for k, v := range jsFuncs {
		funcs[k] = v
	}
	var buf bytes.Buffer
	err = internal.Execute(&buf, tmpl, data, &internal.ExecuteOptions{
		Funcs:      funcs,
		Missingkey: missingkey,
		Package:    tmplPackage,
	})
	if err != nil {
		return errMap(fmt.Errorf("executing template: %w", err).Error())
	}
	return map[string]any{
		"output": buf.String(),
	}
}

func wasmServer(ctx context.Context) js.Func {
	return funcOf(func(_ js.Value, args []js.Value) any {
		cmdPlugins := jsValueStringSlice(args[0])
		grpcPlugins := jsValueStringSlice(args[1])
		jsFuncs := buildJsFuncs(args[2])
		var mux sync.RWMutex
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		funcs, err := plugins.InitPlugins(ctx, cmdPlugins, grpcPlugins, nil)
		if err != nil {
			cancel()
			return errMap(fmt.Errorf("init plugins: %w", err).Error())
		}
		for k, v := range jsFuncs {
			funcs[k] = v
		}
		execute := serverExec(ctx, &mux, funcs)
		var stop js.Func
		stop = funcOf(func(_ js.Value, _ []js.Value) any {
			mux.Lock()
			defer mux.Unlock()
			cancel()
			stop.Release()
			execute.Release()
			return nil
		})
		return map[string]any{
			"stop":    stop,
			"execute": execute,
		}
	})
}

func serverExec(ctx context.Context, mux *sync.RWMutex, funcs map[string]any) js.Func {
	return funcOf(func(_ js.Value, args []js.Value) any {
		tmpl := args[0].String()
		data := jsValueInterface(args[1])
		missingkey := args[2].String()
		tmplPackage := args[3].String()
		mux.RLock()
		defer mux.RUnlock()
		if ctx.Err() != nil {
			return errMap("server stopped")
		}
		var buf bytes.Buffer
		err := internal.Execute(&buf, tmpl, data, &internal.ExecuteOptions{
			Funcs:      funcs,
			Missingkey: missingkey,
			Package:    tmplPackage,
		})
		if err != nil {
			return errMap(fmt.Errorf("executing template: %w", err).Error())
		}
		return map[string]any{
			"output": buf.String(),
		}
	})
}

func buildJsFuncs(arg js.Value) map[string]any {
	if arg.Type() != js.TypeObject {
		return map[string]any{}
	}
	var keys js.Value
	inGoroutine(func() {
		keys = js.Global().Get("Object").Call("keys", arg)
	})
	funcs := make(map[string]any, keys.Length())
	for i := 0; i < keys.Length(); i++ {
		key := keys.Index(i).String()
		fn := arg.Get(key)
		if fn.Type() != js.TypeFunction {
			continue
		}
		funcs[key] = wrapTemplateFunction(fn)
	}
	return funcs
}

type tmplFunc func(args ...any) (any, error)

// wrapTemplateFunction wraps a javascript value that represents a function (jsFunc) in a go function with the signature
// func(args ...any) (any, error). When invoking jsFunc throws an error, (nil, error) is returned. When jsFunc returns
// a value, (value, nil) is returned.
func wrapTemplateFunction(jsFunc js.Value) tmplFunc {
	return func(args ...any) (any, error) {
		got, err := invokeWithErr(jsFunc, args...)
		if err != nil {
			return nil, err
		}
		return got, nil
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	wasmExecFunc := funcOf(wasmExec)
	wasmServerFunc := wasmServer(ctx)
	js.Global().Set("gotmplExec", wasmExecFunc)
	js.Global().Set("gotmplServer", wasmServerFunc)
	js.Global().Set("gotmplStop", funcOf(func(_ js.Value, _ []js.Value) any {
		cancel()
		wasmExecFunc.Release()
		wasmServerFunc.Release()
		return nil
	}))
	<-ctx.Done()
}

func recoverPanic(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	fn()
	return err
}

func invokeWithErr(fn js.Value, args ...any) (any, error) {
	var jsResult js.Value
	jsArgs := make([]any, len(args))
	copy(jsArgs, args)
	err := recoverPanic(func() {
		jsResult = fn.Invoke(jsArgs...)
	})
	if err != nil {
		return nil, err
	}
	return jsValueInterface(jsResult), nil
}

func jsValueInterface(jsVal js.Value) any {
	switch jsVal.Type() {
	case js.TypeUndefined, js.TypeNull:
		return nil
	case js.TypeBoolean:
		return jsVal.Bool()
	case js.TypeNumber:
		f := jsVal.Float()
		if f == math.Trunc(f) {
			return jsVal.Int()
		}
		return jsVal.Float()
	case js.TypeString, js.TypeSymbol:
		return jsVal.String()
	case js.TypeObject:
		// return as either a map or a slice
		if jsVal.Get("length").Type() == js.TypeNumber {
			// slice
			length := jsVal.Get("length").Int()
			slice := make([]any, length)
			for i := 0; i < length; i++ {
				slice[i] = jsValueInterface(jsVal.Index(i))
			}
			return slice
		}
		// map
		var keys js.Value
		inGoroutine(func() {
			keys = js.Global().Get("Object").Call("keys", jsVal)
		})
		m := make(map[string]any, keys.Length())
		for i := 0; i < keys.Length(); i++ {
			key := keys.Index(i).String()
			m[key] = jsValueInterface(jsVal.Get(key))
		}
		return m
	case js.TypeFunction:
		return func(args ...any) (any, error) {
			return invokeWithErr(jsVal, args...)
		}
	default:
		panic(fmt.Sprintf("unknown js type: %s", jsVal.Type().String()))
	}
}

func jsValueStringSlice(v js.Value) []string {
	var ret []string
	if v.Type() != js.TypeObject {
		return ret
	}
	for i := 0; i < v.Length(); i++ {
		if v.Index(i).Type() != js.TypeString {
			continue
		}
		ret = append(ret, v.Index(i).String())
	}
	return ret
}

func inGoroutine(fn func()) {
	done := make(chan struct{})
	go func() {
		fn()
		close(done)
	}()
	<-done
}

func funcOf(f func(js.Value, []js.Value) any) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		var result any
		inGoroutine(func() {
			result = f(this, args)
		})
		return result
	})
}
