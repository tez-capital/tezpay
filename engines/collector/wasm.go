//go:build js && wasm

package collector_engines

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/wasm"
	"github.com/samber/lo"
)

type JsCollector struct {
	collector js.Value
}

func InitJsColletor(collector js.Value) (*JsCollector, error) {
	if collector.Type() != js.TypeObject {
		return nil, fmt.Errorf("invalid collector object")
	}
	result := &JsCollector{
		collector: collector,
	}

	return result, result.RefreshParams()
}

func (engine *JsCollector) GetId() string {
	return "JsColletor"
}

func (engine *JsCollector) RefreshParams() error {
	funcId := "refreshParams"

	_, err := wasm.CallJsFunc(engine.collector, funcId)
	return err
}

func (engine *JsCollector) GetCurrentProtocol() (tezos.ProtocolHash, error) {
	funcId := "getCurrentProtocol"
	result, err := wasm.CallJsFuncExpectResultType(engine.collector, funcId, js.TypeString)
	if err != nil {
		return tezos.ZeroProtocolHash, err
	}
	return tezos.ParseProtocolHash(result.String())
}

func (engine *JsCollector) GetCurrentCycleNumber() (int64, error) {
	funcId := "getCurrentCycleNumber"
	result, err := wasm.CallJsFuncExpectResultType(engine.collector, funcId, js.TypeNumber)
	if err != nil {
		return 0, err
	}

	return int64(result.Int()), nil
}

func (engine *JsCollector) GetLastCompletedCycle() (int64, error) {
	cycle, err := engine.GetCurrentCycleNumber()
	return cycle - 1, err
}

type delegatorInfo struct {
	Address string `json:"address"`
	Balance int64  `json:"balance"`
	Emptied bool   `json:"emptied,omitempty"`
}

type bakersCycleData struct {
	StakingBalance           int64           `json:"stakingBalance"`
	DelegatedBalance         int64           `json:"delegatedBalance"`
	BlockRewards             int64           `json:"blockRewards"`
	MissedBlockRewards       int64           `json:"missedBlockRewards"`
	EndorsementRewards       int64           `json:"endorsementRewards"`
	MissedEndorsementRewards int64           `json:"missedEndorsementRewards"`
	NumDelegators            int32           `json:"numDelegators"`
	FrozenDepositLimit       int64           `json:"frozenDepositLimit"`
	BlockFees                int64           `json:"blockFees"`
	Delegators               []delegatorInfo `json:"delegators"`
}

func (engine *JsCollector) GetCycleData(baker tezos.Address, cycle int64) (*common.BakersCycleData, error) {
	funcId := "getCycleData"
	result, err := wasm.CallJsFuncExpectResultType(engine.collector, funcId, js.TypeString, baker.String(), cycle)
	if err != nil {
		return nil, err
	}

	var responseData bakersCycleData
	err = json.Unmarshal([]byte(result.String()), &responseData)
	if err != nil {
		return nil, err
	}

	data := &common.BakersCycleData{
		StakingBalance:          tezos.NewZ(responseData.StakingBalance),
		DelegatedBalance:        tezos.NewZ(responseData.DelegatedBalance),
		BlockRewards:            tezos.NewZ(responseData.BlockRewards),
		IdealBlockRewards:       tezos.NewZ(responseData.BlockRewards).Add64(responseData.MissedBlockRewards),
		EndorsementRewards:      tezos.NewZ(responseData.EndorsementRewards),
		IdealEndorsementRewards: tezos.NewZ(responseData.EndorsementRewards).Add64(responseData.MissedEndorsementRewards),
		NumDelegators:           responseData.NumDelegators,
		FrozenDepositLimit:      tezos.NewZ(responseData.FrozenDepositLimit),
		BlockFees:               tezos.NewZ(responseData.BlockFees),
		Delegators: lo.Map(responseData.Delegators, func(delegator delegatorInfo, _ int) common.Delegator {
			addr, err := tezos.ParseAddress(delegator.Address)
			if err != nil {
				panic(err)
			}
			return common.Delegator{
				Address: addr,
				Balance: tezos.NewZ(delegator.Balance),
				Emptied: delegator.Emptied,
			}
		}),
	}
	return data, nil
}

func (engine *JsCollector) WasOperationApplied(op tezos.OpHash) (common.OperationStatus, error) {
	funcId := "wasOperationApplied"
	result, err := wasm.CallJsFuncExpectResultType(engine.collector, funcId, js.TypeString, op.String())
	if err != nil {
		return common.OPERATION_STATUS_UNKNOWN, err
	}

	return common.OperationStatus(result.String()), nil
}

func (engine *JsCollector) GetBranch(offset int64) (hash tezos.BlockHash, err error) {
	funcId := "getBranch"
	result, err := wasm.CallJsFuncExpectResultType(engine.collector, funcId, js.TypeString, offset)
	if err != nil {
		return tezos.ZeroBlockHash, err
	}
	return tezos.ParseBlockHash(result.String())
}

func (engine *JsCollector) Simulate(o *codec.Op, publicKey tezos.Key) (*rpc.Receipt, error) {
	funcId := "simulate"
	opJson, err := o.MarshalJSON()
	if err != nil {
		return nil, err
	}

	result, err := wasm.CallJsFuncExpectResultType(engine.collector, funcId, js.TypeString, string(opJson), publicKey.String())
	if err != nil {
		return nil, err
	}

	var operation rpc.Operation
	err = json.Unmarshal([]byte(result.String()), &operation)
	if err != nil {
		return nil, err
	}
	return &rpc.Receipt{
		Op: &operation,
	}, nil
}

func (engine *JsCollector) GetBalance(addr tezos.Address) (tezos.Z, error) {
	funcId := "getBalance"
	result, err := wasm.CallJsFuncExpectResultType(engine.collector, funcId, js.TypeNumber, addr.String())
	if err != nil {
		return tezos.Zero, err
	}

	return tezos.ParseZ(result.String())
}

type JsCycleMonitor struct {
	monitor js.Value
}

func (monitor *JsCycleMonitor) Cancel() {
	funcId := "cancel"
	wasm.CallJsFunc(monitor.monitor, funcId)
}

func (monitor *JsCycleMonitor) WaitForNextCompletedCycle(lastProcessedCycle int64) (int64, error) {
	funcId := "waitForNextCompletedCycle"

	result, err := wasm.CallJsFuncExpectResultType(monitor.monitor, funcId, js.TypeNumber, lastProcessedCycle)
	if err != nil {
		return 0, err
	}

	return int64(result.Int()), nil
}

func (engine *JsCollector) CreateCycleMonitor(options common.CycleMonitorOptions) (common.CycleMonitor, error) {
	funcId := "createCycleMonitor"
	optionsJson, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}
	result, err := wasm.CallJsFuncExpectResultType(engine.collector, funcId, js.TypeObject, optionsJson)
	if err != nil {
		return nil, err
	}

	return &JsCycleMonitor{
		monitor: result,
	}, nil
}

func (engine *JsCollector) SendAnalytics(bakerId string, version string) {
	go func() {
		funcId := "sendAnalytics"

		body := fmt.Sprintf(`{"bakerId": "%s", "version": "%s"}`, bakerId, version)
		wasm.CallJsFunc(engine.collector, funcId, body)
	}()
}
