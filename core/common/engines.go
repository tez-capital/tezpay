package common

import (
	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/signer"
	"blockwatch.cc/tzgo/tezos"
)

type CollectorEngine interface {
	GetId() string
	GetCurrentCycleNumber() (int64, error)
	GetLastCompletedCycle() (int64, error)
	GetCycleData(baker tezos.Address, cycle int64) (*BakersCycleData, error)
	WasOperationApplied(opHash tezos.OpHash) (bool, error)
	GetBranch(offset int64) (tezos.BlockHash, error)
	Simulate(o *codec.Op, publicKey tezos.Key) (*rpc.Receipt, error)
	GetBalance(pkh tezos.Address) (tezos.Z, error)
}

type SignerEngine interface {
	GetId() string
	Sign(op *codec.Op) error
	GetPKH() tezos.Address
	GetKey() tezos.Key
	GetSigner() signer.Signer
}

type TransactorEngine interface {
	GetId() string
	Complete(op *codec.Op, key tezos.Key) error
	Broadcast(op *codec.Op) (tezos.OpHash, error)
	Send(op *codec.Op, opts *rpc.CallOptions) (*rpc.Receipt, error)
	GetLimits() (*OperationLimits, error)
	WaitOpConfirmation(opHash tezos.OpHash, ttl int64, confirmations int64) (*rpc.Receipt, error)
}
