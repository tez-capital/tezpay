package common

import (
	"fmt"
	"math"

	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/trilitech/tzgo/tezos"
)

func FormatTezAmount(amount int64) string {
	if amount == 0 {
		return ""
	}
	return MutezToTezS(amount)
}

func FormatTokenAmount(kind enums.EPayoutTransactionKind, amount int64, alias string, decimals int) string {
	if amount == 0 {
		return ""
	}
	switch kind {
	case enums.PAYOUT_TX_KIND_FA1_2:
		amountFloat := float64(amount) / math.Pow10(decimals)
		if alias != "" {
			return fmt.Sprintf("%.*f %s", decimals, amountFloat, alias)
		}
		return fmt.Sprintf("%.*f FA1", decimals, amountFloat)
	case enums.PAYOUT_TX_KIND_FA2:
		amountFloat := float64(amount) / math.Pow10(decimals)
		if alias != "" {
			return fmt.Sprintf("%.*f %s", decimals, amountFloat, alias)
		}
		return fmt.Sprintf("%.*f FA2", decimals, amountFloat)
	default:
		return FormatTezAmount(amount)
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
