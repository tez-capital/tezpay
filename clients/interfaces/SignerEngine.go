package interfaces

import (
	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/tezos"
)

type SignerEngine interface {
	GetId() string
	Sign(op *codec.Op) error
	GetPKH() tezos.Address
	GetKey() tezos.Key
}
