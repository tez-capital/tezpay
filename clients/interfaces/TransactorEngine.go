package interfaces

import (
	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/common"
)

type TransactorEngine interface {
	GetId() string
	Complete(op *codec.Op, key tezos.Key) error
	Broadcast(op *codec.Op) (tezos.OpHash, error)
	GetLimits() (*common.OperationLimits, error)
	WaitOpConfirmation(opHash tezos.OpHash, ttl int64, confirmations int64) (*rpc.Receipt, error)
}
