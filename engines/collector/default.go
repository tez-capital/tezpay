package collector_engines

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/engines/tzkt"
	"github.com/tez-capital/tezpay/utils"
	"github.com/trilitech/tzgo/codec"
	"github.com/trilitech/tzgo/rpc"
	"github.com/trilitech/tzgo/tezos"
)

type DefaultRpcAndTzktColletor struct {
	rpcs []*rpc.Client
	tzkt *tzkt.Client
}

var (
	defaultCtx context.Context = context.Background()
)

func InitDefaultRpcAndTzktColletor(config *configuration.RuntimeConfiguration) (*DefaultRpcAndTzktColletor, error) {
	http_client := &http.Client{
		Timeout: 10 * time.Second,
	}

	rpc_clients, err := utils.InitializeRpcClients(context.Background(), config.Network.RpcPool, http_client)
	if err != nil {
		return nil, err
	}

	tzkt_client, err := tzkt.InitClient(config.Network.TzktUrl, config.Network.ProtocolRewardsUrl, &tzkt.TzktClientOptions{
		HttpClient:       http_client,
		BalanceCheckMode: config.PayoutConfiguration.BalanceCheckMode,
	})
	if err != nil {
		return nil, err
	}

	result := &DefaultRpcAndTzktColletor{
		rpcs: rpc_clients,
		tzkt: tzkt_client,
	}

	return result, result.RefreshParams()
}

func (engine *DefaultRpcAndTzktColletor) GetId() string {
	return "DefaultRpcAndTzktColletor"
}

func (engine *DefaultRpcAndTzktColletor) RefreshParams() error {
	failures := 0
	for _, rpc := range engine.rpcs {
		err := rpc.Init(context.Background())
		if err != nil {
			slog.Warn("failed to refresh rpc params", "error", err.Error())
			failures++
		}
	}
	if failures == len(engine.rpcs) {
		return fmt.Errorf("failed to refresh rpc params for all clients, all %d failed", failures)
	}

	return nil
}

func (engine *DefaultRpcAndTzktColletor) GetCurrentProtocol() (tezos.ProtocolHash, error) {
	params, err := utils.AttemptWithRpcClients(defaultCtx, engine.rpcs, func(client *rpc.Client) (*tezos.Params, error) {
		return client.GetParams(context.Background(), rpc.Head)
	})
	if err != nil {
		return tezos.ZeroProtocolHash, err
	}
	return params.Protocol, nil
}

func (engine *DefaultRpcAndTzktColletor) IsRevealed(addr tezos.Address) (bool, error) {
	state, err := utils.AttemptWithRpcClients(defaultCtx, engine.rpcs, func(client *rpc.Client) (*rpc.ContractInfo, error) {
		return client.GetContractExt(defaultCtx, addr, rpc.Head)
	})
	if err != nil {
		return false, err
	}
	return state.IsRevealed(), nil
}

func (engine *DefaultRpcAndTzktColletor) GetCurrentCycleNumber() (int64, error) {
	head, err := utils.AttemptWithRpcClients(defaultCtx, engine.rpcs, func(client *rpc.Client) (*rpc.Block, error) {
		return client.GetHeadBlock(defaultCtx)
	})
	if err != nil {
		return 0, err
	}

	return head.GetLevelInfo().Cycle, err
}

func (engine *DefaultRpcAndTzktColletor) GetLastCompletedCycle() (int64, error) {
	cycle, err := engine.GetCurrentCycleNumber()
	return cycle - 1, err
}

func (engine *DefaultRpcAndTzktColletor) GetCycleStakingData(baker tezos.Address, cycle int64) (*common.BakersCycleData, error) {
	return engine.tzkt.GetCycleData(context.Background(), baker, cycle)
}

func (engine *DefaultRpcAndTzktColletor) GetCyclesInDateRange(startDate time.Time, endDate time.Time) ([]int64, error) {
	return engine.tzkt.GetCyclesInDateRange(context.Background(), startDate, endDate)
}

func (engine *DefaultRpcAndTzktColletor) WasOperationApplied(op tezos.OpHash) (common.OperationStatus, error) {
	return engine.tzkt.WasOperationApplied(context.Background(), op)
}

func (engine *DefaultRpcAndTzktColletor) GetBranch(offset int64) (hash tezos.BlockHash, err error) {
	hash, err = utils.AttemptWithRpcClients(defaultCtx, engine.rpcs, func(client *rpc.Client) (tezos.BlockHash, error) {
		return client.GetBlockHash(context.Background(), rpc.NewBlockOffset(rpc.Head, offset))
	})
	return
}

func (engine *DefaultRpcAndTzktColletor) Simulate(o *codec.Op, publicKey tezos.Key) (rcpt *rpc.Receipt, err error) {
	params, err := utils.AttemptWithRpcClients(defaultCtx, engine.rpcs, func(client *rpc.Client) (*tezos.Params, error) {
		return client.GetParams(context.Background(), rpc.Head)
	})

	o = o.WithParams(params)
	for i := 0; i < 5; i++ {
		_, err = utils.AttemptWithRpcClients(defaultCtx, engine.rpcs, func(client *rpc.Client) (bool, error) {
			err := client.Complete(context.Background(), o, publicKey)
			if err != nil {
				return false, err
			}

			rcpt, err = client.Simulate(context.Background(), o, nil)
			if err != nil && rcpt == nil { // we do not retry on receipt errors
				slog.Debug("Internal simulate error - likely networking, retrying", "error", err.Error())
				// sleep 5s * i
				time.Sleep(time.Duration(i*5) * time.Second)
				return false, err
			}
			return true, nil
		})
		if err == nil {
			break
		}
	}
	return rcpt, err
}

func (engine *DefaultRpcAndTzktColletor) GetBalance(addr tezos.Address) (tezos.Z, error) {
	return utils.AttemptWithRpcClients(defaultCtx, engine.rpcs, func(client *rpc.Client) (tezos.Z, error) {
		return client.GetContractBalance(context.Background(), addr, rpc.Head)
	})
}

func (engine *DefaultRpcAndTzktColletor) CreateCycleMonitor(options common.CycleMonitorOptions) (common.CycleMonitor, error) {
	ctx := context.Background()
	monitor, err := utils.AttemptWithRpcClients(ctx, engine.rpcs, func(client *rpc.Client) (common.CycleMonitor, error) {
		return common.NewCycleMonitor(ctx, client, options)
	})
	if err != nil {
		return nil, err
	}
	utils.CallbackOnInterrupt(ctx, monitor.Cancel)
	slog.Info("tracking cycles... (cancel with Ctrl-C/SIGINT)\n\n")
	return monitor, nil
}

func (engine *DefaultRpcAndTzktColletor) SendAnalytics(bakerId string, version string) {
	go func() {
		body := fmt.Sprintf(`{"bakerId": "%s", "version": "%s"}`, bakerId, version)
		resp, err := http.Post("https://analytics.tez.capital/pay", "application/json", strings.NewReader(body))
		if err != nil {
			return
		}
		defer resp.Body.Close()
	}()
}
