package mock

import (
	"time"

	"github.com/tez-capital/tezpay/common"
	"github.com/trilitech/tzgo/codec"
	"github.com/trilitech/tzgo/rpc"
	"github.com/trilitech/tzgo/tezos"
)

type EmptyCollector struct {
}

func (engine *EmptyCollector) GetId() string {
	panic("not implemented")
}

func (engine *EmptyCollector) RefreshParams() error {
	panic("not implemented")
}

func (engine *EmptyCollector) IsRevealed(address tezos.Address) (bool, error) {
	panic("not implemented")
}

func (engine *EmptyCollector) GetCurrentCycleNumber() (int64, error) {
	panic("not implemented")
}

func (engine *EmptyCollector) GetLastCompletedCycle() (int64, error) {
	panic("not implemented")
}

func (engine *EmptyCollector) GetCycleStakingData(baker tezos.Address, cycle int64) (*common.BakersCycleData, error) {
	panic("not implemented")
}

func (engine *EmptyCollector) GetCyclesInDateRange(startDate time.Time, endDate time.Time) ([]int64, error) {
	panic("not implemented")
}

func (engine *EmptyCollector) WasOperationApplied(op tezos.OpHash) (common.OperationStatus, error) {
	panic("not implemented")
}

func (engine *EmptyCollector) CreateCycleMonitor(options common.CycleMonitorOptions) (common.CycleMonitor, error) {
	panic("not implemented")
}

func (engine *EmptyCollector) GetBranch(offset int64) (hash tezos.BlockHash, err error) {
	panic("not implemented")
}

func (engine *EmptyCollector) GetExpectedTxCosts() int64 {
	panic("not implemented")
}

func (engine *EmptyCollector) Simulate(o *codec.Op, publicKey tezos.Key) (*rpc.Receipt, error) {
	panic("not implemented")
}

func (engine *EmptyCollector) GetBalance(addr tezos.Address) (tezos.Z, error) {
	panic("not implemented")
}

func (engine *EmptyCollector) SendAnalytics(bakerId string, version string) {}

func (engine *EmptyCollector) GetCurrentProtocol() (tezos.ProtocolHash, error) {
	panic("not implemented")
}
