//go:build js && wasm

package signer_engines

import (
	"fmt"
	"syscall/js"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/utils"
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
	if !utils.HasJsFunc(jsSigner.signer, funcId) {
		return tezos.ZeroAddress
	}

	result := jsSigner.signer.Call(funcId)
	if result.Type() != js.TypeString {
		return tezos.ZeroAddress
	}

	return tezos.MustParseAddress(result.String())
}

func (jsSigner *JsSigner) GetKey() tezos.Key {
	funcId := "getKey"
	if !utils.HasJsFunc(jsSigner.signer, funcId) {
		return tezos.InvalidKey
	}

	result := jsSigner.signer.Call(funcId)
	if result.Type() != js.TypeString {
		return tezos.InvalidKey
	}

	return tezos.MustParseKey(result.String())
}

func (jsSigner *JsSigner) Sign(op *codec.Op) error {
	funcId := "sign"
	if !utils.HasJsFunc(jsSigner.signer, funcId) {
		return nil
	}

	result := jsSigner.signer.Call(funcId, op.Digest())
	if result.Type() != js.TypeString {
		return nil
	}

	var err error
	op.Signature, err = tezos.ParseSignature(result.String())
	return err
}
