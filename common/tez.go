package common

import (
	"fmt"

	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/trilitech/tzgo/tezos"
)

func FormatAmount(kind enums.EPayoutTransactionKind, amount int64) string {
	if amount == 0 {
		return ""
	}
	switch kind {
	case enums.PAYOUT_TX_KIND_FA1_2:
		return fmt.Sprintf("%d FA1", amount)
	case enums.PAYOUT_TX_KIND_FA2:
		return fmt.Sprintf("%d FA2", amount)
	default:
		return MutezToTezS(amount)
	}
}

func MutezToTezS(amount int64) string {
	if amount == 0 {
		return ""
	}
	tez := float64(amount) / constants.MUTEZ_FACTOR
	return fmt.Sprintf("%f TEZ", tez)
}

func FloatToPercentage(f float64) string {
	if f == 0 {
		return ""
	}
	return fmt.Sprintf("%.2f%%", f*100)
}

func ShortenAddress(taddr tezos.Address) string {
	if taddr.Equal(tezos.ZeroAddress) || taddr.Equal(tezos.InvalidAddress) {
		return ""
	}
	addr := taddr.String()
	total := len(addr)
	if total <= 13 {
		return addr
	}
	return fmt.Sprintf("%s...%s", addr[:5], addr[total-5:])
}

func ToStringEmptyIfZero[T comparable](value T) string {
	var zero T
	if value == zero {
		return ""
	}
	return fmt.Sprintf("%v", value)
}
