package clients

import (
	"context"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/utils"
	"github.com/sirupsen/logrus"
)

type DefaultRpcTransactor struct {
	rpc    *rpc.Client
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
		Cancel: cancel,
	}, nil
}

func (transactor *DefaultRpcTransactor) GetId() string {
	return "DefaultRpcTransactor"
}

func (transactor *DefaultRpcTransactor) GetLimits() (*common.OperationLimits, error) {
	params, err := transactor.rpc.GetParams(context.Background(), rpc.NewBlockOffset(rpc.Head, 0))
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
	err := transactor.rpc.Complete(context.Background(), op, key)
	return err
}

func (transactor *DefaultRpcTransactor) Broadcast(op *codec.Op) (tezos.OpHash, error) {
	return transactor.rpc.Broadcast(context.Background(), op)
}

func (transactor *DefaultRpcTransactor) Send(op *codec.Op, opts *rpc.CallOptions) (*rpc.Receipt, error) {
	return transactor.rpc.Send(context.Background(), op, opts)
}

func (transactor *DefaultRpcTransactor) WaitOpConfirmation(opHash tezos.OpHash, ttl int64, confirmations int64) (*rpc.Receipt, error) {
	ctx, cancel := context.WithCancel(context.Background())
	res := rpc.NewResult(opHash).WithTTL(ttl).WithConfirmations(confirmations)
	transactor.rpc.Listen()
	res.Listen(transactor.rpc.BlockObserver)
	utils.CallbackOnInterrupt(func() {
		logrus.Warnf("waiting for confirmation of '%s' canceled", opHash)
		cancel()
	})
	res.WaitContext(ctx)
	if err := res.Err(); err != nil {
		return nil, err
	}

	// return receipt
	return res.GetReceipt(context.Background())
}
