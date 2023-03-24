package transactor_engines

import (
	"context"
	"errors"
	"time"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/configuration"
	"github.com/alis-is/tezpay/engines/tzkt"
	"github.com/alis-is/tezpay/utils"
	log "github.com/sirupsen/logrus"
)

type DefaultRpcTransactor struct {
	rpcUrl string
	rpc    *rpc.Client
	tzkt   *tzkt.Client
}

type DefaultRpcTransactorOpResult struct {
	opHash tezos.OpHash
	result *rpc.Result
	rpc    *rpc.Client
	tzkt   *tzkt.Client
}

func (result *DefaultRpcTransactorOpResult) GetOpHash() tezos.OpHash {
	return result.opHash
}

func (result *DefaultRpcTransactorOpResult) WaitForApply() error {
	ctx, cancel := context.WithCancel(context.Background())
	utils.CallbackOnInterrupt(ctx, func() {
		log.Warnf("waiting for confirmation of '%s' canceled", result.opHash)
		cancel()
	})
	appliedChan := make(chan common.OperationStatus, 1)
	go func() {
		utils.SleepContext(ctx, 130*time.Second) //give monitor 4 blocks before fallback kicks in
		if ctx.Err() != context.Canceled {
			log.Debug(`failed to confirm with live monitoring, falling back to polling...`)
		}
		for ctx.Err() != context.Canceled {
			applied, _ := result.tzkt.WasOperationApplied(ctx, result.opHash)
			log.Debugf("operation '%s' status check result: %s", result.opHash, applied)
			if applied == common.OPERATION_STATUS_APPLIED || applied == common.OPERATION_STATUS_FAILED {
				cancel()
				appliedChan <- applied
				break
			}
			time.Sleep(15 * time.Second)
		}
		close(appliedChan)
	}()
	result.result.WaitContext(ctx)
	cancel() // cancel fallback
	switch <-appliedChan {
	case common.OPERATION_STATUS_FAILED:
		return errors.New("operation failed")
	case common.OPERATION_STATUS_APPLIED:
		return nil
	}
	if err := result.result.Err(); err != nil {
		return err
	}
	rcpt, err := result.result.GetReceipt(context.Background())
	result.rpc.Close()
	if err != nil {
		return err
	}
	if rcpt.IsSuccess() {
		return nil
	}
	return rcpt.Error()
}

func InitDefaultTransactor(config *configuration.RuntimeConfiguration) (*DefaultRpcTransactor, error) {
	rpcClient, err := rpc.NewClient(config.Network.RpcUrl, nil)
	if err != nil {
		return nil, err
	}

	tzktClient, err := tzkt.InitClient(config.Network.TzktUrl, nil)
	if err != nil {
		return nil, err
	}

	result := &DefaultRpcTransactor{
		rpcUrl: config.Network.RpcUrl,
		rpc:    rpcClient,
		tzkt:   tzktClient,
	}
	return result, result.RefreshParams()
}

func (transactor *DefaultRpcTransactor) GetId() string {
	return "DefaultRpcTransactor"
}

func (transactor *DefaultRpcTransactor) RefreshParams() error {
	return transactor.rpc.Init(context.Background())
}

func (transactor *DefaultRpcTransactor) GetNewRpcClient() (*rpc.Client, error) {
	client, err := rpc.NewClient(transactor.rpcUrl, transactor.rpc.Client())
	client.ChainId = transactor.rpc.ChainId
	client.Params = transactor.rpc.Params
	if err != nil {
		return nil, err
	}
	return client, nil
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
	op = op.WithParams(transactor.rpc.Params)
	err := transactor.rpc.Complete(context.Background(), op, key)
	return err
}

func (transactor *DefaultRpcTransactor) initOpResult(opHash tezos.OpHash, opts *rpc.CallOptions) (*DefaultRpcTransactorOpResult, error) {
	if opts == nil {
		opts = &rpc.DefaultOptions
	}
	rpcClient, err := transactor.GetNewRpcClient()
	if err != nil {
		return nil, err
	}
	err = rpcClient.Init(context.Background())
	if err != nil {
		return nil, err
	}
	rpcClient.Listen()
	res := rpc.NewResult(opHash).WithTTL(opts.TTL).WithConfirmations(opts.Confirmations)
	res.Listen(rpcClient.BlockObserver)
	return &DefaultRpcTransactorOpResult{
		opHash: opHash,
		result: res,
		rpc:    rpcClient,
		tzkt:   transactor.tzkt,
	}, nil
}

func (transactor *DefaultRpcTransactor) Broadcast(op *codec.Op) (tezos.OpHash, error) {
	return transactor.rpc.Broadcast(context.Background(), op)
}

func (transactor *DefaultRpcTransactor) Dispatch(op *codec.Op, opts *rpc.CallOptions) (common.OpResult, error) {
	if opts == nil {
		opts = &rpc.DefaultOptions
	}
	opHash, err := transactor.Broadcast(op)
	if err != nil {
		return nil, err
	}
	result, err := transactor.initOpResult(opHash, opts)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (transactor *DefaultRpcTransactor) Send(op *codec.Op, opts *rpc.CallOptions) (*rpc.Receipt, error) {
	return transactor.rpc.Send(context.Background(), op, opts)
}

func (transactor *DefaultRpcTransactor) WaitOpConfirmation(opHash tezos.OpHash, ttl int64, confirmations int64) (*rpc.Receipt, error) {
	ctx, cancel := context.WithCancel(context.Background())
	res := rpc.NewResult(opHash).WithTTL(ttl).WithConfirmations(confirmations)
	transactor.rpc.Listen()
	res.Listen(transactor.rpc.BlockObserver)
	utils.CallbackOnInterrupt(ctx, func() {
		log.Warnf("waiting for confirmation of '%s' canceled", opHash)
		cancel()
	})
	res.WaitContext(ctx)
	if err := res.Err(); err != nil {
		return nil, err
	}

	// return receipt
	return res.GetReceipt(context.Background())
}
