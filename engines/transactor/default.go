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
	rpc_urls []string
	rpcs     []*rpc.Client
	tzkt     *tzkt.Client
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
			applied, _ := result.tzkt.WasOperationApplied(ctx, result.opHash)
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

	rpc_clients, err := utils.InitializeRpcClients(context.Background(), config.Network.RpcPool, http_client)
	if err != nil {
		return nil, err
	}

	tzktClient, err := tzkt.InitClient(config.Network.TzktUrl, config.Network.ProtocolRewardsUrl, &tzkt.TzktClientOptions{
		HttpClient:       http_client,
		BalanceCheckMode: config.PayoutConfiguration.BalanceCheckMode,
	})
	if err != nil {
		return nil, err
	}

	result := &DefaultRpcTransactor{
		rpc_urls: config.Network.RpcPool,
		rpcs:     rpc_clients,
		tzkt:     tzktClient,
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
			slog.Warn("failed to refresh rpc params", "error", err.Error(), "rpc_url", rpc.BaseURL.String())
			failures++
		}
	}
	if failures == len(transactor.rpcs) {
		return fmt.Errorf("failed to refresh rpc params for all clients, all %d failed", failures)
	} else if failures > 0 {
		slog.Info(">>> at least one RPC client was successfully refreshed - you can ignore above warnings <<<")
	}

	return nil
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
	_, err := utils.AttemptWithRpcClients(context.Background(), transactor.rpcs, func(client *rpc.Client) (bool, error) {
		op = op.WithParams(client.Params)
		err := client.Complete(context.Background(), op, key)
		if err == nil {
			return true, nil
		}
		return false, err
	})
	return err
}

func (transactor *DefaultRpcTransactor) initOpResult(opHash tezos.OpHash, opts *rpc.CallOptions) (*DefaultRpcTransactorOpResult, error) {
	if opts == nil {
		opts = &rpc.DefaultOptions
	}

	rpc_client, err := utils.InitializeSingleRpcFromRpcPool(context.Background(), transactor.rpc_urls, &http.Client{
		Timeout: 10 * 60 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	err = rpc_client.Init(context.Background())
	if err != nil {
		return nil, err
	}
	rpc_client.Listen()
	res := rpc.NewResult(opHash).WithTTL(opts.TTL).WithConfirmations(opts.Confirmations)
	res.Listen(rpc_client.BlockObserver)
	return &DefaultRpcTransactorOpResult{
		opHash: opHash,
		result: res,
		rpc:    rpc_client,
		tzkt:   transactor.tzkt,
	}, nil
}

func (transactor *DefaultRpcTransactor) broadcast(op *codec.Op) (tezos.OpHash, error) {
	return utils.AttemptWithRpcClients(context.Background(), transactor.rpcs, func(client *rpc.Client) (tezos.OpHash, error) {
		return client.Broadcast(context.Background(), op)
	})
}

func (transactor *DefaultRpcTransactor) Dispatch(op *codec.Op, opts *rpc.CallOptions) (common.OpResult, error) {
	if opts == nil {
		opts = &rpc.DefaultOptions
	}
	opHash, err := transactor.broadcast(op)
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
	return utils.AttemptWithRpcClients(context.Background(), transactor.rpcs, func(client *rpc.Client) (*rpc.Receipt, error) {
		return client.Send(context.Background(), op, opts)
	})
}
