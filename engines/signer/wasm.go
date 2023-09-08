//go:build js && wasm

package signer_engines

import (
	"fmt"
	"syscall/js"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/wasm"
)

type JsSigner struct {
	signer js.Value
}

func InitJsSigner(signer js.Value) (*JsSigner, error) {
	if signer.Type() != js.TypeObject {
		return nil, fmt.Errorf("invalid signer object")
	}
	return &JsSigner{
		signer,
	}, nil
}

func (jsSigner *JsSigner) GetId() string {
	return "JsSigner"
}

func (jsSigner *JsSigner) GetPKH() tezos.Address {
	funcId := "getPKH"

	result, err := wasm.CallJsFuncExpectResultType(jsSigner.signer, funcId, js.TypeString)
	if err != nil {
		return tezos.ZeroAddress
	}

	return tezos.MustParseAddress(result.String())
}

func (jsSigner *JsSigner) GetKey() tezos.Key {
	funcId := "getKey"

	result, err := wasm.CallJsFuncExpectResultType(jsSigner.signer, funcId, js.TypeString)
	if err != nil {
		return tezos.InvalidKey
	}

	return tezos.MustParseKey(result.String())
}

func (jsSigner *JsSigner) Sign(op *codec.Op) error {
	funcId := "sign"

	result, err := wasm.CallJsFuncExpectResultType(jsSigner.signer, funcId, js.TypeString, op.Digest())

	op.Signature, err = tezos.ParseSignature(result.String())
	return err
}
