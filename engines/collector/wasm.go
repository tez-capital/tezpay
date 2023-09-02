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
	"github.com/alis-is/tezpay/utils"
)

type DefaultRpcAndTzktColletor struct {
	collector js.Value
}

func InitJsColletor(collector js.Value) (*DefaultRpcAndTzktColletor, error) {
	if collector.Type() != js.TypeObject {
		return nil, fmt.Errorf("invalid collector object")
	}
	result := &DefaultRpcAndTzktColletor{
		collector: collector,
	}

	return result, result.RefreshParams()
}

func (engine *DefaultRpcAndTzktColletor) GetId() string {
	return "DefaultRpcAndTzktColletor"
}

func (engine *DefaultRpcAndTzktColletor) RefreshParams() error {
	funcId := "refreshParams"
	if !utils.HasJsFunc(engine.collector, funcId) {
		return fmt.Errorf("function %s not found", funcId)
	}

	_ = engine.collector.Call(funcId)
	return nil
}

func (engine *DefaultRpcAndTzktColletor) GetCurrentProtocol() (tezos.ProtocolHash, error) {
	funcId := "getCurrentProtocol"
	if !utils.HasJsFunc(engine.collector, funcId) {
		return tezos.ZeroProtocolHash, fmt.Errorf("function %s not found", funcId)
	}

	result := engine.collector.Call(funcId)
	if result.Type() != js.TypeString {
		return tezos.ZeroProtocolHash, fmt.Errorf("%s returned invalid data", funcId)
	}

	return tezos.ParseProtocolHash(result.String())
}

func (engine *DefaultRpcAndTzktColletor) GetCurrentCycleNumber() (int64, error) {
	funcId := "getCurrentCycleNumber"
	if !utils.HasJsFunc(engine.collector, funcId) {
		return 0, fmt.Errorf("function %s not found", funcId)
	}

	result := engine.collector.Call(funcId)
	if result.Type() != js.TypeNumber {
		return 0, fmt.Errorf("%s returned invalid data", funcId)
	}

	return int64(result.Int()), nil
}

func (engine *DefaultRpcAndTzktColletor) GetLastCompletedCycle() (int64, error) {
	cycle, err := engine.GetCurrentCycleNumber()
	return cycle - 1, err
}

func (engine *DefaultRpcAndTzktColletor) GetCycleData(baker tezos.Address, cycle int64) (*common.BakersCycleData, error) {
	funcId := "getCycleData"
	if !utils.HasJsFunc(engine.collector, funcId) {
		return nil, fmt.Errorf("function %s not found", funcId)
	}

	result := engine.collector.Call(funcId, cycle)
	if result.Type() != js.TypeString {
		return nil, fmt.Errorf("%s returned invalid data", funcId)
	}

	var data common.BakersCycleData
	err := json.Unmarshal([]byte(result.String()), &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (engine *DefaultRpcAndTzktColletor) WasOperationApplied(op tezos.OpHash) (common.OperationStatus, error) {
	funcId := "wasOperationApplied"
	if !utils.HasJsFunc(engine.collector, funcId) {
		return common.OPERATION_STATUS_UNKNOWN, fmt.Errorf("function %s not found", funcId)
	}

	result := engine.collector.Call(funcId, op.String())
	if result.Type() != js.TypeString {
		return common.OPERATION_STATUS_UNKNOWN, fmt.Errorf("%s returned invalid data", funcId)
	}

	return common.OperationStatus(result.String()), nil
}

func (engine *DefaultRpcAndTzktColletor) GetBranch(offset int64) (hash tezos.BlockHash, err error) {
	funcId := "getBranch"
	if !utils.HasJsFunc(engine.collector, funcId) {
		return tezos.ZeroBlockHash, fmt.Errorf("function %s not found", funcId)
	}

	result := engine.collector.Call(funcId, offset)
	if result.Type() != js.TypeString {
		return tezos.ZeroBlockHash, fmt.Errorf("%s returned invalid data", funcId)
	}
	return tezos.ParseBlockHash(result.String())
}

func (engine *DefaultRpcAndTzktColletor) Simulate(o *codec.Op, publicKey tezos.Key) (*rpc.Receipt, error) {
	funcId := "simulate"
	if !utils.HasJsFunc(engine.collector, funcId) {
		return nil, fmt.Errorf("function %s not found", funcId)
	}

	opJson, err := o.MarshalJSON()
	if err != nil {
		return nil, err
	}

	result := engine.collector.Call(funcId, string(opJson), publicKey.String())
	if result.Type() != js.TypeString {
		return nil, fmt.Errorf("%s returned invalid data", funcId)
	}

	var receipt rpc.Receipt
	err = json.Unmarshal([]byte(result.String()), &receipt)
	if err != nil {
		return nil, err
	}
	return &receipt, nil
}

func (engine *DefaultRpcAndTzktColletor) GetBalance(addr tezos.Address) (tezos.Z, error) {
	funcId := "getBalance"
	if !utils.HasJsFunc(engine.collector, funcId) {
		return tezos.Zero, fmt.Errorf("function %s not found", funcId)
	}

	result := engine.collector.Call(funcId, addr.String())
	if result.Type() != js.TypeNumber {
		return tezos.Zero, fmt.Errorf("%s returned invalid data", funcId)
	}

	return tezos.ParseZ(result.String())
}

type JsCycleMonitor struct {
	monitor js.Value
}

func (monitor *JsCycleMonitor) Cancel() {
	funcId := "cancel"
	if !utils.HasJsFunc(monitor.monitor, funcId) {
		return
	}

	monitor.monitor.Call(funcId)
}

func (monitor *JsCycleMonitor) WaitForNextCompletedCycle(lastProcessedCycle int64) (int64, error) {
	funcId := "waitForNextCompletedCycle"
	if !utils.HasJsFunc(monitor.monitor, funcId) {
		return 0, fmt.Errorf("function %s not found", funcId)
	}

	result := monitor.monitor.Call(funcId, lastProcessedCycle)
	if result.Type() != js.TypeNumber {
		return 0, fmt.Errorf("%s returned invalid data", funcId)
	}
	return int64(result.Int()), nil
}

func (engine *DefaultRpcAndTzktColletor) CreateCycleMonitor(options common.CycleMonitorOptions) (common.CycleMonitor, error) {
	funcId := "createCycleMonitor"
	if !utils.HasJsFunc(engine.collector, funcId) {
		return nil, fmt.Errorf("function %s not found", funcId)
	}

	optionsJson, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}
	result := engine.collector.Call(funcId, optionsJson)
	if result.Type() != js.TypeObject {
		return nil, fmt.Errorf("%s returned invalid data", funcId)
	}

	return &JsCycleMonitor{
		monitor: result,
	}, nil
}

func (engine *DefaultRpcAndTzktColletor) SendAnalytics(bakerId string, version string) {
	go func() {
		funcId := "sendAnalytics"
		if !utils.HasJsFunc(engine.collector, funcId) {
			return
		}

		body := fmt.Sprintf(`{"bakerId": "%s", "version": "%s"}`, bakerId, version)
		engine.collector.Call(funcId, body)
	}()
}
