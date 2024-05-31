package mock

import "github.com/trilitech/tzgo/tezos"

func GetRandomAddress() tezos.Address {
	k, _ := tezos.GenerateKey(tezos.KeyTypeEd25519)
	return k.Address()
}
