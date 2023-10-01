//go:build js && wasm

package transactor_engines

import (
	"encoding/hex"
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

	var jsLimits common.OperationLimits
	err = json.Unmarshal([]byte(result.String()), &jsLimits)
	if err != nil {
		return nil, err
	}

	return &jsLimits, nil
}

type JsTezosParams struct {
	tezos.Params
	BlockLevel int64 `json:"block_level"`
}

func (engine *JsTransactor) getParams() (*tezos.Params, error) {
	funcId := "getChainParams"

	paramsJson, err := wasm.CallJsFuncExpectResultType(engine.transactor, funcId, js.TypeString)
	if err != nil {
		return nil, err
	}

	var jsParams JsTezosParams
	err = json.Unmarshal([]byte(paramsJson.String()), &jsParams)
	if err != nil {
		return nil, err
	}
	params := jsParams.WithChainId(jsParams.ChainId).WithProtocol(jsParams.Protocol).WithBlock(jsParams.BlockLevel)
	return params, nil
}

type JsTezosOperationContextExtra struct {
	ChainId *tezos.ChainIdHash `json:"chain_id"` // optional, used for remote signing only
	TTL     int64              `json:"ttl"`      // optional, specify TTL in blocks
	Params  *tezos.Params      `json:"params"`   // optional, define protocol to encode for
	Source  tezos.Address      `json:"source"`   // optional, used as manager/sender
}

type JsTezosOperationContext struct {
	Operation *codec.Op                    `json:"operation"`
	Extra     JsTezosOperationContextExtra `json:"extra"`
}

func (op *JsTezosOperationContext) ToOp() *codec.Op {
	op.Operation.ChainId = op.Extra.ChainId
	op.Operation.TTL = op.Extra.TTL
	op.Operation.Params = op.Extra.Params
	op.Operation.Source = op.Extra.Source
	return op.Operation
}

func (engine *JsTransactor) Complete(op *codec.Op, key tezos.Key) (*codec.Op, error) {
	funcId := "complete"

	params, err := engine.getParams()
	if err != nil {
		return nil, err
	}

	op = op.WithParams(params)
	operation := JsTezosOperationContext{op, JsTezosOperationContextExtra{op.ChainId, op.TTL, op.Params, op.Source}}
	operationJson, err := json.Marshal(operation)
	if err != nil {
		return nil, err
	}
	jsResult, err := wasm.CallJsFuncExpectResultType(engine.transactor, funcId, js.TypeString, string(operationJson), key.String())
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(jsResult.String()), &operation)
	if err != nil {
		return nil, err
	}

	result := operation.ToOp()
	return result, err
}

func (engine *JsTransactor) Dispatch(op *codec.Op, opts *common.DispatchOptions) (common.OpResult, error) {
	funcId := "dispatch"

	if opts == nil {
		opts = &common.DispatchOptions{
			TTL:           tezos.DefaultParams.MaxOperationsTTL - 2,
			Confirmations: 2,
		}
	}

	opBytes := op.Bytes()
	opHex := hex.EncodeToString(opBytes)

	optsJson, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}

	result, err := wasm.CallJsFuncExpectResultType(engine.transactor, funcId, js.TypeObject, opHex, string(optsJson))
	return &JsTransactorOpResult{jsResult: result}, err
}
