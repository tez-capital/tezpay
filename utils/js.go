//go:build js && wasm

package utils

import (
	"fmt"
	"syscall/js"
)

func HasJsFunc(obj js.Value, name string) bool {
	f := obj.Get(name)
	return f.Type() == js.TypeFunction
}

func CallJsFunc(obj js.Value, funcId string, args ...interface{}) (js.Value, error) {
	if !HasJsFunc(obj, funcId) {
		return js.Null(), fmt.Errorf("function %s not found", funcId)
	}
	return obj.Call(funcId, args...), nil
}
