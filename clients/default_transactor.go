package clients

import (
	"context"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/common"
)

type DefaultRpcTransactor struct {
	rpc    *rpc.Client
	ctx    context.Context
	Cancel context.CancelFunc
}

func InitDefaultTransactor(rpcUrl string) (*DefaultRpcTransactor, error) {
	rpcClient, err := rpc.NewClient(rpcUrl, nil)
	if err != nil {
		return nil, err
	}
	chainId, err := rpcClient.GetChainId(context.Background())
	if err != nil {
		return nil, err
	}
	rpcClient.ChainId = chainId
	ctx, cancel := context.WithCancel(context.Background())
	rpcClient.Init(ctx)

	return &DefaultRpcTransactor{
		rpc:    rpcClient,
		ctx:    ctx,
		Cancel: cancel,
	}, nil
}

func (transactor *DefaultRpcTransactor) GetId() string {
	return "DefaultRpcTransactor"
}

func (transactor *DefaultRpcTransactor) GetLimits() (*common.OperationLimits, error) {
	params, err := transactor.rpc.GetParams(transactor.ctx, rpc.NewBlockOffset(rpc.Head, 0))
	if err != nil {
		return nil, err
	}
	return &common.OperationLimits{
		HardGasLimitPerOperation:     params.HardGasLimitPerBlock,
		HardStorageLimitPerOperation: params.HardStorageLimitPerOperation,
		MaxOperationDataLength:       params.MaxOperationDataLength,
	}, nil
}

func (transactor *DefaultRpcTransactor) Complete(op *codec.Op, key tezos.Key) error {
	err := transactor.rpc.Complete(transactor.ctx, op, key)
	return err
}

func (transactor *DefaultRpcTransactor) Broadcast(op *codec.Op) (tezos.OpHash, error) {
	return transactor.rpc.Broadcast(transactor.ctx, op)
}

func (transactor *DefaultRpcTransactor) WaitOpConfirmation(opHash tezos.OpHash, ttl int64, confirmations int64) (*rpc.Receipt, error) {
	res := rpc.NewResult(opHash).WithTTL(ttl).WithConfirmations(confirmations)
	transactor.rpc.Listen()
	res.Listen(transactor.rpc.BlockObserver)
	res.WaitContext(transactor.ctx)
	if err := res.Err(); err != nil {
		return nil, err
	}

	// return receipt
	return res.GetReceipt(transactor.ctx)
}
