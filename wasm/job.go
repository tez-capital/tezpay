//go:build wasm

package wasm

import (
	"syscall/js"
)

type Job struct {
	Id            string
	This          js.Value
	Args          []js.Value
	ResultChannel chan WasmExecutionResult
}

func NewJob(id string, this js.Value, args []js.Value) Job {
	return Job{
		Id:            id,
		This:          this,
		Args:          args,
		ResultChannel: make(chan WasmExecutionResult),
	}
}

func (job *Job) GetPromise() interface{} {
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve := args[0]
		reject := args[1]
		go func() {
			result := <-job.ResultChannel
			if result.Success {
				resolve.Invoke(result.Data)
			} else {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(result.Error.Error())
				reject.Invoke(errorObject)
			}
		}()

		return nil
	})
	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}
