//go:build js && wasm

package transactor_engines

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/wasm"
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
	_, err := wasm.CallJsFunc(result.jsResult, funcId)
	return err
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

	_, err := wasm.CallJsFunc(engine.transactor, funcId)
	return err
}

func (engine *JsTransactor) GetLimits() (*common.OperationLimits, error) {
	funcId := "getLimits"

	result, err := wasm.CallJsFuncExpectResultType(engine.transactor, funcId, js.TypeString)
	if err != nil {
		return nil, err
	}

	var limits common.OperationLimits
	err = json.Unmarshal([]byte(result.String()), &limits)
	if err != nil {
		return nil, err
	}
	return &limits, nil
}

type JsTezosParams struct {
	tezos.Params
	BlockLevel int64 `json:"block_level"`
}

func (engine *JsTransactor) Complete(op *codec.Op, key tezos.Key) error {
	funcId := "getChainParams"

	paramsJson, err := wasm.CallJsFuncExpectResultType(engine.transactor, funcId, js.TypeString)
	if err != nil {
		return err
	}

	var jsParams JsTezosParams
	err = json.Unmarshal([]byte(paramsJson.String()), &jsParams)
	if err != nil {
		return err
	}
	params := jsParams.WithProtocol(jsParams.Protocol).WithBlock(jsParams.BlockLevel)

	// TODO: counter and branch

	op = op.WithParams(params)

	return err
}

func (engine *JsTransactor) Dispatch(op *codec.Op, opts *common.DispatchOptions) (common.OpResult, error) {
	funcId := "dispatch"

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

	result, err := wasm.CallJsFuncExpectResultType(engine.transactor, funcId, js.TypeObject, string(opJson), string(optsJson))
	return &JsTransactorOpResult{jsResult: result}, err
}
