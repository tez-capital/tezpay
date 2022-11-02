package ops

import (
	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/clients/interfaces"
)

type OpExecutionContext struct {
	Op         *codec.Op
	Transactor interfaces.TransactorEngine
	opHash     tezos.OpHash
}

func InitOpExecutionContext(op *codec.Op, transactor interfaces.TransactorEngine) *OpExecutionContext {
	return &OpExecutionContext{
		Op:         op,
		opHash:     tezos.ZeroOpHash,
		Transactor: transactor,
	}
}

func (ctx *OpExecutionContext) GetOpHash() tezos.OpHash {
	return ctx.opHash
}

func (ctx *OpExecutionContext) Broadcast() error {
	opHash, err := ctx.Transactor.Broadcast(ctx.Op)
	if err != nil {
		return err
	}
	ctx.opHash = opHash
	return err
}

func (ctx *OpExecutionContext) WaitConfirmation(confirmations int64) (*rpc.Receipt, error) {
	return ctx.Transactor.WaitOpConfirmation(ctx.opHash, ctx.Op.TTL, confirmations)
}
