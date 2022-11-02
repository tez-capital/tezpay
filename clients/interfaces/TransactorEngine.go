package interfaces

import (
	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
	tezpay_tezos "github.com/alis-is/tezpay/clients/tezos"
)

type TransactorEngine interface {
	GetId() string
	Complete(op *codec.Op, key tezos.Key) error
	Broadcast(op *codec.Op) (tezos.OpHash, error)
	GetLimits() (*tezpay_tezos.OperationLimits, error)
	WaitOpConfirmation(opHash tezos.OpHash, ttl int64, confirmations int64) (*rpc.Receipt, error)
}
