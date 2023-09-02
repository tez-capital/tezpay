//go:build js && wasm

package transactor_engines

import (
	"encoding/json"
	"errors"
	"fmt"
	"syscall/js"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/utils"
)

type JsTransactor struct {
	transactor js.Value
}

type JsTransactorOpResult struct {
	jsResult js.Value
}

func (result *JsTransactorOpResult) GetOpHash() tezos.OpHash {
	opHash := result.jsResult.Get("opHash")
	if opHash.Type() != js.TypeString {
		return tezos.ZeroOpHash
	}

	return tezos.MustParseOpHash(opHash.String())
}

func (result *JsTransactorOpResult) WaitForApply() error {
	funcId := "waitForApply"
	if !utils.HasJsFunc(result.jsResult, funcId) {
		return errors.New("function waitForApply not found")
	}

	res := result.jsResult.Call(funcId)
	if res.Type() == js.TypeString {
		return errors.New(res.String())
	}
	return nil
}
func InitJsTransactor(transactor js.Value) (*JsTransactor, error) {
	if transactor.Type() != js.TypeObject {
		return nil, fmt.Errorf("invalid collector object")
	}
	result := &JsTransactor{
		transactor: transactor,
	}

	return result, result.RefreshParams()
}

func (transactor *JsTransactor) GetId() string {
	return "JsTransactor"
}

func (engine *JsTransactor) RefreshParams() error {
	funcId := "refreshParams"
	if !utils.HasJsFunc(engine.transactor, funcId) {
		return fmt.Errorf("function %s not found", funcId)
	}

	_ = engine.transactor.Call(funcId)
	return nil
}

func (engine *JsTransactor) GetLimits() (*common.OperationLimits, error) {
	funcId := "getLimits"
	if !utils.HasJsFunc(engine.transactor, funcId) {
		return nil, fmt.Errorf("function %s not found", funcId)
	}

	result := engine.transactor.Call(funcId)
	if result.Type() != js.TypeString {
		return nil, fmt.Errorf("%s returned invalid data", funcId)
	}

	var limits common.OperationLimits
	err := json.Unmarshal([]byte(result.String()), &limits)
	if err != nil {
		return nil, err
	}
	return &limits, nil
}

func (engine *JsTransactor) Complete(op *codec.Op, key tezos.Key) error {
	funcId := "getChainParams"
	if !utils.HasJsFunc(engine.transactor, funcId) {
		return fmt.Errorf("function %s not found", funcId)
	}

	paramsJson := engine.transactor.Call(funcId)
	if paramsJson.Type() != js.TypeString {
		return fmt.Errorf("%s returned invalid data", funcId)
	}

	var params tezos.Params
	err := json.Unmarshal([]byte(paramsJson.String()), &params)
	if err != nil {
		return err
	}

	op = op.WithParams(&params)

	// TODO: counter and branch

	return err
}

func (engine *JsTransactor) Dispatch(op *codec.Op, opts *common.DispatchOptions) (common.OpResult, error) {
	funcId := "dispatch"
	if !utils.HasJsFunc(engine.transactor, funcId) {
		return nil, fmt.Errorf("function %s not found", funcId)
	}

	if opts == nil {
		opts = &common.DispatchOptions{
			TTL:           tezos.DefaultParams.MaxOperationsTTL - 2,
			Confirmations: 2,
		}
	}

	opJson, err := op.MarshalJSON()
	if err != nil {
		return nil, err
	}

	optsJson, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	result := engine.transactor.Call(funcId, string(opJson), string(optsJson))
	if result.Type() != js.TypeObject {
		return nil, fmt.Errorf("%s returned invalid data", funcId)
	}
	return &JsTransactorOpResult{jsResult: result}, nil
}
