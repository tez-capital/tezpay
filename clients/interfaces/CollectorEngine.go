package interfaces

import (
	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/common"
)

type CollectorEngine interface {
	GetId() string
	GetCurrentCycleNumber() (int64, error)
	GetLastCompletedCycle() (int64, error)
	GetCycleData(baker tezos.Address, cycle int64) (*common.BakersCycleData, error)
	WasOperationApplied(opHash tezos.OpHash) (bool, error)
	GetBranch(offset int64) (tezos.BlockHash, error)
	Simulate(o *codec.Op, publicKey tezos.Key) (*rpc.Receipt, error)
	GetBalance(pkh tezos.Address) (tezos.Z, error)
}
