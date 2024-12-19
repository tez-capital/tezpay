package transactor_engines

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/engines/tzkt"
	"github.com/tez-capital/tezpay/utils"
	"github.com/trilitech/tzgo/codec"
	"github.com/trilitech/tzgo/rpc"
	"github.com/trilitech/tzgo/tezos"
)

type DefaultRpcTransactor struct {
	rpcUrl string
	rpcs   []*rpc.Client
	tzkt   *tzkt.Client
}

type DefaultRpcTransactorOpResult struct {
	opHash     tezos.OpHash
	result     *rpc.Result
	transactor *DefaultRpcTransactor
}

func (result *DefaultRpcTransactorOpResult) GetOpHash() tezos.OpHash {
	return result.opHash
}

func (result *DefaultRpcTransactorOpResult) WaitForApply() error {
	ctx, cancel := context.WithCancel(context.Background())
	utils.CallbackOnInterrupt(ctx, func() {
		slog.Warn("waiting for confirmation canceled", "op_hash", result.opHash)
		cancel()
	})
	appliedChan := make(chan common.OperationStatus, 1)
	go func() {
		utils.SleepContext(ctx, 130*time.Second) //give monitor 4 blocks before fallback kicks in
		if ctx.Err() != context.Canceled {
			slog.Debug(`failed to confirm with live monitoring, falling back to polling`)
		}
		for ctx.Err() != context.Canceled {
			applied, _ := result.transactor.tzkt.WasOperationApplied(ctx, result.opHash)
			slog.Debug("operation status checked", "op_hash", result.opHash, "applied", applied)
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
		return constants.ErrOperationFailed
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
	http_client := &http.Client{
		Timeout: 10 * 60 * time.Second,
	}

	rpc_clients := make([]*rpc.Client, 0, len(config.Network.RpcPool))
	failures := 0
	for _, rpcUrl := range config.Network.RpcPool {
		rpc_client, err := rpc.NewClient(rpcUrl, http_client)
		if err != nil {
			slog.Warn("failed to create rpc client", "url", rpcUrl, "error", err.Error())
			failures++
			continue
		}
		rpc_clients = append(rpc_clients, rpc_client)
	}
	if len(rpc_clients) == 0 {
		return nil, fmt.Errorf("failed to create rpc clients, all %d failed", failures)
	}

	tzktClient, err := tzkt.InitClient(config.Network.TzktUrl, config.Network.ProtocolRewardsUrl, &tzkt.TzktClientOptions{
		HttpClient:       http_client,
		BalanceCheckMode: config.PayoutConfiguration.BalanceCheckMode,
	})
	if err != nil {
		return nil, err
	}

	result := &DefaultRpcTransactor{
		rpcs: rpc_clients,
		tzkt: tzktClient,
	}
	return result, result.RefreshParams()
}

func (transactor *DefaultRpcTransactor) GetId() string {
	return "DefaultRpcTransactor"
}

func (transactor *DefaultRpcTransactor) RefreshParams() error {
	failures := 0
	for _, rpc := range transactor.rpcs {
		err := rpc.Init(context.Background())
		if err != nil {
			slog.Warn("failed to refresh rpc params", "error", err.Error())
			failures++
		}
	}
	if failures == len(transactor.rpcs) {
		return fmt.Errorf("failed to refresh rpc params for all clients, all %d failed", failures)
	}

	return nil
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
	params, err := utils.AttemptWithRpcClients(context.Background(), transactor.rpcs, func(client *rpc.Client) (*tezos.Params, error) {
		return client.GetParams(context.Background(), rpc.Head)
	})
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
		opHash:     opHash,
		result:     res,
		transactor: transactor,
	}, nil
}

func (transactor *DefaultRpcTransactor) Broadcast(op *codec.Op) (tezos.OpHash, error) {
	return utils.AttemptWithRpcClients(context.Background(), transactor.rpcs, func(client *rpc.Client) (tezos.OpHash, error) {
		return client.Broadcast(context.Background(), op)
	})
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
		slog.Warn("waiting for confirmation canceled", "op_hash", opHash)
		cancel()
	})
	res.WaitContext(ctx)
	if err := res.Err(); err != nil {
		return nil, err
	}

	// return receipt
	return res.GetReceipt(context.Background())
}
