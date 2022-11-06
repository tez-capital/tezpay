package clients

import (
	"fmt"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/tezos"
)

type InMemorySigner struct {
	Key tezos.PrivateKey
}

func InitInMemorySigner(key string) (*InMemorySigner, error) {
	tkey, err := tezos.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("invalid key '%s' - %s", key, err.Error())
	}
	return &InMemorySigner{
		Key: tkey,
	}, nil
}

func (signer *InMemorySigner) GetId() string {
	return "InMemorySigner"
}

func (signer *InMemorySigner) GetPKH() tezos.Address {
	return signer.Key.Address()
}

func (signer *InMemorySigner) GetKey() tezos.Key {
	return signer.Key.Public()
}

func (signer *InMemorySigner) Sign(op *codec.Op) error {
	if err := op.Sign(signer.Key); err != nil {
		return err
	}
	return nil
}
