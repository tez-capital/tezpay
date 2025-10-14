package mock

import (
	signer_engines "github.com/tez-capital/tezpay/engines/signer"
	"github.com/trilitech/tzgo/tezos"
)

func InitSimpleSigner() *signer_engines.InMemorySigner {
	key, _ := tezos.GenerateKey(tezos.KeyTypeEd25519)
	encoded := key.String()

	result, _ := signer_engines.InitInMemorySigner(encoded)
	return result
}
