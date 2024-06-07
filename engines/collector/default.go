package collector_engines

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/configuration"
	"github.com/tez-capital/tezpay/engines/tzkt"
	"github.com/tez-capital/tezpay/utils"
	"github.com/trilitech/tzgo/codec"
	"github.com/trilitech/tzgo/rpc"
	"github.com/trilitech/tzgo/tezos"
)

type DefaultRpcAndTzktColletor struct {
	rpc  *rpc.Client
	tzkt *tzkt.Client
}

var (
	defaultCtx context.Context = context.Background()
)

func InitDefaultRpcAndTzktColletor(config *configuration.RuntimeConfiguration) (*DefaultRpcAndTzktColletor, error) {
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	rpcClient, err := rpc.NewClient(config.Network.RpcUrl, &client)
	if err != nil {
		return nil, err
	}

	tzktClient, err := tzkt.InitClient(config.Network.TzktUrl, &client)
	if err != nil {
		return nil, err
	}

	result := &DefaultRpcAndTzktColletor{
		rpc:  rpcClient,
		tzkt: tzktClient,
	}

	return result, result.RefreshParams()
}

func (engine *DefaultRpcAndTzktColletor) GetId() string {
	return "DefaultRpcAndTzktColletor"
}

func (engine *DefaultRpcAndTzktColletor) RefreshParams() error {
	return engine.rpc.Init(context.Background())
}

func (engine *DefaultRpcAndTzktColletor) GetCurrentProtocol() (tezos.ProtocolHash, error) {
	params, err := engine.rpc.GetParams(context.Background(), rpc.Head)

	if err != nil {
		return tezos.ZeroProtocolHash, err
	}
	return params.Protocol, nil
}

func (engine *DefaultRpcAndTzktColletor) IsRevealed(addr tezos.Address) (bool, error) {
	state, err := engine.rpc.GetContractExt(defaultCtx, addr, rpc.Head)
	if err != nil {
		return false, err
	}
	return state.IsRevealed(), nil
}

func (engine *DefaultRpcAndTzktColletor) GetCurrentCycleNumber() (int64, error) {
	head, err := engine.rpc.GetHeadBlock(defaultCtx)
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
	hash, err = engine.rpc.GetBlockHash(context.Background(), rpc.NewBlockOffset(rpc.Head, offset))
	return
}

func (engine *DefaultRpcAndTzktColletor) Simulate(o *codec.Op, publicKey tezos.Key) (rcpt *rpc.Receipt, err error) {
	o = o.WithParams(engine.rpc.Params)
	for i := 0; i < 5; i++ {
		err = engine.rpc.Complete(context.Background(), o, publicKey)
		if err != nil {
			continue
		}

		rcpt, err = engine.rpc.Simulate(context.Background(), o, nil)
		if err != nil && rcpt == nil { // we do not retry on receipt errors
			log.Debug("Internal simulate error - likely networking, retrying: ", err)
			// sleep 5s * i
			time.Sleep(time.Duration(i*5) * time.Second)
			continue
		}
		break
	}
	return rcpt, err
}

func (engine *DefaultRpcAndTzktColletor) GetBalance(addr tezos.Address) (tezos.Z, error) {
	return engine.rpc.GetContractBalance(context.Background(), addr, rpc.Head)
}

func (engine *DefaultRpcAndTzktColletor) CreateCycleMonitor(options common.CycleMonitorOptions) (common.CycleMonitor, error) {
	ctx := context.Background()
	monitor, err := common.NewCycleMonitor(ctx, engine.rpc, options)
	if err != nil {
		return nil, err
	}
	utils.CallbackOnInterrupt(ctx, monitor.Cancel)
	log.Info("tracking cycles... (cancel with Ctrl-C/SIGINT)\n\n")

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
