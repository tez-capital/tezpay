package common

import (
	"errors"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
)

type OpExecutionContext struct {
	Op         *codec.Op
	Transactor TransactorEngine
	result     OpResult
}

func InitOpExecutionContext(op *codec.Op, transactor TransactorEngine) *OpExecutionContext {
	return &OpExecutionContext{
		Op:         op,
		result:     nil,
		Transactor: transactor,
	}
}

func (ctx *OpExecutionContext) GetOpHash() tezos.OpHash {
	if ctx.result == nil {
		return tezos.ZeroOpHash
	}
	return ctx.result.GetOpHash()
}

func (ctx *OpExecutionContext) Dispatch(opts *rpc.CallOptions) error {
	result, err := ctx.Transactor.Dispatch(ctx.Op, opts)
	if err != nil {
		return err
	}
	ctx.result = result
	return err
}

func (ctx *OpExecutionContext) WaitForApply() error {
	if ctx.result == nil {
		return errors.New("operation was not dispatched yet")
	}
	return ctx.result.WaitForApply()
}
