//go:build wasm

package wasm

import (
	"errors"
	"fmt"
	"syscall/js"
)

var (
	ErrorInvalidArguments                    = errors.New("invalid number of arguments")
	ErrorUnhandledPanic                      = errors.New("unhandled panic")
	ErrorInvalidConfiguration                = errors.New("invalid configuration")
	ErrorFailedToLoadConfiguration           = errors.New("failed to load configuration")
	ErrorFailedToInitiaiizeCollector         = errors.New("failed to initialize collector")
	ErrorFailedToInitiaiizeSigner            = errors.New("failed to initialize signer")
	ErrorFailedToInitiaiizeNotificatorLoader = errors.New("failed to initialize notificator loader")
	ErrorInvalidPayoutOptions                = errors.New("invalid payout options")
	ErrorFailedToUnmarshalPayoutOptions      = errors.New("failed to unmarshal payout options")

	JsError   = js.Global().Get("Error")
	JsPromise = js.Global().Get("Promise")
)

type WasmExecutionResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   error       `json:"error,omitempty"`
}

func NewErrorResult(err error) WasmExecutionResult {
	return WasmExecutionResult{
		Success: false,
		Error:   err,
	}
}

func NewSuccessResult(data interface{}) WasmExecutionResult {
	return WasmExecutionResult{
		Success: true,
		Data:    data,
	}
}

func HasJsFunc(obj js.Value, name string) bool {
	f := obj.Get(name)
	return f.Type() == js.TypeFunction
}

func CallJsFunc(obj js.Value, funcId string, args ...interface{}) (js.Value, error) {
	if !HasJsFunc(obj, funcId) {
		return js.Null(), fmt.Errorf("function %s not found", funcId)
	}
	result := obj.Call(funcId, args...)
	if !result.InstanceOf(JsPromise) {
		return result, nil
	}
	resultChannel := make(chan js.Value, 1)
	errorChannel := make(chan error, 1)

	result.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resultChannel <- args[0]
		return nil
	}), js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		err := args[0]

		if err.InstanceOf(JsError) {
			errorChannel <- fmt.Errorf(err.Get("message").String())
		} else if err.Type() == js.TypeString {
			errorChannel <- fmt.Errorf(err.String())
		} else {
			errorChannel <- fmt.Errorf("unknown error")
		}
		return nil
	}))

	select {
	case err := <-errorChannel:
		return js.Null(), err
	case r := <-resultChannel:
		return r, nil
	}
}

func CallJsFuncExpectResultType(obj js.Value, funcId string, expectedType js.Type, args ...interface{}) (js.Value, error) {
	result, err := CallJsFunc(obj, funcId, args...)
	if err != nil {
		return js.Null(), err
	}
	if result.Type() != expectedType {
		return js.Null(), fmt.Errorf("%s returned invalid data", funcId)
	}
	return result, nil
}
