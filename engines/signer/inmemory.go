package signer_engines

import (
	"errors"

	"github.com/alis-is/tezpay/constants"
	"github.com/trilitech/tzgo/codec"
	"github.com/trilitech/tzgo/signer"
	"github.com/trilitech/tzgo/tezos"
)

type InMemorySigner struct {
	Key tezos.PrivateKey
}

func InitInMemorySigner(key string) (*InMemorySigner, error) {
	tkey, err := tezos.ParsePrivateKey(key)
	if err != nil {
		return nil, errors.Join(constants.ErrSignerLoadFailed, err)
	}
	return &InMemorySigner{
		Key: tkey,
	}, nil
}

func (inMemSigner *InMemorySigner) GetId() string {
	return "InMemorySigner"
}

func (inMemSigner *InMemorySigner) GetPKH() tezos.Address {
	return inMemSigner.Key.Address()
}

func (inMemSigner *InMemorySigner) GetKey() tezos.Key {
	return inMemSigner.Key.Public()
}

func (inMemSigner *InMemorySigner) Sign(op *codec.Op) error {
	if err := op.Sign(inMemSigner.Key); err != nil {
		return err
	}
	return nil
}

func (inMemSigner *InMemorySigner) GetSigner() signer.Signer {
	return signer.NewFromKey(inMemSigner.Key)
}
