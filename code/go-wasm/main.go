// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build wasm

package main

import (
	"bytes"
	"fmt"
	"syscall/js"

	"github.com/elastic/package-spec/code/go/pkg/validator"
)

const moduleName = "elasticPackageSpec"

// asyncFunc helps creating functions that return a promise.
//
// Calling async JavaScript APIs causes deadlocks in the JS event loop. Not sure
// how to find if a Go code does it, but for example ValidateFromZip does, so
// we need to run this code in a goroutine and return the result as a promise.
// Related: https://github.com/golang/go/issues/41310
func asyncFunc(fn func(this js.Value, args []js.Value) interface{}) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler := js.FuncOf(func(_ js.Value, handlerArgs []js.Value) interface{} {
			resolve := handlerArgs[0]
			reject := handlerArgs[1]

			go func() {
				result := fn(this, args)
				if err, ok := result.(error); ok {
					reject.Invoke(err.Error())
					return
				}
				resolve.Invoke(result)
			}()

			return nil
		})

		return js.Global().Get("Promise").New(handler)
	})
}

func main() {
	// It doesn't seem to be possible yet to export values as part of the compiled instance.
	// So we have to expose it by setting a global value. It may worth to explore tinygo for this.
	// Related: https://github.com/golang/go/issues/42372
	js.Global().Set(moduleName, make(map[string]interface{}))
	module := js.Global().Get(moduleName)
	module.Set("validateFromZip", asyncFunc(
		func(this js.Value, args []js.Value) interface{} {
			if len(args) == 0 || args[0].IsNull() || args[0].IsUndefined() {
				return fmt.Errorf("package path expected")
			}

			pkgPath := args[0].String()
			return validator.ValidateFromZip(pkgPath)
		},
	))

	module.Set("validateFromZipReader", asyncFunc(
		func(this js.Value, args []js.Value) interface{} {
			if len(args) < 1 || args[0].Type() != js.TypeString {
				return fmt.Errorf("string expected")
			}
			if len(args) < 2 || args[1].Type() != js.TypeNumber {
				return fmt.Errorf("number expected")
			}
			if len(args) < 3 || !args[2].InstanceOf(js.Global().Get("Uint8Array")) {
				return fmt.Errorf("array buffer with content of package expected")
			}

			name := args[0].String()
			size := int64(args[1].Int())

			buf := make([]byte, size)
			js.CopyBytesToGo(buf, args[2])

			reader := bytes.NewReader(buf)
			return validator.ValidateFromZipReader(name, reader, size)
		},
	))

	// Go runtime must be always available at any moment where exported functionality
	// can be executed, so keep it running till done.
	done := make(chan struct{})
	module.Set("stop", js.FuncOf(func(_ js.Value, _ []js.Value) interface{} {
		close(done)
		return nil
	}))
	<-done
}
