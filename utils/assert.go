package utils

import "github.com/trilitech/tzgo/tezos"

func AssertZAmountPositiveOrZero(amount tezos.Z) {
	if amount.IsNeg() {
		panic("amount is negative, this should never happen")
	}
}
