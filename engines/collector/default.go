package collector_engines

import (
	"context"
	"net/http"
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
	chainId, err := rpcClient.GetChainId(context.Background())
	if err != nil {
		return nil, err
	}
	rpcClient.ChainId = chainId
	tzktClient, err := tzkt.InitClient(config.Network.TzktUrl, &client)
	if err != nil {
		return nil, err
	}

	return &DefaultRpcAndTzktColletor{
		rpc:  rpcClient,
		tzkt: tzktClient,
	}, nil
}

func (engine *DefaultRpcAndTzktColletor) GetId() string {
	return "DefaultRpcAndTzktColletor"
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

func (engine *DefaultRpcAndTzktColletor) GetCycleData(baker tezos.Address, cycle int64) (*common.BakersCycleData, error) {
	return engine.tzkt.GetCycleData(context.Background(), baker, cycle)
}

func (engine *DefaultRpcAndTzktColletor) WasOperationApplied(op tezos.OpHash) (common.OperationStatus, error) {
	return engine.tzkt.WasOperationApplied(context.Background(), op)
}

func (engine *DefaultRpcAndTzktColletor) GetBranch(offset int64) (hash tezos.BlockHash, err error) {
	hash, err = engine.rpc.GetBlockHash(context.Background(), rpc.NewBlockOffset(rpc.Head, offset))
	return
}

func (engine *DefaultRpcAndTzktColletor) Simulate(o *codec.Op, publicKey tezos.Key) (*rpc.Receipt, error) {
	err := engine.rpc.Complete(context.Background(), o, publicKey)
	if err != nil {
		return nil, err
	}
	return engine.rpc.Simulate(context.Background(), o, nil)
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
